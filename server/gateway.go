package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"strconv"
	"sync"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/vchitai/grpcx/grpc/gatewayopt"
	"github.com/vchitai/grpcx/grpc/protojson"
)

const (
	metadataPattern    = "pattern"
	defaultGatewayPort = 10080
	HeaderHTTPCode     = "X-Http-Code"
)

// --- CORS ---

// CORSConfig holds CORS configuration for the HTTP gateway.
type CORSConfig struct {
	AllowedOrigins     []string `json:"allowed_origins"     mapstructure:"allowed_origins"     yaml:"allowed_origins"`
	AllowedMethods     []string `json:"allowed_methods"     mapstructure:"allowed_methods"     yaml:"allowed_methods"`
	AllowedHeaders     []string `json:"allowed_headers"     mapstructure:"allowed_headers"     yaml:"allowed_headers"`
	ExposedHeaders     []string `json:"exposed_headers"     mapstructure:"exposed_headers"     yaml:"exposed_headers"`
	AllowCredentials   bool     `json:"allow_credentials"   mapstructure:"allow_credentials"   yaml:"allow_credentials"`
	OptionsPassthrough bool     `json:"options_passthrough" mapstructure:"options_passthrough" yaml:"options_passthrough"`
	MaxAge             int      `json:"max_age"             mapstructure:"max_age"             yaml:"max_age"`
	Debug              bool     `json:"debug"               mapstructure:"debug"               yaml:"debug"`
}

func (c *CORSConfig) toOptions() cors.Options {
	opts := cors.Options{
		AllowedOrigins:     append([]string(nil), c.AllowedOrigins...),
		AllowedMethods:     append([]string(nil), c.AllowedMethods...),
		AllowedHeaders:     append([]string(nil), c.AllowedHeaders...),
		ExposedHeaders:     append([]string(nil), c.ExposedHeaders...),
		AllowCredentials:   c.AllowCredentials,
		OptionsPassthrough: c.OptionsPassthrough,
		MaxAge:             c.MaxAge,
		Debug:              c.Debug,
	}
	if len(opts.AllowedMethods) == 0 {
		opts.AllowedMethods = []string{
			http.MethodDelete, http.MethodGet, http.MethodPatch,
			http.MethodPost, http.MethodPut, http.MethodOptions,
		}
	}
	if len(opts.AllowedHeaders) == 0 {
		opts.AllowedHeaders = []string{
			"Accept", "Authorization", "Content-Type", "Origin", "X-Requested-With",
		}
	}
	return opts
}

// --- Mux / Handlers ---

// HTTPServerMux is the interface for the HTTP server multiplexer.
type HTTPServerMux interface {
	Handle(pattern string, handler http.Handler)
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// HTTPServerHandler registers routes on an HTTPServerMux.
type HTTPServerHandler func(HTTPServerMux)

func PrometheusHandler(mux HTTPServerMux) {
	mux.Handle("/metrics", promhttp.Handler())
}

func PprofHandler(mux HTTPServerMux) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

func CORSHandler(mux HTTPServerMux, cfg *CORSConfig) http.Handler {
	if cfg == nil {
		return cors.AllowAll().Handler(mux)
	}
	return cors.New(cfg.toOptions()).Handler(mux)
}

// --- Response modifier ---

// HTTPResponseModifier allows overriding the HTTP status code from gRPC metadata.
// Usage:
//   - Init: server.WithGatewayMuxOptions(runtime.WithForwardResponseOption(server.HTTPResponseModifier))
//   - In handler: grpc.SetHeader(ctx, metadata.Pairs("x-http-code", "201"))
func HTTPResponseModifier(ctx context.Context, w http.ResponseWriter, _ proto.Message) error {
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return nil
	}
	if vals := md.HeaderMD.Get(HeaderHTTPCode); len(vals) > 0 {
		code, err := strconv.Atoi(vals[0])
		if err != nil {
			return err
		}
		delete(md.HeaderMD, HeaderHTTPCode)
		delete(w.Header(), "Grpc-Metadata-X-Http-Code")
		w.WriteHeader(code)
	}
	return nil
}

// --- Gateway server ---

type gatewayServer struct {
	server *http.Server
	config *gatewayConfig
}

type HTTPServerConfig struct {
	TLSConfig         *tls.Config
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	TLSNextProto      map[string]func(*http.Server, *tls.Conn, http.Handler)
	ConnState         func(net.Conn, http.ConnState)
}

func (c *HTTPServerConfig) applyTo(s *http.Server) {
	s.TLSConfig = c.TLSConfig
	s.ReadTimeout = c.ReadTimeout
	s.ReadHeaderTimeout = c.ReadHeaderTimeout
	s.WriteTimeout = c.WriteTimeout
	s.IdleTimeout = c.IdleTimeout
	s.MaxHeaderBytes = c.MaxHeaderBytes
	s.TLSNextProto = c.TLSNextProto
	s.ConnState = c.ConnState
}

