package server

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	MaxCallRecvMsgSize    = 1024 * 1024 * 50
	ServerShutdownTimeout = 30 * time.Second
)

// Server is the framework instance.
type Server struct {
	grpcServer    *grpcServer
	gatewayServer *gatewayServer
	config        *Config
}

// New creates a server instance.
func New(opts ...Option) (*Server, error) {
	c := createConfig(opts)

	slog.Info("Create grpc server")
	grpcServer := newGrpcServer(c.Grpc, c.ServiceServers)

	conn, err := grpc.NewClient(
		c.Grpc.Addr.String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxCallRecvMsgSize)),
		grpc.WithChainUnaryInterceptor(),
	)
	if err != nil {
		return nil, fmt.Errorf("fail to dial gRPC server: %w", err)
	}

	slog.Info("Create gateway server")
	gatewayServer, err := newGatewayServer(c.Gateway, conn, c.ServiceServers)
	if err != nil {
		return nil, fmt.Errorf("fail to create gateway server: %w", err)
	}

	return &Server{
		grpcServer:    grpcServer,
		gatewayServer: gatewayServer,
		config:        c,
	}, nil
}

// Start starts both gRPC and HTTP gateway servers, blocking until both exit.
func (s *Server) Start() {
	var wg sync.WaitGroup
	wg.Go(func() {
		if err := s.gatewayServer.Serve(); err != nil {
			slog.Error("http server error", "err", err)
		}
	})
	wg.Go(func() {
		if err := s.grpcServer.Serve(context.Background()); err != nil {
			slog.Error("gRPC server error", "err", err)
		}
	})
	wg.Wait()
}

// Stop gracefully shuts down all servers and calls Close on any service that implements Closer.
func (s *Server) Stop() {
	slog.Info("Shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), ServerShutdownTimeout)
	defer cancel()
	for _, ss := range s.config.ServiceServers {
		if c, ok := ss.(Closer); ok {
			c.Close(ctx)
		}
	}
	s.gatewayServer.Shutdown(ctx)
	s.grpcServer.Shutdown(ctx)
}

// Serve starts both servers and returns the first error.
func (s *Server) Serve() error {
	errch := make(chan error, 2)
	go func() {
		if err := s.gatewayServer.Serve(); err != nil {
			errch <- err
		}
	}()
	go func() {
		if err := s.grpcServer.Serve(context.Background()); err != nil {
			errch <- err
		}
	}()
	return <-errch
}

func (s *Server) ServeGateway() error          { return s.gatewayServer.Serve() }
func (s *Server) ServeGRPC() error             { return s.grpcServer.Serve(context.Background()) }
func (s *Server) ShutdownGateway(ctx context.Context) { s.gatewayServer.Shutdown(ctx) }
func (s *Server) ShutdownGRPC(ctx context.Context)    { s.grpcServer.Shutdown(ctx) }
