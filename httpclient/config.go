package httpclient

import "time"

// Config is a YAML/mapstructure-compatible configuration for creating an HTTP client.
// Set at most one of BearerToken, APIKey for authentication.
type Config struct {
	BaseURL       string        `yaml:"base_url"         mapstructure:"base_url"`
	Timeout       time.Duration `yaml:"timeout"          mapstructure:"timeout"`
	RetryCount    int           `yaml:"retry_count"      mapstructure:"retry_count"`
	RetryWaitTime time.Duration `yaml:"retry_wait_time"  mapstructure:"retry_wait_time"`
	RetryMaxWait  time.Duration `yaml:"retry_max_wait"   mapstructure:"retry_max_wait"`

	// Auth — set at most one
	BearerToken  string `yaml:"bearer_token"   mapstructure:"bearer_token"`
	APIKey       string `yaml:"api_key"        mapstructure:"api_key"`
	APIKeyHeader string `yaml:"api_key_header" mapstructure:"api_key_header"` // default: "X-API-Key"

	UserAgent   string `yaml:"user_agent"   mapstructure:"user_agent"`
	TLSInsecure bool   `yaml:"tls_insecure" mapstructure:"tls_insecure"`
}