type gatewayConfig struct {
	Addr              Listen
	ServiceName       string
	MuxOptions        []runtime.ServeMuxOption
	ServerConfig      *HTTPServerConfig
	ServerMiddlewares []HTTPServerMiddleware
	ServerHandlers    []HTTPServerHandler
	BasePathOverride  string
	NoMetricsRecorder bool
	CORS              *CORSConfig
}

func createDefaultGatewayConfig() *gatewayConfig {
	return &gatewayConfig{
		Addr: Listen{Host: "0.0.0.0", Port: defaultGatewayPort},
		MuxOptions: []runtime.ServeMuxOption{
			gatewayopt.FormURLEncodedMarshaler(),
			runtime.SetQueryParameterParser(protojson.NewGeaQueryParser()),
			runtime.WithMarshalerOption(runtime.MIMEWildcard, protojson.NewGeaMarshaler()),
			runtime.WithMetadata(SetMetadataPattern),
			runtime.WithMetadata(SetMetadataAuthorization),
			runtime.WithMetadata(SetMetadataCookie),
			runtime.WithMetadata(SetMetadataAPIKey),
			runtime.WithMetadata(SetMetadataLanguage),
			runtime.WithOutgoingHeaderMatcher(func(s string) (string, bool) {
				if s == "set-cookie" {
					return "Set-Cookie", true
				}
				return "", false
			}),
			runtime.WithErrorHandler(ErrorHandler),
		},
		ServerHandlers:    []HTTPServerHandler{PrometheusHandler},
		ServerMiddlewares: []HTTPServerMiddleware{loggingMiddleware},
	}
}

func newGatewayServer(c *gatewayConfig, conn *grpc.ClientConn, servers []ServiceServer) (*gatewayServer, error) {
	mux := runtime.NewServeMux(c.MuxOptions...)
	for _, svr := range servers {
		if gs, ok := svr.(GatewayServer); ok {
			if err := gs.RegisterWithHandler(context.Background(), mux, conn); err != nil {
				return nil, fmt.Errorf("failed to register handler: %w", err)
			}
		}
		if cs, ok := svr.(CustomRouteServer); ok {
			if err := cs.RegisterCustomRoutes(context.Background(), mux); err != nil {
				return nil, fmt.Errorf("failed to register custom routes: %w", err)
			}
		}
	}

	var handler http.Handler = mux
	for i := len(c.ServerMiddlewares) - 1; i >= 0; i-- {
		handler = c.ServerMiddlewares[i](handler)
	}

	var httpMux HTTPServerMux = http.NewServeMux()
	if !c.NoMetricsRecorder {
		httpMux = &MetricsRecordingMux{
			HTTPServerMux: httpMux,
			metrics:       getHTTPMetrics(),
			serviceName:   c.ServiceName,
		}
	}

	for _, h := range c.ServerHandlers {
		h(httpMux)
	}

	if len(c.BasePathOverride) == 0 {
		httpMux.Handle("/", handler)
	} else {
		httpMux.Handle(c.BasePathOverride, handler)
	}

	svr := &http.Server{
		Addr:              c.Addr.String(),
		Handler:           CORSHandler(httpMux, c.CORS),
		ReadHeaderTimeout: ServerShutdownTimeout,
	}
	if cfg := c.ServerConfig; cfg != nil {
		cfg.applyTo(svr)
	}

	return &gatewayServer{server: svr, config: c}, nil
}

// httpMetrics records per-route HTTP metrics using prometheus/client_golang.
type httpMetrics struct {
	reqTotal    *prometheus.CounterVec
	reqDuration *prometheus.HistogramVec
}

var (
	globalHTTPMetrics     *httpMetrics
	globalHTTPMetricsOnce sync.Once
)

func mustRegister[C prometheus.Collector](c C) C {
	if err := prometheus.Register(c); err != nil {
		var are prometheus.AlreadyRegisteredError
		if errors.As(err, &are) {
			return are.ExistingCollector.(C)
		}
		panic(err)
	}
	return c
}

func getHTTPMetrics() *httpMetrics {
	globalHTTPMetricsOnce.Do(func() {
		globalHTTPMetrics = &httpMetrics{
			reqTotal: mustRegister(prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests completed.",
			}, []string{"service", "handler", "method", "code"})),
			reqDuration: mustRegister(prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds.",
				Buckets: prometheus.DefBuckets,
			}, []string{"service", "handler", "method"})),
		}
	})
	return globalHTTPMetrics
}

func (m *httpMetrics) wrap(service, pattern string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := wrapResponseWriter(w)
		start := time.Now()
		h.ServeHTTP(rw, r)
		dur := time.Since(start).Seconds()
		code := strconv.Itoa(rw.status)
		m.reqTotal.WithLabelValues(service, pattern, r.Method, code).Inc()
		m.reqDuration.WithLabelValues(service, pattern, r.Method).Observe(dur)
	})
}

