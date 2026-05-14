// Package logger provides a slog-based structured logger and context helpers.
//
// [New] creates a *slog.Logger configured from environment variables:
//
//	LOG_LEVEL=debug|info|warn|error  (default: info)
//	LOG_ENCODER=json|console         (default: auto — console when stderr is a TTY, JSON otherwise)
//
// Inject a logger into a context with [WithContext] and retrieve it with [FromContext].
// [FromContext] falls back to slog.Default() so it is always safe to call.
package logger

import (
	"log/slog"
	"os"
	"strings"
	"time"

	charmlog "github.com/charmbracelet/log"
)

// New returns a *slog.Logger configured from environment variables.
// When stderr is a TTY (or LOG_ENCODER=console), it uses a colored human-readable format.
// Set LOG_ENCODER=json to force JSON even on a TTY.
func New() *slog.Logger {
	level := parseLevel(os.Getenv("LOG_LEVEL"))
	enc := strings.ToLower(os.Getenv("LOG_ENCODER"))

	useConsole := enc == "console" || (enc != "json" && isTerminal(os.Stderr))

	var handler slog.Handler
	if useConsole {
		handler = charmlog.NewWithOptions(os.Stderr, charmlog.Options{
			Level:           charmlog.Level(level),
			ReportTimestamp: true,
			TimeFormat:      time.TimeOnly,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	}

	return slog.New(handler)
}

// isTerminal reports whether f is connected to an interactive terminal.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func parseLevel(s string) slog.Level {
	var l slog.Level
	_ = l.UnmarshalText([]byte(s))
	return l
}
