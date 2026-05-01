package config

import (
	"strings"
	"testing"
)

func TestLoad_success(t *testing.T) {
	t.Setenv("JAPANPOST_CLIENT_ID", "cid")
	t.Setenv("JAPANPOST_SECRET_KEY", "secret")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("JAPANPOST_API_BASE_URL", "https://example.jp/api/")
	t.Setenv("JAPANPOST_X_FORWARDED_FOR", "10.0.0.1")
	t.Setenv("JAPANPOST_EC_UID", "ec-1")
	t.Setenv("JAPANPOST_TOKEN_SCOPE", " J1 ")
	t.Setenv("JAPANPOST_TOKEN_PATH", "/custom/token")
	t.Setenv("JAPANPOST_SEARCH_CODE_PATH", "/custom/search")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr = %q, want :9090", cfg.HTTPAddr)
	}
	if cfg.JapanPostBaseURL != "https://example.jp/api" {
		t.Errorf("JapanPostBaseURL = %q", cfg.JapanPostBaseURL)
	}
	if cfg.JapanPostClientID != "cid" || cfg.JapanPostSecretKey != "secret" {
		t.Errorf("credentials not set")
	}
	if cfg.JapanPostXForwardedFor != "10.0.0.1" {
		t.Errorf("JapanPostXForwardedFor = %q", cfg.JapanPostXForwardedFor)
	}
	if cfg.JapanPostECUID != "ec-1" {
		t.Errorf("JapanPostECUID = %q", cfg.JapanPostECUID)
	}
	if cfg.JapanPostTokenScope != "J1" {
		t.Errorf("JapanPostTokenScope = %q, want trimmed J1", cfg.JapanPostTokenScope)
	}
	if cfg.JapanPostTokenPath != "/custom/token" {
		t.Errorf("JapanPostTokenPath = %q", cfg.JapanPostTokenPath)
	}
	if cfg.JapanPostSearchCodePath != "/custom/search" {
		t.Errorf("JapanPostSearchCodePath = %q", cfg.JapanPostSearchCodePath)
	}
}

func TestLoad_defaults(t *testing.T) {
	t.Setenv("JAPANPOST_CLIENT_ID", "cid")
	t.Setenv("JAPANPOST_SECRET_KEY", "secret")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("JAPANPOST_API_BASE_URL", "")
	t.Setenv("JAPANPOST_TOKEN_PATH", "")
	t.Setenv("JAPANPOST_SEARCH_CODE_PATH", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("default HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.JapanPostBaseURL != "https://api.da.pf.japanpost.jp" {
		t.Errorf("default JapanPostBaseURL = %q", cfg.JapanPostBaseURL)
	}
	if cfg.JapanPostTokenPath != "/api/v2/j/token" {
		t.Errorf("default JapanPostTokenPath = %q", cfg.JapanPostTokenPath)
	}
	if cfg.JapanPostSearchCodePath != "/api/v2/searchcode" {
		t.Errorf("default JapanPostSearchCodePath = %q", cfg.JapanPostSearchCodePath)
	}
}

func TestLoad_missingClientID(t *testing.T) {
	t.Setenv("JAPANPOST_CLIENT_ID", "")
	t.Setenv("JAPANPOST_SECRET_KEY", "secret")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "JAPANPOST_CLIENT_ID") {
		t.Fatalf("expected JAPANPOST_CLIENT_ID error, got %v", err)
	}
}

func TestLoad_missingSecret(t *testing.T) {
	t.Setenv("JAPANPOST_CLIENT_ID", "cid")
	t.Setenv("JAPANPOST_SECRET_KEY", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "JAPANPOST_SECRET_KEY") {
		t.Fatalf("expected JAPANPOST_SECRET_KEY error, got %v", err)
	}
}

func TestLoad_tokenPathMustStartWithSlash(t *testing.T) {
	t.Setenv("JAPANPOST_CLIENT_ID", "cid")
	t.Setenv("JAPANPOST_SECRET_KEY", "secret")
	t.Setenv("JAPANPOST_TOKEN_PATH", "no-leading-slash")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "JAPANPOST_TOKEN_PATH") {
		t.Fatalf("expected JAPANPOST_TOKEN_PATH error, got %v", err)
	}
}

func TestLoad_searchPathMustStartWithSlash(t *testing.T) {
	t.Setenv("JAPANPOST_CLIENT_ID", "cid")
	t.Setenv("JAPANPOST_SECRET_KEY", "secret")
	t.Setenv("JAPANPOST_SEARCH_CODE_PATH", "bad")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "JAPANPOST_SEARCH_CODE_PATH") {
		t.Fatalf("expected JAPANPOST_SEARCH_CODE_PATH error, got %v", err)
	}
}
