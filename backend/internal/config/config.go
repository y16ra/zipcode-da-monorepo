package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds server and Japan Post API settings.
type Config struct {
	HTTPAddr string

	JapanPostBaseURL       string
	JapanPostClientID      string
	JapanPostSecretKey     string
	JapanPostXForwardedFor string
	JapanPostECUID         string
	// JapanPostTokenScope is sent as JSON "scope" on token requests when non-empty (e.g. J1).
	JapanPostTokenScope string
	// JapanPostTokenPath is the path for token issuance (default /api/v2/j/token per current API reference request samples).
	JapanPostTokenPath string
	// JapanPostSearchCodePath is the path prefix before /{search_code} (default /api/v2/searchcode).
	JapanPostSearchCodePath string
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:               getenv("HTTP_ADDR", ":8080"),
		JapanPostBaseURL:       strings.TrimRight(getenv("JAPANPOST_API_BASE_URL", "https://api.da.pf.japanpost.jp"), "/"),
		JapanPostClientID:      os.Getenv("JAPANPOST_CLIENT_ID"),
		JapanPostSecretKey:     os.Getenv("JAPANPOST_SECRET_KEY"),
		JapanPostXForwardedFor: getenv("JAPANPOST_X_FORWARDED_FOR", "127.0.0.1"),
		JapanPostECUID:         os.Getenv("JAPANPOST_EC_UID"),
		JapanPostTokenScope:    strings.TrimSpace(os.Getenv("JAPANPOST_TOKEN_SCOPE")),
		JapanPostTokenPath:      strings.TrimSpace(getenv("JAPANPOST_TOKEN_PATH", "/api/v2/j/token")),
		JapanPostSearchCodePath: strings.TrimSpace(getenv("JAPANPOST_SEARCH_CODE_PATH", "/api/v2/searchcode")),
	}
	if !strings.HasPrefix(cfg.JapanPostTokenPath, "/") {
		return Config{}, fmt.Errorf("JAPANPOST_TOKEN_PATH must start with /")
	}
	if !strings.HasPrefix(cfg.JapanPostSearchCodePath, "/") {
		return Config{}, fmt.Errorf("JAPANPOST_SEARCH_CODE_PATH must start with /")
	}
	if cfg.JapanPostClientID == "" {
		return Config{}, fmt.Errorf("JAPANPOST_CLIENT_ID is required")
	}
	if cfg.JapanPostSecretKey == "" {
		return Config{}, fmt.Errorf("JAPANPOST_SECRET_KEY is required")
	}
	return cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
