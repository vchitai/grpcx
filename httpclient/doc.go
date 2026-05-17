// Package httpclient wraps imroc/req v3 to provide config-driven HTTP client creation.
//
//	// From config struct (YAML/viper compatible):
//	cfg := httpclient.Config{
//	    BaseURL:    "https://api.stripe.com",
//	    Timeout:    30 * time.Second,
//	    RetryCount: 3,
//	    BearerToken: os.Getenv("STRIPE_KEY"),
//	}
//	c := httpclient.NewFromConfig(cfg)
//
//	// Or with functional options:
//	c := httpclient.New(
//	    httpclient.WithBaseURL("https://api.example.com"),
//	    httpclient.WithTimeout(10 * time.Second),
//	    httpclient.WithAPIKey("my-key", "X-API-Key"),
//	)
package httpclient
