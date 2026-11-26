package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A body over maxRequestBody must fail the handler's decode instead of being
// read in full; a body under the limit passes through untouched.
func TestBodyLimitMiddleware(t *testing.T) {
	var read int64
	h := bodyLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var v map[string]string
		err := json.NewDecoder(r.Body).Decode(&v)
		n, _ := io.Copy(io.Discard, r.Body)
		read = n
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	small := httptest.NewRecorder()
	h.ServeHTTP(small, httptest.NewRequest("POST", "/api/x", strings.NewReader(`{"a":"b"}`)))
	if small.Code != http.StatusOK {
		t.Errorf("small body: got %d, want 200", small.Code)
	}

	big := `{"a":"` + strings.Repeat("x", maxRequestBody+1024) + `"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/x", strings.NewReader(big)))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("oversized body: got %d, want 400", rec.Code)
	}
	if read != 0 {
		t.Errorf("oversized body: handler read %d more bytes past the limit", read)
	}
}

func TestServersSetTimeouts(t *testing.T) {
	ds := New(Options{})
	if ds.server.ReadHeaderTimeout != readHeaderTimeout || ds.server.IdleTimeout != idleTimeout {
		t.Errorf("dev server timeouts = %v, %v", ds.server.ReadHeaderTimeout, ds.server.IdleTimeout)
	}
	// No write timeout: live reload holds its connection open.
	if ds.server.WriteTimeout != 0 {
		t.Errorf("dev server WriteTimeout = %v, want 0", ds.server.WriteTimeout)
	}
}
