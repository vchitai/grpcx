// Package middleware provides gRPC unary and stream server interceptors.
//
// # Interceptors
//
// [ErrorWrapperUnaryServerInterceptor] maps application errors to gRPC status codes,
// optionally localizes messages via [WithLocalizeMessageFunc], and attaches debug stack
// traces in development mode. It handles three error categories:
//   - *errs.Error — application errors with a stable code and metadata
//   - gRPC status errors — forwarded as-is
//   - unhandled errors — mapped to Internal with the original logged server-side
//
// [RequestIDUnaryServerInterceptor] generates a unique request ID for every call,
// attaches it to the handler context, and forwards it as a gRPC trailer so the
// gateway can set the X-Request-Id response header.
//
// [RecoverHandler] converts panics to Internal gRPC errors and logs the stack trace.
// Use it with grpc-ecosystem's recovery interceptor:
//
//	grpcRecovery.UnaryServerInterceptor(
//	    grpcRecovery.WithRecoveryHandlerContext(middleware.RecoverHandler()),
//	)
//
// # HTTP middlewares
//
// [WithHTTPMethod] extracts the HTTP method from the incoming request and stores it
// in gRPC metadata so handlers can read it with [rpcctx.GetHTTPMethodFromContext].
//
// [WithRequestID] reads X-Request-Id from the HTTP request (or generates one) and
// injects it into gRPC metadata so it is available throughout the call chain.
package middleware
