package middleware

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// TimeoutUnaryServerInterceptor adds a default timeout to RPCs that have no
// client-side deadline. If the context already has a deadline (client sent one),
// it is left untouched.
func TimeoutUnaryServerInterceptor(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if _, ok := ctx.Deadline(); ok {
			// Client already provided a deadline — honour it as-is.
			return handler(ctx, req)
		}
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return handler(ctx, req)
	}
}
