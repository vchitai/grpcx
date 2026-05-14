package i18n_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vchitai/grpcx/i18n"
)

func makeFS(files map[string]string) fs.FS {
	m := make(fstest.MapFS, len(files))
	for name, content := range files {
		m[name] = &fstest.MapFile{Data: []byte(content)}
	}
	return m
}

func TestNew_noOp(t *testing.T) {
	loc := i18n.New("en")
	// unknown key is returned as-is
	assert.Equal(t, "ERR_FOO", loc.Translate("ERR_FOO", nil, "en"))
}

func TestNewFromFS_basic(t *testing.T) {
	fsys := makeFS(map[string]string{
		"en.yaml": "ERR_NOT_FOUND: \"Resource not found.\"\n",
	})
	loc, err := i18n.NewFromFS(fsys, "en")
	require.NoError(t, err)
	assert.Equal(t, "Resource not found.", loc.Translate("ERR_NOT_FOUND", nil, "en"))
}

func TestNewFromFS_templateInterpolation(t *testing.T) {
	fsys := makeFS(map[string]string{
		"en.yaml": "ERR_STOCK: \"Only {{.available}} units available for SKU {{.sku}}.\"\n",
	})
	loc, err := i18n.NewFromFS(fsys, "en")
	require.NoError(t, err)

	msg := loc.Translate("ERR_STOCK", map[string]any{"available": 3, "sku": "ABC"}, "en")
	assert.Equal(t, "Only 3 units available for SKU ABC.", msg)
}

func TestNewFromFS_fallbackLanguage(t *testing.T) {
	fsys := makeFS(map[string]string{
		"en.yaml": "ERR_X: \"English message.\"\n",
	})
	loc, err := i18n.NewFromFS(fsys, "en")
	require.NoError(t, err)

	// vi not loaded → falls back to en
	assert.Equal(t, "English message.", loc.Translate("ERR_X", nil, "vi"))
}

func TestNewFromFS_missingKey(t *testing.T) {
	fsys := makeFS(map[string]string{
		"en.yaml": "ERR_A: \"exists\"\n",
	})
	loc, err := i18n.NewFromFS(fsys, "en")
	require.NoError(t, err)

	assert.Equal(t, "ERR_MISSING", loc.Translate("ERR_MISSING", nil, "en"))
}

func TestNewFromFS_multiLang(t *testing.T) {
	fsys := makeFS(map[string]string{
		"en.yaml": "GREETING: \"Hello\"\n",
		"vi.yaml": "GREETING: \"Xin chào\"\n",
	})
	loc, err := i18n.NewFromFS(fsys, "en")
	require.NoError(t, err)

	assert.Equal(t, "Hello", loc.Translate("GREETING", nil, "en"))
	assert.Equal(t, "Xin chào", loc.Translate("GREETING", nil, "vi"))
}

func TestNewFromFS_ignoresNonYAML(t *testing.T) {
	fsys := makeFS(map[string]string{
		"en.yaml":  "ERR_X: \"ok\"\n",
		"README.md": "# docs",
	})
	_, err := i18n.NewFromFS(fsys, "en")
	require.NoError(t, err)
}
