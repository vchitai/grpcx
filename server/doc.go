// Package server provides the core server lifecycle for gRPC + grpc-gateway services.
//
// # Overview
//
// A single [New] call creates both a gRPC server and an HTTP reverse proxy (grpc-gateway),
// wires them together over a local loopback connection, and returns a [Server] that can
// be started and stopped as a unit.
//
// # Defaults
//
// The gRPC server comes pre-configured with:
//   - Prometheus metrics (grpc_server_* histograms and counters)
//   - Structured request/response logging via slog (health endpoints excluded)
//   - Up to 1000 concurrent streams
//
// The HTTP gateway comes pre-configured with:
//   - Prometheus HTTP metrics per route
//   - Structured access logging (skips /metrics, /health, /ready)
//   - JSON marshaling with proto field names and enum strings
//   - Query-string parsing compatible with proto field names
//   - Cookie, Authorization, Accept-Language, and X-API-Key header forwarding
//   - Automatic X-Http-Code override support
//
// # Typical usage
//
//	srv, err := server.New(
//	    server.WithGrpcAddr("0.0.0.0", 10443),
//	    server.WithGatewayAddr("0.0.0.0", 10080),
//	    server.WithGrpcServerUnaryInterceptors(
//	        middleware.ErrorWrapperUnaryServerInterceptor(),
//	        middleware.RequestIDUnaryServerInterceptor(),
//	    ),
//	    server.WithServiceServer(mySvc),
//	)
//	if err != nil { ... }
//	if err := srv.Serve(); err != nil { ... }
//
// # Service interfaces
//
// A service implementation must satisfy at least [ServiceServer] to register gRPC handlers.
// It may also implement [GatewayServer] (HTTP), [CustomRouteServer] (raw HTTP routes),
// and [Closer] (graceful shutdown cleanup).
package server
