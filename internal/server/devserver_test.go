package server

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/build"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/theme"
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

func TestFileHandler_NoStoreHeader(t *testing.T) {
	// The dev server must mark all build output non-cacheable so stable-named
	// assets (e.g. the non-fingerprinted sarde.js bundle) are never served
	// stale from the browser's disk cache after a rebuild.
	tmpDir := t.TempDir()
	assetsDir := filepath.Join(tmpDir, "assets", "js")
	os.MkdirAll(assetsDir, 0o755)
	os.WriteFile(filepath.Join(assetsDir, "sarde.js"), []byte("console.log('x')"), 0o644)

	ds := &DevServer{outputDir: tmpDir}
	handler := ds.fileHandler()

	cases := []struct {
		name string
		path string
	}{
		{"served asset", "/assets/js/sarde.js"},
		{"missing file", "/does/not/exist.js"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if got := w.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q, want %q", got, "no-store")
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

// --- lifecycle tests (Stop nil-guard, Ready port reporting) ---

// testBuilderFactory returns a BuilderFactory over a minimal real project so
// the dev server's initial build can run during lifecycle tests.
func testBuilderFactory(t *testing.T) (func() *build.SiteBuilder, string) {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "content"), 0o755)
	os.WriteFile(filepath.Join(dir, "sarde.yaml"), []byte("site:\n  title: \"Lifecycle Test\"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "content", "_index.md"), []byte("---\ntitle: Home\n---\n# Home\n"), 0o644)

	thm, _ := theme.LoadFromFS(embedded.ThemeFS(), ".")
	light := theme.ResolveTokens(theme.DefaultTokens(), thm, "", nil)
	light = theme.DeriveTokens(light)
	dark := theme.ResolveDarkTokens(theme.DefaultDarkTokens(), thm, "", nil)
	themeCfg := &engine.ThemeConfig{
		Name:        "default",
		Tokens:      light,
		DarkTokens:  dark,
		DarkEnabled: true,
		StyleTag:    theme.GenerateStyleTag(light, dark),
	}

	cfg := config.Defaults()
	factory := func() *build.SiteBuilder {
		return build.NewSiteBuilder(build.BuildOptions{
			ProjectDir:  dir,
			Config:      cfg,
			ThemeConfig: themeCfg,
			EmbeddedFS:  embedded.ThemeFS(),
		})
	}
	return factory, dir
}

// Stop before Start must not panic (ds.server used to be nil until Start ran).
func TestDevServer_StopBeforeStart_NoPanic(t *testing.T) {
	ds := New(Options{ProjectDir: t.TempDir(), OutputDir: t.TempDir()})
	if err := ds.Stop(); err != nil {
		t.Fatalf("Stop before Start: %v", err)
	}
}

func TestDevServer_StopThenStart_ReadyClosedWithoutPort(t *testing.T) {
	ds := New(Options{ProjectDir: t.TempDir(), OutputDir: t.TempDir()})
	if err := ds.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	err := ds.Start()
	if !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("Start after Stop = %v, want http.ErrServerClosed", err)
	}
	if _, ok := <-ds.Ready(); ok {
		t.Error("Ready should be closed without a port when stopped before serving")
	}
}

// When the requested port is occupied, Ready must report the actually-bound
// port, not the requested one.
func TestDevServer_Ready_ReportsActualPortOnConflict(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	busyPort := busy.Addr().(*net.TCPAddr).Port

	factory, projDir := testBuilderFactory(t)
	ds := New(Options{
		ProjectDir:     projDir,
		OutputDir:      filepath.Join(projDir, "dist"),
		Host:           "127.0.0.1",
		Port:           busyPort,
		BuilderFactory: factory,
	})

	done := make(chan error, 1)
	go func() { done <- ds.Start() }()

	port, ok := <-ds.Ready()
	if !ok {
		t.Fatal("Ready closed without a port")
	}
	if port == busyPort {
		t.Errorf("port = %d, must differ from the occupied port", port)
	}
	if port <= 0 {
		t.Errorf("port = %d, want > 0", port)
	}

	if err := ds.Stop(); err != nil {
		t.Errorf("Stop: %v", err)
	}
	// Join the Start goroutine so TempDir cleanup doesn't race the build.
	if err := <-done; err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Errorf("Start returned %v, want nil or ErrServerClosed", err)
	}
}

func TestOnFileChange_CSSHotSwapSyncsOutputCopy(t *testing.T) {
	// CSS hot-swap skips the rebuild that copies static/ to the output dir,
	// and the file handler serves only from the output dir, so onFileChange
	// must sync the changed file there before broadcasting.
	projectDir := t.TempDir()
	outputDir := filepath.Join(projectDir, "dist")

	src := filepath.Join(projectDir, "static", "css", "style.css")
	os.MkdirAll(filepath.Dir(src), 0o755)
	os.WriteFile(src, []byte("body{color:red}"), 0o644)

	outCopy := filepath.Join(outputDir, "css", "style.css")
	os.MkdirAll(filepath.Dir(outCopy), 0o755)
	os.WriteFile(outCopy, []byte("body{color:blue}"), 0o644)

	ds := &DevServer{projectDir: projectDir, outputDir: outputDir, hub: NewHub()}
	ds.onFileChange([]FileChange{{Path: src, Kind: ChangeCSS}})

	got, err := os.ReadFile(outCopy)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "body{color:red}" {
		t.Errorf("output copy = %q, want the freshly saved CSS", got)
	}
}

func TestSyncStaticFile_RejectsPathsOutsideStatic(t *testing.T) {
	projectDir := t.TempDir()
	ds := &DevServer{projectDir: projectDir, outputDir: filepath.Join(projectDir, "dist")}

	if err := ds.syncStaticFile(filepath.Join(projectDir, "content", "style.css")); err == nil {
		t.Error("expected error for a file outside static/")
	}
}
