package worker

import (
	"context"

	"github.com/hibiken/asynq"
)

// ServerOption configures a Server.
type ServerOption func(*serverConfig)

type serverConfig struct {
	concurrency int
	queues      map[string]int
}

// defaultServerConfig returns the default server configuration:
// concurrency=10, queues={"critical":6,"default":3,"low":1}.
func defaultServerConfig() *serverConfig {
	return &serverConfig{
		concurrency: 10,
		queues: map[string]int{
			"critical": 6,
			"default":  3,
			"low":      1,
		},
	}
}

// WithConcurrency sets the maximum number of concurrent task processors.
func WithConcurrency(n int) ServerOption {
	return func(c *serverConfig) {
		c.concurrency = n
	}
}

// WithQueues sets the queue names and their priority weights.
func WithQueues(queues map[string]int) ServerOption {
	return func(c *serverConfig) {
		c.queues = queues
	}
}

// Server processes background tasks.
type Server struct {
	srv *asynq.Server
	mux *asynq.ServeMux
}

// NewServer creates a new Server connected to Redis using cfg.
// loggingMiddleware is added to the mux by default.
func NewServer(cfg RedisConfig, opts ...ServerOption) *Server {
	sc := defaultServerConfig()
	for _, o := range opts {
		o(sc)
	}

	srv := asynq.NewServer(cfg.clientOpt(), asynq.Config{
		Concurrency: sc.concurrency,
		Queues:      sc.queues,
		Logger:      newAsynqLogger(),
	})

	mux := asynq.NewServeMux()
	mux.Use(loggingMiddleware)

	return &Server{
		srv: srv,
		mux: mux,
	}
}

// NewServerFromConfig creates a Server from a YAML-mapped ServerConfig.
// Non-zero fields in cfg override the defaults; opts are applied afterwards.
func NewServerFromConfig(redis RedisConfig, cfg ServerConfig, opts ...ServerOption) *Server {
	var derived []ServerOption
	if cfg.Concurrency > 0 {
		derived = append(derived, WithConcurrency(cfg.Concurrency))
	}
	if len(cfg.Queues) > 0 {
		derived = append(derived, WithQueues(cfg.Queues))
	}
	return NewServer(redis, append(derived, opts...)...)
}

// handle registers a raw asynq.Handler for the given task type name.
// This is unexported so external callers use the type-safe Register function.
func (s *Server) handle(typeName string, h asynq.Handler) {
	s.mux.Handle(typeName, h)
}

// Register wires a typed handler fn to the server for the given task.
// It is a package-level function to work around Go's restriction on generic
// methods on non-generic types.
func Register[P any](s *Server, task Task[P], fn func(ctx context.Context, payload P) error) {
	s.handle(task.TypeName(), task.Handler(fn))
}

// Run blocks until the server exits (OS signal or unrecoverable error).
// asynq handles SIGTERM and SIGINT internally, so this is suitable for use
// as the body of a simple main().
func (s *Server) Run() error {
	return s.srv.Run(s.mux)
}

// Start runs the server in the background (non-blocking).
func (s *Server) Start() error {
	return s.srv.Start(s.mux)
}

// Stop gracefully shuts down the server, waiting for in-flight tasks to finish.
func (s *Server) Stop() {
	s.srv.Stop()
	s.srv.Shutdown()
}
