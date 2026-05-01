package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorsMiddleware_preflight(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGIN", "https://app.example")
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("next must not run for OPTIONS")
	})
	h := corsMiddleware(next)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/search/zipcode", nil)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "https://app.example" {
		t.Errorf("Allow-Origin = %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}
	if rr.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("missing Allow-Methods")
	}
}

func TestCorsMiddleware_defaultOriginStar(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGIN", "")
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	h := corsMiddleware(next)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api", nil)
	h.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Allow-Origin = %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}
	if rr.Code != http.StatusTeapot {
		t.Fatalf("next status = %d", rr.Code)
	}
}
