package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/vchitai/grpcx/rpcctx"
	"github.com/vchitai/grpcx/logger"
)

// HTTPServerMiddleware is an HTTP middleware function.
type HTTPServerMiddleware func(http.Handler) http.Handler

// PassedHeaderDeciderFunc returns true if a header should be forwarded to gRPC metadata.
type PassedHeaderDeciderFunc func(string) bool

// createPassingHeaderMiddleware forwards whitelisted headers to gRPC metadata by
// adding the grpc-gateway metadata prefix. The decision is cached per header key.
func createPassingHeaderMiddleware(decide PassedHeaderDeciderFunc) HTTPServerMiddleware {
	return func(next http.Handler) http.Handler {
		cache := new(sync.Map)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			newHeader := make(http.Header, len(r.Header))
			for k, v := range r.Header {
				setHeader(newHeader, k, v)
				if newKey, ok := cache.Load(k); ok {
					setHeader(newHeader, newKey.(string), v)
				} else if decide(k) {
					prefixed := runtime.MetadataHeaderPrefix + k
					cache.Store(k, prefixed)
					setHeader(newHeader, prefixed, v)
				}
			}
			r.Header = newHeader
			next.ServeHTTP(w, r)
		})
	}
}

func setHeader(header http.Header, key string, values []string) {
	for _, v := range values {
		header.Add(key, v)
	}
}

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) Status() int { return rw.status }

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				logger.FromContext(r.Context()).ErrorContext(r.Context(), "panic recovered",
					"trace", string(debug.Stack()),
				)
			}
		}()

		if r.URL.Path == "/metrics" || r.URL.Path == "/health" || r.URL.Path == "/ready" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		wrapped := wrapResponseWriter(w)
		next.ServeHTTP(wrapped, r)

		log := logger.FromContext(r.Context())
		msg := fmt.Sprintf("%s %s %d %dms", r.Method, r.URL.Path, wrapped.Status(), time.Since(start).Milliseconds())
		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"ip", rpcctx.ReadUserIP(r),
			"status", wrapped.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
		}

		switch {
		case wrapped.Status() >= http.StatusInternalServerError:
			log.ErrorContext(r.Context(), msg, attrs...)
		case wrapped.Status() >= http.StatusBadRequest:
			log.WarnContext(r.Context(), msg, attrs...)
		default:
			log.DebugContext(r.Context(), msg, attrs...)
		}
	})
}

// PatchFieldMaskMiddleware automatically derives update_mask from the JSON keys
// present in a PATCH request body, so REST clients don't need to send update_mask
// explicitly. grpc-gateway maps the ?update_mask query param to the proto field.
//
// Example: PATCH /v1/users/123 {"name":"x"} → ?update_mask=name is injected.
func PatchFieldMaskMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			next.ServeHTTP(w, r)
			return
		}

		// Skip if client already provided update_mask.
		if r.URL.Query().Get("update_mask") != "" {
			next.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(body))
		if err != nil || len(body) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(body, &raw); err != nil {
			next.ServeHTTP(w, r)
			return
		}

		paths := make([]string, 0, len(raw))
		for k := range raw {
			paths = append(paths, k)
		}
		sort.Strings(paths) // deterministic order

		q := r.URL.Query()
		q.Set("update_mask", strings.Join(paths, ","))
		r.URL.RawQuery = q.Encode()

		next.ServeHTTP(w, r)
	})
}
