// Package logger provides a slog-based structured logger and context helpers.
//
// [New] creates a *slog.Logger configured from environment variables:
//
//	LOG_LEVEL=debug|info|warn|error  (default: info)
//	LOG_ENCODER=json|console         (default: console)
//
// Set LOG_ENCODER=json (or CI=true) for machine-readable output in CI / production.
//
// Inject a logger into a context with [WithContext] and retrieve it with [FromContext].
// [FromContext] falls back to slog.Default() so it is always safe to call.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a *slog.Logger configured from environment variables.
// Console (colored, human-readable) is the default.
// Set LOG_ENCODER=json or CI=true to switch to structured JSON output.
func New() *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	enc := strings.ToLower(os.Getenv("LOG_ENCODER"))

	useJSON := enc == "json" || os.Getenv("CI") != ""

	var handler slog.Handler
	if useJSON {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	} else {
		handler = newPrettyHandler(os.Stderr, level)
	}

	return slog.New(handler)
}

func parseLevel(s string) slog.Level {
	var l slog.Level
	_ = l.UnmarshalText([]byte(s))
	return l
}
