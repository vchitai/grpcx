package server

import (
	"context"
	"fmt"
	"net"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// Config is the top-level server configuration.
type Config struct {
	Gateway        *gatewayConfig
	Grpc           *grpcConfig
	ServiceServers []ServiceServer
}

func createDefaultConfig() *Config {
	return &Config{
		Grpc:    createDefaultGRPCConfig(),
		Gateway: createDefaultGatewayConfig(),
	}
}

// Listen represents a TCP address to bind to.
type Listen struct {
	Host string `json:"host" mapstructure:"host" yaml:"host"`
	Port int    `json:"port" mapstructure:"port" yaml:"port"`
}

func (l *Listen) String() string {
	return fmt.Sprintf("%s:%d", l.Host, l.Port)
}

func (l *Listen) CreateListener(ctx context.Context) (net.Listener, error) {
	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", l.String())
	if err != nil {
		return nil, fmt.Errorf("failed to listen %s: %w", l.String(), err)
	}
	return lis, nil
}

// ServiceServer is the only required interface. It registers gRPC handlers.
type ServiceServer interface {
	RegisterWithServer(*grpc.Server)
}

// GatewayServer is optionally implemented to register with the HTTP gateway mux.
type GatewayServer interface {
	RegisterWithHandler(context.Context, *runtime.ServeMux, *grpc.ClientConn) error
}

// CustomRouteServer is optionally implemented to add custom HTTP routes.
type CustomRouteServer interface {
	RegisterCustomRoutes(context.Context, *runtime.ServeMux) error
}

// Closer is optionally implemented for cleanup during server shutdown.
type Closer interface {
	Close(context.Context)
}
