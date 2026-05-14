// Package logger provides a slog-based structured logger and context helpers.
//
// [New] creates a *slog.Logger configured from environment variables:
//
//	LOG_LEVEL=debug|info|warn|error  (default: info)
//	LOG_ENCODER=console              (default: json, writes to stderr)
//
// Inject a logger into a context with [WithContext] and retrieve it with [FromContext].
// [FromContext] falls back to slog.Default() so it is always safe to call.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a *slog.Logger configured from environment variables:
//
//	LOG_LEVEL:   debug | info | warn | error  (default: info)
//	LOG_ENCODER: console                       (default: json)
func New() *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(os.Getenv("LOG_LEVEL"))}

	var handler slog.Handler
	if strings.EqualFold(os.Getenv("LOG_ENCODER"), "console") {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	return slog.New(handler)
}

func parseLevel(s string) slog.Level {
	var l slog.Level
	_ = l.UnmarshalText([]byte(s))
	return l
}
