package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	prettyTS  = lipgloss.NewStyle().Faint(true)
	prettyMsg = lipgloss.NewStyle()
	prettyKey = lipgloss.NewStyle().Faint(true).Italic(true)
	prettyVal = lipgloss.NewStyle()

	prettyErrKey = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Italic(true)
	prettyErrVal = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	prettyLevels = map[slog.Level]string{
		slog.LevelDebug: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")).Render("DBG"),
		slog.LevelInfo:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("35")).Render("INF"),
		slog.LevelWarn:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render("WRN"),
		slog.LevelError: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")).Render("ERR"),
	}
	prettyFatal = lipgloss.NewStyle().Bold(true).
			Background(lipgloss.Color("196")).Foreground(lipgloss.Color("15")).Render("FTL")
)

func renderLevel(l slog.Level) string {
	if s, ok := prettyLevels[l]; ok {
		return s
	}
	if l > slog.LevelError {
		return prettyFatal
	}
	return l.String()
}

// prettyHandler is a pino-pretty-inspired slog.Handler.
// Line 1: HH:MM:SS.mmm LEVEL message
// Following lines: 4-space-indented "key: value" pairs.
// Values that look like JSON are pretty-printed with indentation below the key.
type prettyHandler struct {
	mu     sync.Mutex
	w      io.Writer
	level  slog.Level
	attrs  []slog.Attr
	groups []string
}

func newPrettyHandler(w io.Writer, level slog.Level) *prettyHandler {
	return &prettyHandler{w: w, level: level}
}

func (h *prettyHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level
}

func (h *prettyHandler) Handle(_ context.Context, r slog.Record) error {
	var buf bytes.Buffer

	ts := r.Time.Format("15:04:05.000")
	buf.WriteString(prettyTS.Render(ts))
	buf.WriteByte(' ')
	buf.WriteString(renderLevel(r.Level))
	buf.WriteByte(' ')
	buf.WriteString(prettyMsg.Render(r.Message))
	buf.WriteByte('\n')

	for _, a := range h.attrs {
		writeAttr(&buf, h.groups, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		writeAttr(&buf, h.groups, a)
		return true
	})

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf.Bytes())
	return err
}

func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	merged := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(merged, h.attrs)
	copy(merged[len(h.attrs):], attrs)
	return &prettyHandler{
		w:      h.w,
		level:  h.level,
		attrs:  merged,
		groups: append([]string{}, h.groups...),
	}
}

func (h *prettyHandler) WithGroup(name string) slog.Handler {
	return &prettyHandler{
		w:      h.w,
		level:  h.level,
		attrs:  append([]slog.Attr{}, h.attrs...),
		groups: append(append([]string{}, h.groups...), name),
	}
}

func writeAttr(buf *bytes.Buffer, groups []string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}

	if a.Value.Kind() == slog.KindGroup {
		prefix := groups
		if a.Key != "" {
			prefix = append(append([]string{}, groups...), a.Key)
		}
		for _, ga := range a.Value.Group() {
			writeAttr(buf, prefix, ga)
		}
		return
	}

	key := a.Key
	if len(groups) > 0 {
		key = strings.Join(groups, ".") + "." + key
	}

	isErr := key == "err" || key == "error"
	ks, vs := prettyKey, prettyVal
	if isErr {
		ks, vs = prettyErrKey, prettyErrVal
	}

	val := fmtValue(a.Value)

	if looksLikeJSON(val) {
		var indented bytes.Buffer
		if err := json.Indent(&indented, []byte(val), "", "  "); err == nil {
			val = indented.String()
		}
	}

	lines := strings.Split(val, "\n")
	fmt.Fprintf(buf, "    %s: %s\n", ks.Render(key), vs.Render(lines[0]))
	for _, line := range lines[1:] {
		fmt.Fprintf(buf, "    %s\n", vs.Render(line))
	}
}

func fmtValue(v slog.Value) string {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindBool:
		if v.Bool() {
			return "true"
		}
		return "false"
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindFloat64:
		return fmt.Sprintf("%g", v.Float64())
	case slog.KindInt64:
		return fmt.Sprintf("%d", v.Int64())
	case slog.KindUint64:
		return fmt.Sprintf("%d", v.Uint64())
	case slog.KindTime:
		return v.Time().Format(time.RFC3339)
	case slog.KindAny:
		if s, ok := v.Any().(fmt.Stringer); ok {
			return s.String()
		}
		return fmt.Sprintf("%+v", v.Any())
	default:
		return fmt.Sprintf("%+v", v.Any())
	}
}

func looksLikeJSON(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 || (s[0] != '{' && s[0] != '[') {
		return false
	}
	return json.Valid([]byte(s))
}
