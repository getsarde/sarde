package server

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression test: the 404 fallback used a bare filepath.Join on the first
// URL segment and served the result via os.ReadFile (bypassing
// http.ServeFile's dot-dot guard). On Windows a backslash-encoded segment
// like "..%5C..%5C.." escaped the output dir and disclosed any file named
// 404.html outside it.
func TestFileHandler_404FallbackTraversalBlocked(t *testing.T) {
	parent := t.TempDir()

	// Output dir is a child; the secret 404.html lives OUTSIDE it.
	outDir := filepath.Join(parent, "site", "dist")
	os.MkdirAll(outDir, 0o755)
	secret := "TOP-SECRET-OUTSIDE-FILE"
	os.WriteFile(filepath.Join(parent, "404.html"), []byte(secret), 0o644)

	ds := &DevServer{outputDir: outDir}
	handler := ds.fileHandler()

	// Each first-segment payload targets parent/404.html via traversal.
	for _, seg := range []string{
		`..%5C..`, // encoded backslashes, decoded to ..\.. by the mux
		`..\..`,
		`../..`,
	} {
		req := httptest.NewRequest("GET", "/"+seg+"/whatever", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if strings.Contains(w.Body.String(), secret) {
			t.Errorf("segment %q disclosed a file outside the output dir", seg)
		}
	}
}

// The legitimate language-specific 404 fallback must keep working.
func TestFileHandler_404FallbackLangSpecific(t *testing.T) {
	outDir := t.TempDir()
	frDir := filepath.Join(outDir, "fr")
	os.MkdirAll(frDir, 0o755)
	os.WriteFile(filepath.Join(frDir, "404.html"), []byte("page introuvable"), 0o644)

	ds := &DevServer{outputDir: outDir}
	handler := ds.fileHandler()

	req := httptest.NewRequest("GET", "/fr/missing-page/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
	if !strings.Contains(w.Body.String(), "page introuvable") {
		t.Errorf("language 404 page not served, body: %q", w.Body.String())
	}
}
