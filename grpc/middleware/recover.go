package middleware

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vchitai/grpcx/logger"
)

func RecoverHandler() recovery.RecoveryHandlerFuncContext {
	return func(ctx context.Context, p any) error {
		logger.FromContext(ctx).ErrorContext(ctx, "PANIC",
			"panic", fmt.Sprintf("%v", p),
			"stack", string(debug.Stack()),
		)
		return status.Errorf(codes.Internal, "%v", p)
	}
}
