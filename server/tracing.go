package server

// This file documents the OpenTelemetry wiring that is applied by default.
//
// gRPC side:
//   otelgrpc.NewServerHandler() is added as a grpc.StatsHandler in
//   createDefaultGRPCConfig() via grpcConfig.ServerOption.  The stats-handler
//   approach (otelgrpc v0.46+) is preferred over the deprecated interceptor
//   approach because it captures streaming RPCs correctly and avoids double
//   instrumentation.
//
// HTTP / grpc-gateway side:
//   otelhttp.NewMiddleware("grpc-gateway") is prepended to
//   gatewayConfig.ServerMiddlewares in createDefaultGatewayConfig(), so every
//   inbound HTTP request is traced end-to-end through the gateway and into the
//   gRPC handler.
//
// Applications call tracing.Setup() at startup to wire a real exporter; without
// it the global no-op provider is used so no panics occur.
