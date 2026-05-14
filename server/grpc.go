package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
)

var serverMetrics = grpcprom.NewServerMetrics(
	grpcprom.WithServerHandlingTimeHistogram(),
)

func init() {
	if err := prometheus.Register(serverMetrics); err != nil {
		var are prometheus.AlreadyRegisteredError
		if !errors.As(err, &are) {
			panic(err)
		}
	}
}

// defaultSensitiveFields are redacted from gRPC payload logs.
var defaultSensitiveFields = []string{"password", "new_password", "passcode", "token", "otp"}

type grpcConfig struct {
	Addr                     Listen
	ServerUnaryInterceptors  []grpc.UnaryServerInterceptor  // app-provided, appended after defaults
	ServerStreamInterceptors []grpc.StreamServerInterceptor // app-provided, appended after defaults
	ServerOption             []grpc.ServerOption
	MaxConcurrentStreams      uint32
	SensitiveFields          []string // fields to redact from payload logs
	SkipLoggingMethods       []string // full method names to skip request/response logging
}

var defaultServerLoggingTimestamp = time.RFC3339

const (
	maxConcurrentStreams = 1000
	defaultServerPort    = 10443
)

func createDefaultGRPCConfig() *grpcConfig {
	return &grpcConfig{
		Addr:                Listen{Host: "0.0.0.0", Port: defaultServerPort},
		MaxConcurrentStreams: maxConcurrentStreams,
		SensitiveFields:     defaultSensitiveFields,
	}
}

// buildDefaultUnaryInterceptors creates the framework's built-in unary interceptors
// using the final config values. Called at server creation, after options are applied.
func (c *grpcConfig) buildDefaultUnaryInterceptors() []grpc.UnaryServerInterceptor {
	l := slog.Default()
	skip := make(map[string]bool, len(c.SkipLoggingMethods))
	for _, m := range c.SkipLoggingMethods {
		skip[m] = true
	}
	logInterceptor := logging.UnaryServerInterceptor(
		InterceptorLogger(l, c.SensitiveFields...),
		logging.WithLogOnEvents(logging.PayloadSent, logging.PayloadReceived),
		logging.WithTimestampFormat(defaultServerLoggingTimestamp),
		logging.WithLevels(DefaultServerCodeToLevel),
	)
	return []grpc.UnaryServerInterceptor{
		serverMetrics.UnaryServerInterceptor(),
		func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			if skip[info.FullMethod] {
				return handler(ctx, req)
			}
			return logInterceptor(ctx, req, info, handler)
		},
	}
}

func (c *grpcConfig) buildDefaultStreamInterceptors() []grpc.StreamServerInterceptor {
	l := slog.Default()
	return []grpc.StreamServerInterceptor{
		serverMetrics.StreamServerInterceptor(),
		logging.StreamServerInterceptor(
			InterceptorLogger(l, c.SensitiveFields...),
			logging.WithLogOnEvents(logging.PayloadSent, logging.PayloadReceived),
			logging.WithTimestampFormat(defaultServerLoggingTimestamp),
			logging.WithLevels(DefaultServerCodeToLevel),
		),
	}
}

func (c *grpcConfig) ServerOptions() []grpc.ServerOption {
	unary := append(c.buildDefaultUnaryInterceptors(), c.ServerUnaryInterceptors...)
	stream := append(c.buildDefaultStreamInterceptors(), c.ServerStreamInterceptors...)
	return append(
		[]grpc.ServerOption{
			grpc.ChainUnaryInterceptor(unary...),
			grpc.ChainStreamInterceptor(stream...),
			grpc.MaxConcurrentStreams(c.MaxConcurrentStreams),
		},
		c.ServerOption...,
	)
}

type grpcServer struct {
	server *grpc.Server
	config *grpcConfig
}

func newGrpcServer(c *grpcConfig, servers []ServiceServer) *grpcServer {
	s := grpc.NewServer(c.ServerOptions()...)
	for _, svr := range servers {
		svr.RegisterWithServer(s)
	}
	return &grpcServer{server: s, config: c}
}

func (s *grpcServer) Serve(ctx context.Context) error {
	listener, err := s.config.Addr.CreateListener(ctx)
	if err != nil {
		return fmt.Errorf("failed to create listener %w", err)
	}
	slog.Info("gRPC server starting", "addr", listener.Addr().String())

	if err = s.server.Serve(listener); err != nil {
		slog.Info("gRPC server stopped", "err", err)
		return fmt.Errorf("failed to serve gRPC server %w", err)
	}
	return nil
}

func (s *grpcServer) Shutdown(_ context.Context) {
	slog.Info("gRPC server shutting down")
	s.server.GracefulStop()
}

func DefaultServerCodeToLevel(code codes.Code) logging.Level {
	switch code {
	case codes.OK, codes.NotFound, codes.Canceled, codes.AlreadyExists, codes.InvalidArgument, codes.Unauthenticated:
		return logging.LevelDebug
	case codes.DeadlineExceeded, codes.PermissionDenied, codes.ResourceExhausted,
		codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unavailable:
		return logging.LevelWarn
	case codes.Unknown, codes.Unimplemented, codes.Internal, codes.DataLoss:
		return logging.LevelError
	default:
		return logging.LevelError
	}
}
