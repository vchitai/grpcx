// Package ratelimit provides gRPC unary interceptors for rate limiting.
// The Limiter interface is intentionally simple so callers can swap in
// Redis-backed, distributed, or other implementations.  The built-in
// [TokenBucketLimiter] is in-memory and suitable for single-instance
// deployments.
//
// Rate limiting is an opt-in, app-level concern — it is NOT wired into the
// default server interceptor chain.  Applications add it explicitly:
//
//	limiter := ratelimit.NewTokenBucketLimiter(100, 200)
//	server.WithGrpcServerUnaryInterceptors(
//	    ratelimit.NewUnaryServerInterceptor(ratelimit.KeyByIP(), limiter),
//	)
package ratelimit

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/vchitai/grpcx/auth"
)

// Limiter decides whether a request is allowed.
type Limiter interface {
	// Allow returns true if the request identified by key should be processed.
	// Returning false causes the interceptor to reply with ResourceExhausted.
	Allow(key string) bool
}

// KeyFunc extracts a rate-limit key from the gRPC request context and method
// info.  Return an empty string to skip rate limiting for this request.
type KeyFunc func(ctx context.Context, info *grpc.UnaryServerInfo) string

// NewUnaryServerInterceptor returns a gRPC unary interceptor that rate-limits
// requests.  Requests exceeding the limit receive a ResourceExhausted error.
// If keyFn returns an empty string the request is allowed through unconditionally.
func NewUnaryServerInterceptor(keyFn KeyFunc, limiter Limiter) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		key := keyFn(ctx, info)
		if key == "" {
			return handler(ctx, req)
		}
		if !limiter.Allow(key) {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}

// KeyByUserID returns a KeyFunc that uses the authenticated user's ID as the
// key.  If no valid claims are present it falls back to the gRPC peer address.
func KeyByUserID() KeyFunc {
	return func(ctx context.Context, _ *grpc.UnaryServerInfo) string {
		if claims, ok := auth.ClaimsFromContext(ctx); ok && claims.UserID != "" {
			return "user:" + claims.UserID
		}
		return peerAddr(ctx)
	}
}

// KeyByIP returns a KeyFunc that uses the client's IP address (from gRPC peer
// info) as the rate-limit key.
func KeyByIP() KeyFunc {
	return func(ctx context.Context, _ *grpc.UnaryServerInfo) string {
		return peerAddr(ctx)
	}
}

// KeyByMethod returns a KeyFunc that uses the full gRPC method name as the key,
// implementing a global per-method rate limit shared across all callers.
func KeyByMethod() KeyFunc {
	return func(_ context.Context, info *grpc.UnaryServerInfo) string {
		return "method:" + info.FullMethod
	}
}

// peerAddr extracts the remote address from gRPC peer info.
// Returns an empty string when peer info is unavailable.
func peerAddr(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}
	return ""
}
