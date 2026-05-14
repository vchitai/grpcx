# grpcx

A production-ready Go framework for building gRPC + REST services.

grpcx bundles the boilerplate every gRPC service needs — server lifecycle, structured errors, i18n, request tracing, auth, logging — into a small, composable library. Clone [`go-grpc-starter`](https://github.com/vchitai/go-grpc-starter) to see a working service built on it.

## Features

- **One-call server** — start gRPC + grpc-gateway HTTP in one call with graceful shutdown
- **Structured errors** — machine-readable codes, metadata, and automatic HTTP mapping
- **i18n** — localize error messages from YAML files with Go template interpolation
- **Auth** — JWT (HMAC-SHA256) and API key interceptors, claims in context
- **Observability** — Prometheus metrics, structured request logging, request IDs
- **Config** — generic YAML + env-var loader (`__` separator, e.g. `PG__HOST`)
- **Utilities** — paging/ordering helpers, nanoid generator, generic collection functions

## Installation

```bash
go get github.com/vchitai/grpcx
```

Requires Go 1.22+.

## Quick Start

```go
package main

import (
    "log/slog"

    "github.com/vchitai/grpcx/grpc/middleware"
    "github.com/vchitai/grpcx/logger"
    pkgserver "github.com/vchitai/grpcx/server"
)

func main() {
    l := logger.New()
    slog.SetDefault(l)

    srv, err := pkgserver.New(
        pkgserver.WithGrpcAddr("0.0.0.0", 10443),
        pkgserver.WithGatewayAddr("0.0.0.0", 10080),
        pkgserver.WithGrpcServerUnaryInterceptors(
            middleware.ErrorWrapperUnaryServerInterceptor(),
            middleware.RequestIDUnaryServerInterceptor(),
        ),
        pkgserver.WithServiceServer(myService),
    )
    if err != nil {
        l.Error("failed to create server", "err", err)
        return
    }
    if err := srv.Serve(); err != nil {
        l.Error("server error", "err", err)
    }
}
```

## Package Overview

| Package | Import path | Purpose |
|---|---|---|
| `server` | `grpcx/server` | gRPC + HTTP gateway lifecycle |
| `errs` | `grpcx/errs` | Structured, gRPC-aware errors |
| `config` | `grpcx/config` | Generic YAML + env config loader |
| `i18n` | `grpcx/i18n` | Server-side localization |
| `auth` | `grpcx/auth` | JWT and API key authentication |
| `logger` | `grpcx/logger` | slog setup and context helpers |
| `rpcctx` | `grpcx/rpcctx` | gRPC metadata helpers |
| `query` | `grpcx/query` | Paging and ordering for list queries |
| `id` | `grpcx/id` | URL-safe nanoid generator |
| `collection` | `grpcx/collection` | Generic slice and map utilities |
| `grpc/middleware` | `grpcx/grpc/middleware` | gRPC server interceptors |
| `grpc/gatewayopt` | `grpcx/grpc/gatewayopt` | grpc-gateway mux options |
| `grpc/protojson` | `grpcx/grpc/protojson` | JSON marshaling for grpc-gateway |

---

## Errors

Use `errs` to return structured errors from gRPC handlers. Each error carries a stable `Code` (used as an i18n key), a developer-readable message, and optional metadata.

```go
// Shorthand constructors
return nil, errs.NotFound("ERR_USER_NOT_FOUND", "user not found").
    WithMetadata("id", req.UserId)

return nil, errs.FailedPrecondition("ERR_OUT_OF_STOCK", "insufficient stock").
    WithMetadata("sku", req.Sku).
    WithMetadata("available", stock)

// Field-level validation errors
return nil, errs.Validation("invalid request", map[string]string{
    "email": "FIELD_INVALID_EMAIL",
    "name":  "FIELD_REQUIRED",
})
```

The gateway translates these to JSON automatically:

```json
{"code": "ERR_USER_NOT_FOUND", "message": "user not found", "metadata": {"id": "abc123"}}
{"code": "VALIDATION_ERROR", "message": "invalid request", "metadata": {"email": "FIELD_INVALID_EMAIL"}}
```

### Service-to-service error inspection

```go
code, metadata := errs.Parse(err)
if code == "ERR_USER_NOT_FOUND" { ... }
```

---

## i18n

Embed your own YAML translation files. Keys are error codes; values are Go templates.

```yaml
# translations/en.yaml
ERR_USER_NOT_FOUND: "User was not found."
ERR_OUT_OF_STOCK: "Only {{.available}} units available for SKU {{.sku}}."
```

```go
//go:embed translations/*.yaml
var translationFiles embed.FS

loc, err := i18n.NewFromFS(translationFiles, "en")
if err != nil { ... }

srv, err := pkgserver.New(
    pkgserver.WithGrpcServerUnaryInterceptors(
        middleware.ErrorWrapperUnaryServerInterceptor(
            middleware.WithLocalizeMessageFunc(loc.Translate),
        ),
    ),
)
```

The client's `Accept-Language` header is forwarded by the gateway and resolved automatically.

---

## Config

Define your config struct, embed a YAML defaults file, and call `config.MustLoad`:

```go
//go:embed config.yaml
var defaultConfig []byte

type Config struct {
    Environment string        `mapstructure:"environment"`
    Server      ServerConfig  `mapstructure:"server"`
    Postgres    PostgresConfig `mapstructure:"pg"`
}

cfg := config.MustLoad[Config](defaultConfig)
```

Environment variables override YAML using `__` as the key separator:

```bash
PG__HOST=db.prod.internal
PG__PASSWORD=secret
SERVER__GRPC__PORT=9443
```

---

## Authentication

```go
jwtVal    := auth.NewJWTValidator(cfg.Auth.JWT.Secret)
apiKeyVal := auth.NewAPIKeyValidator(cfg.Auth.APIKeys) // map[string]string: key → name

srv, err := pkgserver.New(
    pkgserver.WithGrpcServerUnaryInterceptors(
        auth.AuthUnaryServerInterceptor(jwtVal, apiKeyVal),
    ),
)
```

After authentication, claims are available in the handler context:

```go
claims := auth.ClaimsFromContext(ctx)  // JWT: UserID, Role
vendor := auth.VendorFromContext(ctx)  // API key: VendorID, Name
```

---

## Logging

`logger.New()` reads two env vars:

```bash
LOG_LEVEL=debug   # debug | info | warn | error  (default: info)
LOG_ENCODER=console  # console | json             (default: json)
```

Inject into and retrieve from context:

```go
ctx = logger.WithContext(ctx, l.With("user_id", userID))
log := logger.FromContext(ctx)
log.InfoContext(ctx, "action completed")
```

---

## Request IDs

`RequestIDUnaryServerInterceptor` generates a unique ID per request, attaches it to the handler context, and forwards it as an HTTP response header. Clients can propagate a trace ID by sending `X-Request-Id`:

```go
id := rpcctx.GetRequestIDFromContext(ctx)
```

---

## Query Helpers

```go
// Paging
p := query.NewOffsetPaging(req.Page, req.PageSize)
db.Offset(p.Offset()).Limit(p.Limit())

// Ordering
o := query.NewOrder(req.OrderBy, query.ParseDirection(req.Direction))
db.OrderExpr(o.String()) // → "created_at DESC NULLS LAST"
```

---

## Server Options

| Option | Description |
|---|---|
| `WithGrpcAddr(host, port)` | Override gRPC listen address (default: `0.0.0.0:10443`) |
| `WithGatewayAddr(host, port)` | Override HTTP listen address (default: `0.0.0.0:10080`) |
| `WithGrpcServerUnaryInterceptors(...)` | Append unary interceptors |
| `WithGrpcServerStreamInterceptors(...)` | Append stream interceptors |
| `WithGatewayServerMiddlewares(...)` | Append HTTP middlewares |
| `WithGatewayCORS(cfg)` | Configure CORS |
| `WithGatewayServiceName(name)` | Set service name for metrics labels |
| `WithServiceServer(srv...)` | Register service implementations |
| `WithPassedHeader(decider)` | Forward custom HTTP headers to gRPC metadata |
| `WithGatewayBasePathOverride(path)` | Mount the gateway mux at a custom base path |
| `WithoutGatewayMetricsRecorder()` | Disable Prometheus HTTP metrics |

### Default interceptors (gRPC)

Wired automatically — no configuration required:

- Prometheus metrics (`grpc_server_*`)
- Structured request/response logging (skips health endpoints)

### Default middlewares (HTTP gateway)

- Structured access logging (skips `/metrics`, `/health`, `/ready`)
- Prometheus HTTP metrics per route

---

## Service Interface

Implement one or more of these interfaces on your service:

```go
// Required — registers gRPC handlers.
type ServiceServer interface {
    RegisterWithServer(*grpc.Server)
}

// Optional — registers HTTP gateway handlers.
type GatewayServer interface {
    RegisterWithHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error
}

// Optional — adds custom HTTP routes (webhooks, file upload, etc.).
type CustomRouteServer interface {
    RegisterCustomRoutes(context.Context, *runtime.ServeMux) error
}

// Optional — called during graceful shutdown.
type Closer interface {
    Close(context.Context)
}
```

---

## License

MIT
