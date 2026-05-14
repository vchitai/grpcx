package server

import (
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// Option configures a gRPC and gateway server.
type Option func(*Config)

func createConfig(opts []Option) *Config {
	c := createDefaultConfig()
	for _, f := range opts {
		f(c)
	}
	return c
}

func WithGatewayAddr(host string, port int) Option {
	return func(c *Config) {
		c.Gateway.Addr = Listen{Host: host, Port: port}
	}
}

func WithGatewayAddrListen(l Listen) Option {
	return func(c *Config) { c.Gateway.Addr = l }
}

func WithGatewayMuxOptions(opts ...runtime.ServeMuxOption) Option {
	return func(c *Config) {
		c.Gateway.MuxOptions = append(c.Gateway.MuxOptions, opts...)
	}
}

func WithGatewayServerMiddlewares(middlewares ...HTTPServerMiddleware) Option {
	return func(c *Config) {
		c.Gateway.ServerMiddlewares = append(c.Gateway.ServerMiddlewares, middlewares...)
	}
}

func WithGatewayServerHandler(handlers ...HTTPServerHandler) Option {
	return func(c *Config) {
		c.Gateway.ServerHandlers = append(c.Gateway.ServerHandlers, handlers...)
	}
}

func WithGatewayCORS(cfg *CORSConfig) Option {
	return func(c *Config) {
		if cfg == nil {
			c.Gateway.CORS = nil
			return
		}
		cc := *cfg
		cc.AllowedOrigins = append([]string(nil), cfg.AllowedOrigins...)
		cc.AllowedMethods = append([]string(nil), cfg.AllowedMethods...)
		cc.AllowedHeaders = append([]string(nil), cfg.AllowedHeaders...)
		cc.ExposedHeaders = append([]string(nil), cfg.ExposedHeaders...)
		c.Gateway.CORS = &cc
	}
}

func WithGatewayServerConfig(cfg *HTTPServerConfig) Option {
	return func(c *Config) { c.Gateway.ServerConfig = cfg }
}

func WithGatewayServiceName(serviceName string) Option {
	return func(c *Config) { c.Gateway.ServiceName = serviceName }
}

// WithPassedHeader forwards whitelisted headers to gRPC metadata.
func WithPassedHeader(decider PassedHeaderDeciderFunc) Option {
	return WithGatewayServerMiddlewares(createPassingHeaderMiddleware(decider))
}

func WithGatewayBasePathOverride(basePath string) Option {
	return func(c *Config) { c.Gateway.BasePathOverride = basePath }
}

func WithoutGatewayMetricsRecorder() Option {
	return func(c *Config) { c.Gateway.NoMetricsRecorder = true }
}

// --- gRPC options ---

func WithGrpcAddr(host string, port int) Option {
	return func(c *Config) {
		c.Grpc.Addr = Listen{Host: host, Port: port}
	}
}

func WithGrpcAddrListen(l Listen) Option {
	return func(c *Config) { c.Grpc.Addr = l }
}

func WithGrpcServerUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) Option {
	return func(c *Config) {
		c.Grpc.ServerUnaryInterceptors = append(c.Grpc.ServerUnaryInterceptors, interceptors...)
	}
}

func WithGrpcServerStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) Option {
	return func(c *Config) {
		c.Grpc.ServerStreamInterceptors = append(c.Grpc.ServerStreamInterceptors, interceptors...)
	}
}

func WithServiceServer(srv ...ServiceServer) Option {
	return func(c *Config) {
		c.ServiceServers = append(c.ServiceServers, srv...)
	}
}

// WithSensitiveFields overrides the list of proto field names redacted from payload logs.
// The default list is: password, new_password, passcode, token, otp.
func WithSensitiveFields(fields ...string) Option {
	return func(c *Config) {
		c.Grpc.SensitiveFields = fields
	}
}

// WithSkipLoggingMethods skips request/response payload logging for the given full method names.
// Useful for noisy health-check or polling endpoints.
//
//	server.WithSkipLoggingMethods(
//	    "/grpc.health.v1.Health/Check",
//	    "/api.health.v1.HealthService/Liveness",
//	)
func WithSkipLoggingMethods(methods ...string) Option {
	return func(c *Config) {
		c.Grpc.SkipLoggingMethods = append(c.Grpc.SkipLoggingMethods, methods...)
	}
}
