// Package config provides a generic YAML + environment variable configuration loader.
//
// Call [MustLoad] (or [Load]) with an embedded YAML byte slice to obtain a populated
// config struct. Environment variables override YAML values using __ as the key
// separator, matching the dot-separated YAML path:
//
//	PG__HOST=db.prod.internal   overrides   pg.host
//	SERVER__GRPC__PORT=9443     overrides   server.grpc.port
//
// Typical usage:
//
//	//go:embed config.yaml
//	var defaultConfig []byte
//
//	type Config struct {
//	    Environment string `mapstructure:"environment"`
//	    Server      struct {
//	        GRPC server.Listen `mapstructure:"grpc"`
//	    } `mapstructure:"server"`
//	}
//
//	cfg := config.MustLoad[Config](defaultConfig)
package config

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// MustLoad unmarshals defaultYAML + env overrides into T. Panics on error.
// Env vars override using __ as separator (e.g. PG__HOST overrides pg.host).
func MustLoad[T any](defaultYAML []byte) *T {
	cfg, err := Load[T](defaultYAML)
	if err != nil {
		panic(fmt.Sprintf("config: failed to load: %v", err))
	}
	return cfg
}

// Load unmarshals defaultYAML + env overrides into T.
func Load[T any](defaultYAML []byte) (*T, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBuffer(defaultYAML)); err != nil {
		return nil, fmt.Errorf("read default config: %w", err)
	}
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	v.AutomaticEnv()
	cfg := new(T)
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return cfg, nil
}
