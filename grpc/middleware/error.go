package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	sysRuntime "runtime"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vchitai/grpcx/logger"
	"github.com/vchitai/grpcx/rpcctx"
	"github.com/vchitai/grpcx/errs"
)

const (
	debugInfoSkip        = 3
	internalErrorCode    = "INTERNAL_SERVER_ERROR"
	internalErrorMessage = "internal_server_error"
)

// LocalizeMessageFunc resolves an error code to a localized message.
// key is XError.Code (e.g. "ERR_INSUFFICIENT_STOCK"), data is XError.Metadata, lang is the BCP-47 tag.
// Return key unchanged if no translation is found.
type LocalizeMessageFunc func(key string, data map[string]any, lang string) string

type errorWrapperConfig struct {
	localizeMessage LocalizeMessageFunc
	ll              *slog.Logger
	debuggerEnabled bool
}

type ErrorWrapperOption func(o *errorWrapperConfig)

func WithLocalizeMessageFunc(localizeMessage LocalizeMessageFunc) ErrorWrapperOption {
	return func(o *errorWrapperConfig) {
		o.localizeMessage = localizeMessage
	}
}

func WithLogger(ll *slog.Logger) ErrorWrapperOption {
	return func(o *errorWrapperConfig) {
		o.ll = ll
	}
}

func WithDebuggerEnabled(debuggerEnabled bool) ErrorWrapperOption {
	return func(o *errorWrapperConfig) {
		o.debuggerEnabled = debuggerEnabled
	}
}

func ToHTTPError(ctx context.Context, localiseFunc LocalizeMessageFunc, rErr error) error {
	noopLocalize := func(key string, _ map[string]any, _ string) string { return key }
	if localiseFunc == nil {
		localiseFunc = noopLocalize
	}
	cfg := &errorWrapperConfig{
		localizeMessage: localiseFunc,
		ll:              logger.FromContext(ctx),
		debuggerEnabled: false,
	}
	return ConvertGRPCErrorToHTTPError(ctx, cfg, rErr)
}

func ConvertGRPCErrorToHTTPError(ctx context.Context, cfg *errorWrapperConfig, rErr error) error {
	lang := rpcctx.GetLanguageFromContext(ctx)
	if lang == "" {
		lang = "en"
	}
	ll := cfg.ll

	fallbackErr := errs.New(codes.Unknown, internalErrorCode, internalErrorMessage)
	if localized := cfg.localizeMessage(fallbackErr.Code, fallbackErr.Metadata, lang); localized != fallbackErr.Code {
		fallbackErr.Message = localized
	}
	fallback := fallbackErr.GRPCStatus()

	var (
		xErr *errs.Error
		stt  *status.Status
		err  error
	)

	if errors.As(rErr, &xErr) {
		if localized := cfg.localizeMessage(xErr.Code, xErr.Metadata, lang); localized != xErr.Code {
			xErr.Message = localized
		}
		stt = xErr.GRPCStatus()
	} else if s, ok := status.FromError(rErr); ok && s.Code() != codes.Unknown {
		stt = s
	} else if cfg.debuggerEnabled {
		stt = fallback
		stt, err = stt.WithDetails(&errdetails.DebugInfo{
			StackEntries: GetStackEntries(debugInfoSkip),
			Detail:       rErr.Error(),
		})
		if err != nil {
			ll.ErrorContext(ctx, "while adding debug details to error", "err", err)
			stt = fallback
		}
	} else {
		stt = fallback
		ll.ErrorContext(ctx, "unhandled error",
			"status", runtime.HTTPStatusFromCode(stt.Code()),
			"err", rErr,
		)
	}

	if stt.Code() == codes.Internal {
		stt = fallback
	}

	return stt.Err()
}

// ErrorWrapperUnaryServerInterceptor applies gRPC error handling and i18n localization.
func ErrorWrapperUnaryServerInterceptor(opts ...ErrorWrapperOption) grpc.UnaryServerInterceptor {
	cfg := &errorWrapperConfig{
		localizeMessage: func(key string, _ map[string]any, _ string) string { return key },
		ll:              slog.Default(),
		debuggerEnabled: false,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, rErr error) {
		resp, rErr = handler(ctx, req)
		if rErr != nil {
			return nil, ConvertGRPCErrorToHTTPError(ctx, cfg, rErr)
		}
		return resp, nil
	}
}

// GetStackEntries returns a stack trace as []string for DebugInfo details.
func GetStackEntries(skip int) []string {
	var pcs [32]uintptr
	n := sysRuntime.Callers(skip, pcs[:])
	frames := sysRuntime.CallersFrames(pcs[:n])

	var entries []string
	for {
		frame, more := frames.Next()
		entries = append(entries, fmt.Sprintf("%s:%d", frame.File, frame.Line))
		if !more {
			break
		}
	}
	return entries
}
