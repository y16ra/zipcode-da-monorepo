package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/y16ra/zipcode-da-monorepo/internal/client"
)

type mockZipBackend struct {
	search func(ctx context.Context, zipcode string, opts client.SearchOpts) ([]byte, int, error)
}

func (m *mockZipBackend) SearchCode(ctx context.Context, zipcode string, opts client.SearchOpts) ([]byte, int, error) {
	if m.search == nil {
		return []byte(`{}`), http.StatusOK, nil
	}
	return m.search(ctx, zipcode, opts)
}

func TestSearchZipcodeHandler_methodNotAllowed(t *testing.T) {
	h := &SearchZipcodeHandler{JP: &mockZipBackend{}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/zipcode", nil)

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rr.Code)
	}
}

func TestSearchZipcodeHandler_invalidJSON(t *testing.T) {
	h := &SearchZipcodeHandler{JP: &mockZipBackend{}}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/zipcode", bytes.NewReader([]byte(`{`)))

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestSearchZipcodeHandler_zipcodeValidation(t *testing.T) {
	calls := 0
	h := &SearchZipcodeHandler{
		JP: &mockZipBackend{
			search: func(context.Context, string, client.SearchOpts) ([]byte, int, error) {
				calls++
				return nil, 0, nil
			},
		},
	}

	for name, body := range map[string]string{
		"empty":       `{"zipcode":""}`,
		"non_digit":   `{"zipcode":"abc"}`,
		"too_short":   `{"zipcode":"12"}`,
		"too_long":    `{"zipcode":"12345678"}`,
		"page_zero":   `{"zipcode":"100","page":0}`,
		"limit_zero":  `{"zipcode":"100","limit":0}`,
		"limit_huge":  `{"zipcode":"100","limit":1001}`,
		"choiki_bad":  `{"zipcode":"100","choikitype":3}`,
		"search_bad":  `{"zipcode":"100","searchtype":0}`,
		"search_bad2": `{"zipcode":"100","searchtype":3}`,
	} {
		t.Run(name, func(t *testing.T) {
			calls = 0
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/search/zipcode", stringsReader(body))
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body %q", rr.Code, rr.Body.String())
			}
			if calls != 0 {
				t.Fatalf("backend should not be called")
			}
		})
	}
}

func TestSearchZipcodeHandler_hyphenNormalized(t *testing.T) {
	var gotCode string
	h := &SearchZipcodeHandler{
		JP: &mockZipBackend{
			search: func(_ context.Context, zipcode string, _ client.SearchOpts) ([]byte, int, error) {
				gotCode = zipcode
				return []byte(`{"ok":true}`), http.StatusOK, nil
			},
		},
	}
	rr := httptest.NewRecorder()
	body := `{"zipcode":"100-0001"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/zipcode", stringsReader(body))

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rr.Code, rr.Body.String())
	}
	if gotCode != "1000001" {
		t.Errorf("upstream zipcode = %q, want 1000001", gotCode)
	}
}

func TestSearchZipcodeHandler_optsPassedAndDefaultECUID(t *testing.T) {
	var got client.SearchOpts
	h := &SearchZipcodeHandler{
		JP: &mockZipBackend{
			search: func(_ context.Context, _ string, opts client.SearchOpts) ([]byte, int, error) {
				got = opts
				return []byte(`[]`), http.StatusOK, nil
			},
		},
		DefaultECUID: "default-ec",
	}
	page, limit, c1, s2 := 2, 50, 1, 2
	rr := httptest.NewRecorder()
	reqBody, _ := json.Marshal(map[string]any{
		"zipcode":     "1000000",
		"page":        page,
		"limit":       limit,
		"choikitype":  c1,
		"searchtype":  s2,
		"ec_uid":      "",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/zipcode", bytes.NewReader(reqBody))

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if got.Page != 2 || got.Limit != 50 || got.ChoiKiType != 1 || got.SearchType != 2 {
		t.Errorf("opts = %+v", got)
	}
	if got.ECUID != "default-ec" {
		t.Errorf("ECUID = %q, want default-ec", got.ECUID)
	}
}

func TestSearchZipcodeHandler_requestECUIDOverridesDefault(t *testing.T) {
	var got client.SearchOpts
	h := &SearchZipcodeHandler{
		JP: &mockZipBackend{
			search: func(_ context.Context, _ string, opts client.SearchOpts) ([]byte, int, error) {
				got = opts
				return []byte(`{}`), http.StatusOK, nil
			},
		},
		DefaultECUID: "default-ec",
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/zipcode", stringsReader(`{"zipcode":"100","ec_uid":"req-ec"}`))

	h.ServeHTTP(rr, req)

	if got.ECUID != "req-ec" {
		t.Errorf("ECUID = %q", got.ECUID)
	}
}

func TestSearchZipcodeHandler_upstreamHTTPStatusError(t *testing.T) {
	up := &client.HTTPStatusError{StatusCode: http.StatusNotFound, Message: "nf"}
	h := &SearchZipcodeHandler{
		JP: &mockZipBackend{
			search: func(context.Context, string, client.SearchOpts) ([]byte, int, error) {
				return nil, 0, up
			},
		},
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/zipcode", stringsReader(`{"zipcode":"100"}`))

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body %q", rr.Code, rr.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if resp["error"] != "upstream error" {
		t.Errorf("error field = %q", resp["error"])
	}
}

func TestSearchZipcodeHandler_upstreamPlainErrorUsesStatusFromInt(t *testing.T) {
	h := &SearchZipcodeHandler{
		JP: &mockZipBackend{
			search: func(context.Context, string, client.SearchOpts) ([]byte, int, error) {
				return nil, http.StatusServiceUnavailable, errors.New("boom")
			},
		},
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/zipcode", stringsReader(`{"zipcode":"100"}`))

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", rr.Code)
	}
}

func stringsReader(s string) io.Reader {
	return bytes.NewReader([]byte(s))
}
