package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/y16ra/zipcode-da-monorepo/internal/config"
)

const (
	userAgent = "zipcode-da-monorepo/1.0"
	tokenSkew   = 60 * time.Second
	httpTimeout = 30 * time.Second
)

// JapanPost is a client for 郵便番号・デジタルアドレス API (token + searchcode).
type JapanPost struct {
	cfg        config.Config
	httpClient *http.Client

	mu           sync.Mutex
	cachedToken  string
	cachedExpiry time.Time
}

// NewJapanPost creates a Japan Post API client.
func NewJapanPost(cfg config.Config) *JapanPost {
	return &JapanPost{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

type tokenRequest struct {
	GrantType  string `json:"grant_type"`
	ClientID   string `json:"client_id"`
	SecretKey  string `json:"secret_key"`
	Scope      string `json:"scope,omitempty"`
}

type tokenResponse struct {
	Scope     string `json:"scope"`
	TokenType string `json:"token_type"`
	ExpiresIn int    `json:"expires_in"`
	Token     string `json:"token"`
}

// apiErrorBody matches Japan Post API error JSON (e.g. 400/401 responses in the reference).
type apiErrorBody struct {
	RequestID string `json:"request_id"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

func summarizeJapanPostError(op string, status int, body []byte) string {
	raw := strings.TrimSpace(string(body))
	var ae apiErrorBody
	if json.Unmarshal(body, &ae) == nil && (ae.ErrorCode != "" || ae.Message != "" || ae.RequestID != "") {
		var b strings.Builder
		fmt.Fprintf(&b, "%s: status %d", op, status)
		if ae.ErrorCode != "" {
			fmt.Fprintf(&b, " [%s]", ae.ErrorCode)
		}
		if ae.Message != "" {
			fmt.Fprintf(&b, " %s", ae.Message)
		}
		if ae.RequestID != "" {
			fmt.Fprintf(&b, " (request_id=%s)", ae.RequestID)
		}
		return b.String()
	}
	return fmt.Sprintf("%s: status %d: %s", op, status, raw)
}

// SearchOpts are optional query parameters for searchcode V2.
type SearchOpts struct {
	Page       int
	Limit      int
	ChoiKiType int
	SearchType int
	ECUID      string
}

// SearchCode calls GET {base}{JAPANPOST_SEARCH_CODE_PATH}/{search_code} with query parameters.
func (c *JapanPost) SearchCode(ctx context.Context, searchCode string, opts SearchOpts) ([]byte, int, error) {
	token, err := c.accessToken(ctx)
	if err != nil {
		return nil, 0, err
	}

	searchPrefix := strings.TrimSuffix(c.cfg.JapanPostSearchCodePath, "/")
	u, err := url.Parse(c.cfg.JapanPostBaseURL + searchPrefix + "/" + url.PathEscape(searchCode))
	if err != nil {
		return nil, 0, err
	}
	q := u.Query()
	if opts.Page > 0 {
		q.Set("page", strconv.Itoa(opts.Page))
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.ChoiKiType > 0 {
		q.Set("choikitype", strconv.Itoa(opts.ChoiKiType))
	}
	if opts.SearchType > 0 {
		q.Set("searchtype", strconv.Itoa(opts.SearchType))
	}
	if opts.ECUID != "" {
		q.Set("ec_uid", opts.ECUID)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Forwarded-For", c.cfg.JapanPostXForwardedFor)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return body, resp.StatusCode, &HTTPStatusError{
			StatusCode: resp.StatusCode,
			Message:    summarizeJapanPostError("japanpost searchcode", resp.StatusCode, body),
		}
	}
	return body, resp.StatusCode, nil
}

// HTTPStatusError carries a non-2xx status from the Japan Post API.
type HTTPStatusError struct {
	StatusCode int
	Message    string
}

func (e *HTTPStatusError) Error() string {
	return e.Message
}

func (c *JapanPost) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cachedToken != "" && time.Until(c.cachedExpiry) > tokenSkew {
		return c.cachedToken, nil
	}

	tok, exp, err := c.fetchToken(ctx)
	if err != nil {
		return "", err
	}
	c.cachedToken = tok
	c.cachedExpiry = exp
	return tok, nil
}

func (c *JapanPost) fetchToken(ctx context.Context) (string, time.Time, error) {
	treq := tokenRequest{
		GrantType: "client_credentials",
		ClientID:  c.cfg.JapanPostClientID,
		SecretKey: c.cfg.JapanPostSecretKey,
	}
	if c.cfg.JapanPostTokenScope != "" {
		treq.Scope = c.cfg.JapanPostTokenScope
	}
	payload, err := json.Marshal(treq)
	if err != nil {
		return "", time.Time{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.JapanPostBaseURL+c.cfg.JapanPostTokenPath, bytes.NewReader(payload))
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", c.cfg.JapanPostXForwardedFor)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", time.Time{}, &HTTPStatusError{
			StatusCode: resp.StatusCode,
			Message:    summarizeJapanPostError("japanpost token", resp.StatusCode, body),
		}
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", time.Time{}, fmt.Errorf("japanpost token: decode: %w", err)
	}
	if tr.Token == "" {
		return "", time.Time{}, fmt.Errorf("japanpost token: empty token in response")
	}
	exp := time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	if tr.ExpiresIn <= 0 {
		exp = time.Now().Add(5 * time.Minute)
	}
	return tr.Token, exp, nil
}
