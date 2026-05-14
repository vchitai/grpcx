package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/vchitai/grpcx/rpcctx"
	"github.com/vchitai/grpcx/logger"
)

func WithHTTPMethod() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("Grpc-Metadata-"+rpcctx.MdKeyHTTPMethod, r.Method)
			next.ServeHTTP(w, r)
		})
	}
}

func WithRequestID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get(rpcctx.HeaderRequestID)
			if reqID == "" {
				reqID = uuid.NewString()
			}
			w.Header().Set("Grpc-Metadata-"+rpcctx.HeaderRequestID, reqID)
			next.ServeHTTP(w, r)
		})
	}
}

// RequestIDUnaryServerInterceptor extracts or generates a request ID, then
// injects an enriched logger (with request_id + method) into the context so
// all downstream layers can call logger.FromContext(ctx).
func RequestIDUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		reqID := rpcctx.GetRequestIDFromContext(ctx)
		if reqID == "" {
			reqID = uuid.NewString()
		}
		ctx = rpcctx.AttachRequestIDToIncomingContext(ctx, reqID)
		if err := grpc.SetHeader(ctx, metadata.Pairs(strings.ToLower(rpcctx.HeaderRequestID), reqID)); err != nil {
			slog.ErrorContext(ctx, "failed to set request id header", "err", err)
		}

		l := logger.FromContext(ctx).With(
			slog.String("request_id", reqID),
			slog.String("method", info.FullMethod),
		)
		ctx = logger.WithContext(ctx, l)

		return handler(ctx, req)
	}
}
