package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/y16ra/zipcode-da-monorepo/internal/config"
)

func testConfig(base string) config.Config {
	return config.Config{
		JapanPostBaseURL:        strings.TrimRight(base, "/"),
		JapanPostClientID:       "test-client",
		JapanPostSecretKey:      "test-secret",
		JapanPostXForwardedFor:  "127.0.0.1",
		JapanPostTokenPath:      "/api/v2/j/token",
		JapanPostSearchCodePath: "/api/v2/searchcode",
	}
}

func newTestJapanPost(cfg config.Config, srv *httptest.Server) *JapanPost {
	return &JapanPost{
		cfg:        cfg,
		httpClient: srv.Client(),
	}
}

func TestJapanPost_SearchCode_success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/j/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("token: want POST, got %s", r.Method)
		}
		b, _ := io.ReadAll(r.Body)
		var tr tokenRequest
		if err := json.Unmarshal(b, &tr); err != nil {
			t.Fatalf("token body: %v", err)
		}
		if tr.ClientID != "test-client" || tr.SecretKey != "test-secret" || tr.GrantType != "client_credentials" {
			t.Fatalf("token request fields: %+v", tr)
		}
		if tr.Scope != "J1" {
			t.Fatalf("token scope = %q, want J1", tr.Scope)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"tok-1","expires_in":3600,"token_type":"Bearer"}`))
	})
	mux.HandleFunc("/api/v2/searchcode/1000000", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("search: want GET, got %s", r.Method)
		}
		if ah := r.Header.Get("Authorization"); ah != "Bearer tok-1" {
			t.Errorf("Authorization = %q", ah)
		}
		q := r.URL.Query()
		if q.Get("page") != "2" || q.Get("limit") != "50" || q.Get("choikitype") != "1" || q.Get("searchtype") != "2" || q.Get("ec_uid") != "ec-test" {
			t.Errorf("query = %v", q)
		}
		_, _ = w.Write([]byte(`{"hits":1}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.JapanPostTokenScope = "J1"
	jp := newTestJapanPost(cfg, srv)

	body, status, err := jp.SearchCode(context.Background(), "1000000", SearchOpts{
		Page:       2,
		Limit:      50,
		ChoiKiType: 1,
		SearchType: 2,
		ECUID:      "ec-test",
	})
	if err != nil {
		t.Fatalf("SearchCode: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if string(body) != `{"hits":1}` {
		t.Fatalf("body = %s", body)
	}
}

func TestJapanPost_SearchCode_upstreamErrorJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/j/token", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"token":"t","expires_in":3600}`))
	})
	mux.HandleFunc("/api/v2/searchcode/100", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"request_id":"rid-1","error_code":"E1","message":"bad zip"}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	jp := newTestJapanPost(testConfig(srv.URL), srv)
	_, status, err := jp.SearchCode(context.Background(), "100", SearchOpts{Page: 1, Limit: 10})
	if err == nil {
		t.Fatal("expected error")
	}
	if status != http.StatusBadRequest {
		t.Errorf("status = %d", status)
	}
	var he *HTTPStatusError
	if !errors.As(err, &he) || he.StatusCode != http.StatusBadRequest {
		t.Fatalf("err = %#v", err)
	}
	if !strings.Contains(he.Message, "japanpost searchcode") || !strings.Contains(he.Message, "E1") || !strings.Contains(he.Message, "bad zip") {
		t.Errorf("message = %q", he.Message)
	}
}

func TestJapanPost_accessTokenUsesCache(t *testing.T) {
	tokenPosts := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/j/token", func(w http.ResponseWriter, _ *http.Request) {
		tokenPosts++
		_, _ = w.Write([]byte(`{"token":"cached-t","expires_in":7200}`))
	})
	mux.HandleFunc("/api/v2/searchcode/111", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"a":1}`))
	})
	mux.HandleFunc("/api/v2/searchcode/222", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"b":2}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	jp := newTestJapanPost(testConfig(srv.URL), srv)

	if _, _, err := jp.SearchCode(context.Background(), "111", SearchOpts{Page: 1, Limit: 10}); err != nil {
		t.Fatalf("first SearchCode: %v", err)
	}
	if _, _, err := jp.SearchCode(context.Background(), "222", SearchOpts{Page: 1, Limit: 10}); err != nil {
		t.Fatalf("second SearchCode: %v", err)
	}
	if tokenPosts != 1 {
		t.Fatalf("token POST count = %d, want 1", tokenPosts)
	}
}

func TestJapanPost_fetchToken_invalidJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/j/token", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	jp := newTestJapanPost(testConfig(srv.URL), srv)
	_, _, err := jp.SearchCode(context.Background(), "100", SearchOpts{Page: 1, Limit: 10})
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestJapanPost_fetchToken_emptyToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/j/token", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"token":"","expires_in":100}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	jp := newTestJapanPost(testConfig(srv.URL), srv)
	_, _, err := jp.SearchCode(context.Background(), "100", SearchOpts{Page: 1, Limit: 10})
	if err == nil || !strings.Contains(err.Error(), "empty token") {
		t.Fatalf("expected empty token error, got %v", err)
	}
}

func TestJapanPost_tokenExpiresInZeroUsesFallbackExpiry(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/j/token", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"token":"t","expires_in":0}`))
	})
	mux.HandleFunc("/api/v2/searchcode/100", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	jp := newTestJapanPost(testConfig(srv.URL), srv)
	if _, _, err := jp.SearchCode(context.Background(), "100", SearchOpts{Page: 1, Limit: 10}); err != nil {
		t.Fatalf("SearchCode: %v", err)
	}

	jp.mu.Lock()
	exp := jp.cachedExpiry
	jp.mu.Unlock()
	if !exp.After(time.Now().Add(4 * time.Minute)) {
		t.Fatalf("expected ~5m fallback expiry, got until %s", exp)
	}
}
