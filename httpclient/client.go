package httpclient

import (
	"net/http"
	"time"

	"github.com/imroc/req/v3"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Option configures a req.Client.
type Option func(*req.Client)

// WithBaseURL sets the base URL for all requests.
func WithBaseURL(url string) Option {
	return func(c *req.Client) {
		c.SetBaseURL(url)
	}
}

// WithTimeout sets the request timeout. Default: 30s.
func WithTimeout(d time.Duration) Option {
	return func(c *req.Client) {
		c.SetTimeout(d)
	}
}

// WithRetry sets the retry count and optional wait times.
//
//	WithRetry(3)                                        — 3 retries with req's default backoff
//	WithRetry(3, 500*time.Millisecond, 30*time.Second)  — retry with min/max wait
func WithRetry(count int, waitTimes ...time.Duration) Option {
	return func(c *req.Client) {
		c.SetCommonRetryCount(count)
		if len(waitTimes) >= 2 {
			c.SetCommonRetryBackoffInterval(waitTimes[0], waitTimes[1])
		} else if len(waitTimes) == 1 {
			c.SetCommonRetryFixedInterval(waitTimes[0])
		}
	}
}

// WithBearerToken sets Authorization: Bearer <token>.
func WithBearerToken(token string) Option {
	return func(c *req.Client) {
		c.SetCommonBearerAuthToken(token)
	}
}

// WithAPIKey sets a header-based API key. If header is empty, uses "X-API-Key".
func WithAPIKey(key, header string) Option {
	return func(c *req.Client) {
		if header == "" {
			header = "X-API-Key"
		}
		c.SetCommonHeader(header, key)
	}
}

// WithBasicAuth sets HTTP Basic Auth credentials.
func WithBasicAuth(username, password string) Option {
	return func(c *req.Client) {
		c.SetCommonBasicAuth(username, password)
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *req.Client) {
		c.SetUserAgent(ua)
	}
}

// WithHeader sets a common header on all requests.
func WithHeader(key, value string) Option {
	return func(c *req.Client) {
		c.SetCommonHeader(key, value)
	}
}

// WithTLSInsecure disables TLS certificate verification. Do not use in production.
func WithTLSInsecure() Option {
	return func(c *req.Client) {
		c.EnableInsecureSkipVerify()
	}
}

// WithOTel wraps the transport with otelhttp.NewTransport so all requests emit
// OpenTelemetry spans. Uses the globally registered tracer provider.
func WithOTel() Option {
	return func(c *req.Client) {
		c.GetTransport().WrapRoundTrip(func(rt http.RoundTripper) http.RoundTripper {
			return otelhttp.NewTransport(rt)
		})
	}
}

// WithSlogLogger replaces req's default logger with one backed by slog.Default().
func WithSlogLogger() Option {
	return func(c *req.Client) {
		c.SetLogger(&slogLogger{})
	}
}

// New creates a req.Client with slog logger and 30s default timeout pre-applied,
// then applies the given options.
func New(opts ...Option) *req.Client {
	c := req.NewClient()
	// Always apply defaults first.
	WithSlogLogger()(c)
	WithTimeout(30 * time.Second)(c)
	for _, o := range opts {
		o(c)
	}
	return c
}

// NewFromConfig creates a req.Client from cfg, then applies any extra opts.
// Zero-value fields in cfg are skipped (e.g. zero Timeout means keep default 30s).
func NewFromConfig(cfg Config, opts ...Option) *req.Client {
	var derived []Option

	if cfg.BaseURL != "" {
		derived = append(derived, WithBaseURL(cfg.BaseURL))
	}
	if cfg.Timeout != 0 {
		derived = append(derived, WithTimeout(cfg.Timeout))
	}
	if cfg.RetryCount > 0 {
		switch {
		case cfg.RetryWaitTime != 0 && cfg.RetryMaxWait != 0:
			derived = append(derived, WithRetry(cfg.RetryCount, cfg.RetryWaitTime, cfg.RetryMaxWait))
		case cfg.RetryWaitTime != 0:
			derived = append(derived, WithRetry(cfg.RetryCount, cfg.RetryWaitTime))
		default:
			derived = append(derived, WithRetry(cfg.RetryCount))
		}
	}
	if cfg.BearerToken != "" {
		derived = append(derived, WithBearerToken(cfg.BearerToken))
	}
	if cfg.APIKey != "" {
		derived = append(derived, WithAPIKey(cfg.APIKey, cfg.APIKeyHeader))
	}
	if cfg.UserAgent != "" {
		derived = append(derived, WithUserAgent(cfg.UserAgent))
	}
	if cfg.TLSInsecure {
		derived = append(derived, WithTLSInsecure())
	}

	return New(append(derived, opts...)...)
}
