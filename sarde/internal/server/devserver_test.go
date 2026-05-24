package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileHandler_CleanURL(t *testing.T) {
	// Create temp output directory with clean URL structure.
	tmpDir := t.TempDir()
	docsDir := filepath.Join(tmpDir, "docs", "intro")
	os.MkdirAll(docsDir, 0o755)
	os.WriteFile(filepath.Join(docsDir, "index.html"), []byte("<html>intro</html>"), 0o644)

	ds := &DevServer{outputDir: tmpDir}
	handler := ds.fileHandler()

	tests := []struct {
		name     string
		path     string
		wantCode int
		wantBody string
	}{
		{"trailing slash", "/docs/intro/", 200, "<html>intro</html>"},
		{"no trailing slash", "/docs/intro", 200, "<html>intro</html>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", w.Code, tt.wantCode)
			}
			if !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Errorf("body = %q, want to contain %q", w.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestFileHandler_404Fallback(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "404.html"), []byte("<html>not found</html>"), 0o644)

	ds := &DevServer{outputDir: tmpDir}
	handler := ds.fileHandler()

	req := httptest.NewRequest("GET", "/nonexistent/page/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
	if !strings.Contains(w.Body.String(), "not found") {
		t.Errorf("body = %q, want 404.html content", w.Body.String())
	}
}

func TestInjectScript_HTML(t *testing.T) {
	ds := &DevServer{liveReload: true}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		w.Write([]byte("<html><body><h1>Hello</h1></body></html>"))
	})

	handler := ds.injectScript(inner)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "<script>") {
		t.Error("expected script injection in HTML response")
	}
	if !strings.Contains(body, "</body>") {
		t.Error("expected </body> tag preserved")
	}
	// Script should appear before </body>.
	scriptIdx := strings.Index(body, "<script>")
	bodyIdx := strings.Index(body, "</body>")
	if scriptIdx > bodyIdx {
		t.Error("script should be injected before </body>")
	}
}

func TestInjectScript_SkipsAssets(t *testing.T) {
	ds := &DevServer{liveReload: true}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("body { color: red; }"))
	})

	handler := ds.injectScript(inner)

	tests := []string{"/style.css", "/app.js", "/image.png", "/font.woff2"}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if strings.Contains(w.Body.String(), "<script>") {
				t.Errorf("script should not be injected for %s", path)
			}
		})
	}
}

func TestInjectScript_SkipsWebSocket(t *testing.T) {
	ds := &DevServer{liveReload: true}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ws endpoint"))
	})

	handler := ds.injectScript(inner)

	req := httptest.NewRequest("GET", "/ws", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if strings.Contains(w.Body.String(), "<script>") {
		t.Error("script should not be injected for /ws")
	}
}
