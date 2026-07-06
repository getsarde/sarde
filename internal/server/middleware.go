package server

import (
	"net/http"
	"net/url"
	"runtime/debug"
	"time"

	"github.com/getsarde/sarde/internal/devlog"
)

// apiAllowedOrigin reports whether a browser Origin may talk to the sidecar
// API: the Tauri webview origins and loopback-hosted frontends. Arbitrary web
// pages are rejected. The bearer token is the primary gate — this restores
// the browser's same-origin protections that a wildcard would disable.
func apiAllowedOrigin(origin string) bool {
	switch origin {
	case "tauri://localhost", "https://tauri.localhost", "http://tauri.localhost":
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// corsMiddleware echoes CORS headers only for allowlisted origins.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && apiAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// recovererMiddleware catches panics and returns a 500 JSON error.
func recovererMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				devlog.Error("server", "PANIC: %v\n%s", err, debug.Stack())
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"success":false,"error":{"code":"INTERNAL_ERROR","message":"internal server error"}}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// loggerMiddleware logs HTTP requests.
func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		devlog.Request(r.Method, r.URL.Path, sw.status, time.Since(start))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}
