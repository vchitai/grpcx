// Package i18n provides server-side localization for gRPC error codes.
// Translation keys are application error codes (e.g. "ERR_OUT_OF_STOCK").
// Values support Go text/template interpolation against error metadata.
//
// Typical usage:
//
//	//go:embed translations/*.yaml
//	var translationFiles embed.FS
//
//	loc, err := i18n.NewFromFS(translationFiles, "en")
//	// pass loc.Translate to middleware.WithLocalizeMessageFunc
package i18n

import (
	"bytes"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// Localizer loads YAML translation files and resolves error codes to localized messages.
type Localizer struct {
	translations map[string]map[string]string // lang → code → template string
	fallback     string
}

// New returns a no-op Localizer with no translations loaded.
// Translate returns the key unchanged. Useful as a safe default or for testing.
func New(fallback string) *Localizer {
	return &Localizer{
		translations: make(map[string]map[string]string),
		fallback:     fallback,
	}
}

// NewFromFS loads translation YAML files from fsys.
// Each file must be named <lang>.yaml (e.g. "en.yaml", "vi.yaml").
// fallback is the language used when the requested language has no translation.
func NewFromFS(fsys fs.FS, fallback string) (*Localizer, error) {
	l := &Localizer{
		translations: make(map[string]map[string]string),
		fallback:     fallback,
	}
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		lang := strings.TrimSuffix(name, ext)
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, err
		}
		var m map[string]string
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		l.translations[lang] = m
	}
	return l, nil
}

// Translate resolves key in the given lang, interpolating data into the template.
// Returns key unchanged if no translation is found (safe fallback for i18n keys).
func (l *Localizer) Translate(key string, data map[string]any, lang string) string {
	msg := l.lookup(key, lang)
	if msg == "" {
		return key
	}
	if len(data) == 0 {
		return msg
	}
	tmpl, err := template.New("").Parse(msg)
	if err != nil {
		return msg
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return msg
	}
	return buf.String()
}

func (l *Localizer) lookup(key, lang string) string {
	if m, ok := l.translations[lang]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	if lang != l.fallback {
		if m, ok := l.translations[l.fallback]; ok {
			return m[key]
		}
	}
	return ""
}
