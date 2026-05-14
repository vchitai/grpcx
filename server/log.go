package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
)

// InterceptorLogger adapts slog to the grpc-middleware logging.Logger interface.
// Fields named in sensitiveFields are redacted from gRPC request/response payloads.
func InterceptorLogger(l *slog.Logger, sensitiveFields ...string) logging.Logger {
	sensitive := make(map[string]bool, len(sensitiveFields))
	for _, f := range sensitiveFields {
		sensitive[f] = true
	}
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		for i := 0; i+1 < len(fields); i += 2 {
			key, ok := fields[i].(string)
			if !ok {
				continue
			}
			if key == "grpc.request.content" || key == "grpc.response.content" {
				fields[i+1] = redactSensitive(fmt.Sprintf("%v", fields[i+1]), sensitive)
			}
		}
		l.Log(ctx, slogLevel(lvl), msg, fields...)
	})
}

func redactSensitive(data string, sensitive map[string]bool) string {
	parts := strings.Fields(data)
	for i, part := range parts {
		for field := range sensitive {
			if strings.HasPrefix(part, field+":") || strings.HasPrefix(part, field+"=") {
				parts[i] = field + `:"********"`
			}
		}
	}
	return strings.Join(parts, " ")
}

func slogLevel(lvl logging.Level) slog.Level {
	switch lvl {
	case logging.LevelDebug:
		return slog.LevelDebug
	case logging.LevelInfo:
		return slog.LevelInfo
	case logging.LevelWarn:
		return slog.LevelWarn
	case logging.LevelError:
		return slog.LevelError
	default:
		return slog.LevelError
	}
}
