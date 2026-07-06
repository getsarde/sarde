package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// buildAuthedHandler assembles the same middleware chain Start() uses, with a
// known token and a stub mux, so auth behavior is tested without binding a
// real listener or a ProjectManager.
func buildAuthedHandler(token string) http.Handler {
	s := &APIServer{token: token}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	var handler http.Handler = mux
	handler = s.authMiddleware(handler)
	handler = corsMiddleware(handler)
	return handler
}

func TestAuthMiddleware_RejectsMissingToken(t *testing.T) {
	h := buildAuthedHandler("secret-token")
	req := httptest.NewRequest("GET", "/api/config", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", rec.Code)
	}
}

func TestAuthMiddleware_RejectsWrongToken(t *testing.T) {
	h := buildAuthedHandler("secret-token")
	req := httptest.NewRequest("GET", "/api/config", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong token, got %d", rec.Code)
	}
}

func TestAuthMiddleware_AcceptsBearerToken(t *testing.T) {
	h := buildAuthedHandler("secret-token")
	req := httptest.NewRequest("GET", "/api/config", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with valid token, got %d", rec.Code)
	}
}

func TestAuthMiddleware_AcceptsQueryToken(t *testing.T) {
	// WebSocket clients cannot set headers; the token rides a query param.
	h := buildAuthedHandler("secret-token")
	req := httptest.NewRequest("GET", "/api/config?token=secret-token", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 with query token, got %d", rec.Code)
	}
}

func TestAuthMiddleware_HealthIsTokenless(t *testing.T) {
	h := buildAuthedHandler("secret-token")
	req := httptest.NewRequest("GET", "/api/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for tokenless health check, got %d", rec.Code)
	}
}

func TestAuthMiddleware_EmptyServerTokenRejectsAll(t *testing.T) {
	// A zero-value server (Start never ran) must fail closed.
	h := buildAuthedHandler("")
	req := httptest.NewRequest("GET", "/api/config", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when server token is empty, got %d", rec.Code)
	}
}

func TestCORS_ForeignOriginGetsNoCORSHeaders(t *testing.T) {
	h := buildAuthedHandler("secret-token")
	req := httptest.NewRequest("GET", "/api/config", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("foreign origin must not receive Access-Control-Allow-Origin, got %q", got)
	}
}

func TestCORS_AllowedOriginsEchoed(t *testing.T) {
	h := buildAuthedHandler("secret-token")
	for _, origin := range []string{
		"tauri://localhost",
		"https://tauri.localhost",
		"http://localhost:5173",
		"http://127.0.0.1:4727",
	} {
		req := httptest.NewRequest("GET", "/api/config", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Authorization", "Bearer secret-token")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("origin %q: expected echo, got %q", origin, got)
		}
	}
}

func TestSameOriginWS(t *testing.T) {
	cases := []struct {
		name   string
		origin string
		host   string
		want   bool
	}{
		{"empty origin (non-browser)", "", "localhost:4727", true},
		{"exact match", "http://localhost:4727", "localhost:4727", true},
		{"loopback spellings", "http://127.0.0.1:4727", "localhost:4727", true},
		{"lan same-origin", "http://192.168.1.5:4727", "192.168.1.5:4727", true},
		{"foreign page", "https://evil.example.com", "localhost:4727", false},
		{"wrong port", "http://localhost:9999", "localhost:4727", false},
	}
	for _, tc := range cases {
		req := httptest.NewRequest("GET", "http://"+tc.host+"/ws", nil)
		req.Host = tc.host
		if tc.origin != "" {
			req.Header.Set("Origin", tc.origin)
		}
		if got := sameOriginWS(req); got != tc.want {
			t.Errorf("%s: sameOriginWS = %v, want %v", tc.name, got, tc.want)
		}
	}
}
