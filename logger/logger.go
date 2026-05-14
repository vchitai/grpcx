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
	"time"

	charmlog "github.com/charmbracelet/log"
	"github.com/charmbracelet/lipgloss"
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
		handler = newConsoleHandler(level)
	}

	return slog.New(handler)
}

// newConsoleHandler returns a charmbracelet/log handler with custom styles.
func newConsoleHandler(level slog.Level) slog.Handler {
	h := charmlog.NewWithOptions(os.Stderr, charmlog.Options{
		Level:           charmlog.Level(level),
		ReportTimestamp: true,
		TimeFormat:      time.TimeOnly,
	})
	h.SetStyles(consoleStyles())
	return h
}

// consoleStyles returns a clean, readable style set for terminal output.
func consoleStyles() *charmlog.Styles {
	s := charmlog.DefaultStyles()

	s.Timestamp = lipgloss.NewStyle().Faint(true)
	s.Separator = lipgloss.NewStyle().Faint(true)
	s.Key = lipgloss.NewStyle().Faint(true).Italic(true)
	s.Value = lipgloss.NewStyle()
	s.Message = lipgloss.NewStyle()

	s.Levels = map[charmlog.Level]lipgloss.Style{
		charmlog.DebugLevel: lipgloss.NewStyle().
			SetString("DBG").
			Bold(true).
			Foreground(lipgloss.Color("63")), // slate blue
		charmlog.InfoLevel: lipgloss.NewStyle().
			SetString("INF").
			Bold(true).
			Foreground(lipgloss.Color("35")), // green
		charmlog.WarnLevel: lipgloss.NewStyle().
			SetString("WRN").
			Bold(true).
			Foreground(lipgloss.Color("214")), // amber
		charmlog.ErrorLevel: lipgloss.NewStyle().
			SetString("ERR").
			Bold(true).
			Foreground(lipgloss.Color("196")), // red
		charmlog.FatalLevel: lipgloss.NewStyle().
			SetString("FTL").
			Bold(true).
			Background(lipgloss.Color("196")).
			Foreground(lipgloss.Color("15")), // white on red
	}

	// Highlight the "err" key in red so errors stand out.
	s.Keys["err"] = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Italic(true)
	s.Values["err"] = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	return s
}

func parseLevel(s string) slog.Level {
	var l slog.Level
	_ = l.UnmarshalText([]byte(s))
	return l
}
