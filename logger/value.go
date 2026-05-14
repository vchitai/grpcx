package logger

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// jsonValue wraps any value and renders it as compact JSON when logged.
// It implements both fmt.Stringer (for console handlers) and slog.LogValuer
// (for structured JSON handlers), so it works correctly with any slog handler.
type jsonValue struct{ v any }

// String is called by console log handlers (e.g. charmbracelet/log) that use fmt.Sprintf.
func (j jsonValue) String() string {
	b, err := json.Marshal(j.v)
	if err != nil {
		return fmt.Sprintf("%+v", j.v)
	}
	return string(b)
}

// LogValue is called by structured slog handlers (e.g. slog.JSONHandler).
func (j jsonValue) LogValue() slog.Value {
	return slog.StringValue(j.String())
}

// JSON wraps v so that it logs as compact JSON instead of a raw Go value.
//
// Works with any slog handler — console (charmbracelet) and JSON alike:
//
//	slog.Info("config loaded", "cfg", logger.JSON(cfg))
//	slog.Info("items", "list", logger.JSON(items))
func JSON(v any) jsonValue {
	return jsonValue{v: v}
}
