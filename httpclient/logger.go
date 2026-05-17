package httpclient

import (
	"fmt"
	"log/slog"
)

// slogLogger bridges req's Logger interface to slog.Default().
type slogLogger struct{}

// Errorf implements req's Logger interface by delegating to slog.Default().Error.
func (l *slogLogger) Errorf(format string, v ...any) {
	slog.Default().Error(fmt.Sprintf(format, v...))
}

// Warnf implements req's Logger interface by delegating to slog.Default().Warn.
func (l *slogLogger) Warnf(format string, v ...any) {
	slog.Default().Warn(fmt.Sprintf(format, v...))
}

// Debugf implements req's Logger interface by delegating to slog.Default().Debug.
func (l *slogLogger) Debugf(format string, v ...any) {
	slog.Default().Debug(fmt.Sprintf(format, v...))
}
