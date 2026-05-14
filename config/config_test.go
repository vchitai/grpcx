package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vchitai/grpcx/config"
)

type testCfg struct {
	App struct {
		Name string `mapstructure:"name"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"app"`
	DB struct {
		Host string `mapstructure:"host"`
	} `mapstructure:"db"`
}

var defaultYAML = []byte(`
app:
  name: myapp
  port: 8080
db:
  host: localhost
`)

func TestLoad_defaults(t *testing.T) {
	cfg, err := config.Load[testCfg](defaultYAML)
	require.NoError(t, err)
	assert.Equal(t, "myapp", cfg.App.Name)
	assert.Equal(t, 8080, cfg.App.Port)
	assert.Equal(t, "localhost", cfg.DB.Host)
}

func TestLoad_envOverride(t *testing.T) {
	t.Setenv("APP__NAME", "overridden")
	t.Setenv("DB__HOST", "db.prod")

	cfg, err := config.Load[testCfg](defaultYAML)
	require.NoError(t, err)
	assert.Equal(t, "overridden", cfg.App.Name)
	assert.Equal(t, "db.prod", cfg.DB.Host)
	assert.Equal(t, 8080, cfg.App.Port) // unchanged
}

func TestLoad_envPortOverride(t *testing.T) {
	t.Setenv("APP__PORT", "9090")

	cfg, err := config.Load[testCfg](defaultYAML)
	require.NoError(t, err)
	assert.Equal(t, 9090, cfg.App.Port)
}

func TestLoad_invalidYAML(t *testing.T) {
	_, err := config.Load[testCfg]([]byte("{\tinvalid"))
	require.Error(t, err)
}

func TestMustLoad_panicsOnError(t *testing.T) {
	assert.Panics(t, func() {
		config.MustLoad[testCfg]([]byte("{\tinvalid"))
	})
}

func TestMustLoad_success(t *testing.T) {
	assert.NotPanics(t, func() {
		cfg := config.MustLoad[testCfg](defaultYAML)
		assert.Equal(t, "myapp", cfg.App.Name)
	})
}