// MetricsRecordingMux wraps an HTTPServerMux and records per-route Prometheus metrics.
type MetricsRecordingMux struct {
	HTTPServerMux
	metrics     *httpMetrics
	serviceName string
}

func (mux *MetricsRecordingMux) Handle(pattern string, handler http.Handler) {
	mux.HTTPServerMux.Handle(pattern, mux.metrics.wrap(mux.serviceName, pattern, handler))
}

func (mux *MetricsRecordingMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	mux.Handle(pattern, http.HandlerFunc(handler))
}

func (s *gatewayServer) Serve() error {
	slog.Info("http server starting", "addr", s.config.Addr.String())
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Info("http server stopped", "err", err)
		return err
	}
	return nil
}

func (s *gatewayServer) Shutdown(ctx context.Context) {
	err := s.server.Shutdown(ctx)
	slog.Info("http server shutdown complete")
	if err != nil {
		slog.Info("http server shutdown error", "err", err)
	}
}

// --- Metadata setters ---

func SetMetadataPattern(ctx context.Context, _ *http.Request) metadata.MD {
	md := make(map[string]string)
	if pattern, ok := runtime.HTTPPathPattern(ctx); ok {
		md[metadataPattern] = pattern
	}
	return metadata.New(md)
}

// SetMetadataAuthorization forwards authentication to gRPC metadata.
// It prefers the session_token cookie (wrapping it as a Bearer token) over
// a plain Authorization header, so cookie-based sessions take precedence.
func SetMetadataAuthorization(_ context.Context, r *http.Request) metadata.MD {
	if cookie, err := r.Cookie("session_token"); err == nil {
		return metadata.Pairs("authorization", "Bearer "+cookie.Value)
	}
	if h := r.Header.Get("Authorization"); h != "" {
		return metadata.Pairs("authorization", h)
	}
	return nil
}

// SetMetadataAPIKey forwards X-API-Key from HTTP to gRPC metadata.
func SetMetadataAPIKey(_ context.Context, r *http.Request) metadata.MD {
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return metadata.Pairs("x-api-key", apiKey)
	}
	return nil
}

// SetMetadataCookie forwards the Cookie header to gRPC metadata.
func SetMetadataCookie(_ context.Context, r *http.Request) metadata.MD {
	if cookieHeader := r.Header.Get("Cookie"); cookieHeader != "" {
		return metadata.Pairs("cookie", cookieHeader)
	}
	return nil
}

// SetMetadataLanguage forwards Accept-Language to gRPC metadata for server-side i18n.
func SetMetadataLanguage(_ context.Context, r *http.Request) metadata.MD {
	if lang := r.Header.Get("Accept-Language"); lang != "" {
		return metadata.Pairs("accept-language", lang)
	}
	return nil
}

// --- Error handler ---

type ErrorMessage struct {
	Code     string         `json:"code"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func ErrorHandler(
	_ context.Context,
	_ *runtime.ServeMux,
	marshaler runtime.Marshaler,
	w http.ResponseWriter,
	_ *http.Request,
	err error,
) {
	const fallback = `{"code": "INTERNAL_SERVER_ERROR", "message": "Something went wrong. Please try again later."}`

	var customStatus *runtime.HTTPStatusError
	if errors.As(err, &customStatus) {
		err = customStatus.Err
	}

	s := status.Convert(err)
	pb := s.Proto()

	w.Header().Del("Trailer")
	w.Header().Del("Transfer-Encoding")
	w.Header().Set("Content-Type", marshaler.ContentType(pb))

	if s.Code() == codes.Unauthenticated {
		w.Header().Set("WWW-Authenticate", "Unauthorized")
	}

	errMessage := ErrorMessage{Code: "UNKNOWN_ERROR", Message: s.Message()}
	for _, detail := range s.Details() {
		switch v := detail.(type) {
		case *errdetails.ErrorInfo:
			errMessage.Code = v.Reason
			if len(v.Metadata) > 0 {
				errMessage.Metadata = make(map[string]any, len(v.Metadata))
				for k, val := range v.Metadata {
					errMessage.Metadata[k] = val
				}
			}
		case *errdetails.BadRequest:
			errMessage.Code = "VALIDATION_ERROR"
			fields := make(map[string]any, len(v.GetFieldViolations()))
			for _, fv := range v.GetFieldViolations() {
				fields[fv.GetField()] = fv.GetDescription()
			}
			errMessage.Metadata = fields
		}
	}

	buf, merr := marshaler.Marshal(errMessage)
	if merr != nil {
		grpclog.Infof("Failed to marshal error message %q: %v", s, merr)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(fallback)); err != nil {
			grpclog.Infof("Failed to write response: %v", err)
		}
		return
	}

	st := runtime.HTTPStatusFromCode(s.Code())
	if customStatus != nil {
		st = customStatus.HTTPStatus
	}
	w.WriteHeader(st)
	if _, err := w.Write(buf); err != nil { // nosemgrep
		grpclog.Infof("Failed to write response: %v", err)
	}
}
