// Package tracing provides helpers to initialise an OpenTelemetry tracer
// provider backed by an OTLP gRPC exporter and register it as the global
// provider.  Applications call [Setup] once at startup and defer the returned
// shutdown function.
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Option configures the tracing provider.
type Option func(*config)

type config struct {
	serviceName  string
	otlpEndpoint string
	insecure     bool
}

func defaultConfig() *config {
	return &config{
		serviceName:  "grpc-service",
		otlpEndpoint: "localhost:4317",
		insecure:     true,
	}
}

// WithServiceName sets the OTEL service name resource attribute.
func WithServiceName(name string) Option {
	return func(c *config) { c.serviceName = name }
}

// WithOTLPEndpoint sets the OTLP collector endpoint (default: "localhost:4317").
func WithOTLPEndpoint(endpoint string) Option {
	return func(c *config) { c.otlpEndpoint = endpoint }
}

// WithInsecure disables TLS for the OTLP exporter.
func WithInsecure() Option {
	return func(c *config) { c.insecure = true }
}

// Setup initialises the global OpenTelemetry tracer provider with an OTLP gRPC
// exporter.  Returns a shutdown function that must be called on application exit.
// If the OTLP connection cannot be established, a no-op provider is registered
// so the application continues to function without tracing.
func Setup(ctx context.Context, opts ...Option) (shutdown func(context.Context) error, err error) {
	cfg := defaultConfig()
	for _, o := range opts {
		o(cfg)
	}

	dialOpts := []grpc.DialOption{}
	if cfg.insecure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	exporterOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.otlpEndpoint),
		otlptracegrpc.WithDialOption(dialOpts...),
	}
	if cfg.insecure {
		exporterOpts = append(exporterOpts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, exporterOpts...)
	if err != nil {
		// Fall back to no-op so the application can still start without a
		// collector available.
		otel.SetTracerProvider(noop.NewTracerProvider())
		return func(context.Context) error { return nil },
			fmt.Errorf("tracing: failed to create OTLP exporter (falling back to no-op): %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return tp.Shutdown, nil
}

// Tracer is a convenience wrapper that returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
