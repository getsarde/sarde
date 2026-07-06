package server

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/outputpath"
)

// devRequestLogger wraps an HTTP handler to log requests with colored status codes.
func devRequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		purpose := r.Header.Get("Purpose")
		if purpose == "" {
			purpose = r.Header.Get("Sec-Purpose")
		}
		if purpose == "prefetch" && sw.status == http.StatusOK {
			return
		}
		devlog.Request(r.Method, r.URL.Path, sw.status, time.Since(start))
	})
}

// fileHandler returns an HTTP handler that serves static files with clean URL support.
func (ds *DevServer) fileHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Dev server: never let the browser cache build output. Embedded
		// theme/plugin assets keep stable URLs across rebuilds (e.g. the
		// non-fingerprinted sarde.js bundle), so without this a fixed asset
		// can be served stale from disk cache and mask freshly-built changes.
		w.Header().Set("Cache-Control", "no-store")
		r.Header.Del("If-Modified-Since")
		r.Header.Del("If-None-Match")

		urlPath := r.URL.Path

		// Try the exact file first.
		filePath := filepath.Join(ds.outputDir, filepath.FromSlash(urlPath))

		// Clean URL: /foo/ → foo/index.html
		if strings.HasSuffix(urlPath, "/") {
			candidate := filepath.Join(filePath, "index.html")
			if _, err := os.Stat(candidate); err == nil {
				http.ServeFile(w, r, candidate)
				return
			}
		}

		// Clean URL: /foo → foo/index.html (no extension)
		if filepath.Ext(urlPath) == "" && !strings.HasSuffix(urlPath, "/") {
			candidate := filepath.Join(filePath, "index.html")
			if _, err := os.Stat(candidate); err == nil {
				http.ServeFile(w, r, candidate)
				return
			}
		}

		// Exact file exists (skip directories to avoid raw listings).
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			http.ServeFile(w, r, filePath)
			return
		}

		// 404 fallback: try language-specific 404 first (e.g. fr/404.html),
		// then fall back to the root 404.html.
		devlog.Warn("server", "[404] %s → tried %s", urlPath, filePath)
		notFound := ""
		trimmed := strings.TrimPrefix(urlPath, strings.TrimRight(ds.basePath, "/"))
		trimmed = strings.TrimPrefix(trimmed, "/")
		if seg, _, ok := strings.Cut(trimmed, "/"); ok && seg != "" {
			// SafeJoin, not a bare filepath.Join: seg is split on "/" only, so
			// an encoded-backslash segment like "..\..\.." would otherwise
			// survive into the join and escape outputDir on Windows (this read
			// bypasses http.ServeFile's dot-dot guard).
			if langCandidate, err := outputpath.SafeJoin(ds.outputDir, path.Join(seg, consts.Template404)); err == nil {
				if _, err := os.Stat(langCandidate); err == nil {
					notFound = langCandidate
				}
			}
		}
		if notFound == "" {
			root404 := filepath.Join(ds.outputDir, consts.Template404)
			if _, err := os.Stat(root404); err == nil {
				notFound = root404
			}
		}
		if notFound != "" {
			w.WriteHeader(http.StatusNotFound)
			data, _ := os.ReadFile(notFound)
			w.Write(data)
			return
		}

		// Minimal HTML so injectScript can inject the live reload script,
		// enabling the error overlay even when no build output exists.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>404</title></head><body></body></html>`))
	})
}

// injectScript wraps an HTTP handler to inject the live reload script into HTML responses.
func (ds *DevServer) injectScript(next http.Handler) http.Handler {
	script := []byte("\n<script>" + string(embedded.LiveReloadJS) + "</script>\n")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip injection for WebSocket, static assets.
		if r.URL.Path == "/ws" {
			next.ServeHTTP(w, r)
			return
		}
		ext := strings.ToLower(filepath.Ext(r.URL.Path))
		if ext != "" && ext != ".html" && ext != ".htm" {
			next.ServeHTTP(w, r)
			return
		}

		// Buffer the response. Default to 200 so a handler that writes
		// nothing (never calls Write/WriteHeader) doesn't yield WriteHeader(0).
		buf := &bufferedResponseWriter{
			header:     make(http.Header),
			body:       &bytes.Buffer{},
			statusCode: http.StatusOK,
		}
		next.ServeHTTP(buf, r)

		// Copy headers.
		for k, v := range buf.header {
			w.Header()[k] = v
		}

		body := buf.body.Bytes()

		// Inject script before </body> if present. The page's build ID lets
		// the client compare against ReloadSync announcements and reload only
		// when the page actually predates the latest build.
		if idx := bytes.LastIndex(body, []byte("</body>")); idx != -1 {
			buildID := int64(0)
			if ds.hub != nil { // zero-value DevServer in tests has no hub
				buildID = ds.hub.BuildID()
			}
			w.Header().Del("Content-Length")
			w.WriteHeader(buf.statusCode)
			w.Write(body[:idx])
			fmt.Fprintf(w, "\n<script>window.__SARDE_LR_BUILD__=%d;</script>", buildID)
			w.Write(script)
			w.Write(body[idx:])
		} else {
			w.WriteHeader(buf.statusCode)
			w.Write(body)
		}
	})
}

// bufferedResponseWriter captures an HTTP response in memory.
// Follows the http.ResponseWriter contract: first Write implies WriteHeader(200).
type bufferedResponseWriter struct {
	header      http.Header
	body        *bytes.Buffer
	statusCode  int
	wroteHeader bool
}

func (b *bufferedResponseWriter) Header() http.Header {
	return b.header
}

func (b *bufferedResponseWriter) Write(data []byte) (int, error) {
	if !b.wroteHeader {
		b.WriteHeader(http.StatusOK)
	}
	return b.body.Write(data)
}

func (b *bufferedResponseWriter) WriteHeader(statusCode int) {
	if b.wroteHeader {
		return
	}
	b.wroteHeader = true
	b.statusCode = statusCode
}

// ReadFrom implements io.ReaderFrom to support io.Copy from http.ServeFile.
func (b *bufferedResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	if !b.wroteHeader {
		b.WriteHeader(http.StatusOK)
	}
	return io.Copy(b.body, r)
}
