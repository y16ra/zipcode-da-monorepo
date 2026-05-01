package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/y16ra/zipcode-da-monorepo/internal/client"
)

var digitsOnly = regexp.MustCompile(`^\d{3,7}$`)

// SearchZipcodeRequest is the JSON body for POST /api/v1/search/zipcode.
type SearchZipcodeRequest struct {
	Zipcode     string `json:"zipcode"`
	Page        *int   `json:"page"`
	Limit       *int   `json:"limit"`
	ChoiKiType  *int   `json:"choikitype"`
	SearchType  *int   `json:"searchtype"`
	ECUID       string `json:"ec_uid"`
}

// SearchZipcodeHandler proxies zipcode search to Japan Post searchcode API.
type SearchZipcodeHandler struct {
	JP           *client.JapanPost
	DefaultECUID string
}

func (h *SearchZipcodeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req SearchZipcodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	code, err := normalizeZipcode(req.Zipcode)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	page := 1
	if req.Page != nil {
		page = *req.Page
		if page < 1 {
			writeJSONError(w, http.StatusBadRequest, "page must be >= 1")
			return
		}
	}

	limit := 1000
	if req.Limit != nil {
		limit = *req.Limit
		if limit < 1 || limit > 1000 {
			writeJSONError(w, http.StatusBadRequest, "limit must be between 1 and 1000")
			return
		}
	}

	opts := client.SearchOpts{
		Page:  page,
		Limit: limit,
	}
	if req.ChoiKiType != nil {
		v := *req.ChoiKiType
		if v != 1 && v != 2 {
			writeJSONError(w, http.StatusBadRequest, "choikitype must be 1 or 2")
			return
		}
		opts.ChoiKiType = v
	}
	if req.SearchType != nil {
		v := *req.SearchType
		if v != 1 && v != 2 {
			writeJSONError(w, http.StatusBadRequest, "searchtype must be 1 or 2")
			return
		}
		opts.SearchType = v
	}
	if req.ECUID != "" {
		opts.ECUID = req.ECUID
	} else if h.DefaultECUID != "" {
		opts.ECUID = h.DefaultECUID
	}

	body, status, err := h.JP.SearchCode(r.Context(), code, opts)
	if err != nil {
		msg := err.Error()
		respCode := http.StatusBadGateway
		var he *client.HTTPStatusError
		if errors.As(err, &he) && he.StatusCode >= 400 && he.StatusCode < 600 {
			respCode = he.StatusCode
		} else if status >= 400 && status < 600 {
			respCode = status
		}
		writeJSONErrorWithDetail(w, respCode, "upstream error", msg)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func normalizeZipcode(raw string) (string, error) {
	s := strings.ReplaceAll(strings.TrimSpace(raw), "-", "")
	if s == "" {
		return "", errors.New("zipcode is required")
	}
	if !digitsOnly.MatchString(s) {
		return "", errors.New("zipcode must be 3-7 digits (hyphens allowed)")
	}
	return s, nil
}

func writeJSONError(w http.ResponseWriter, code int, message string) {
	writeJSONErrorWithDetail(w, code, message, "")
}

func writeJSONErrorWithDetail(w http.ResponseWriter, code int, message, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := map[string]string{"error": message}
	if detail != "" {
		resp["detail"] = detail
	}
	b, _ := json.Marshal(resp)
	_, _ = w.Write(b)
}
