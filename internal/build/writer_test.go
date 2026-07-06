package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPageOutputPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/", "index.html"},
		{"/docs/intro/", "docs/intro/index.html"},
		{"/blog/hello-world/", "blog/hello-world/index.html"},
		{"/about/", "about/index.html"},
		{"/404.html", "404.html"},
	}
	for _, tt := range tests {
		got := PageOutputPath(tt.input)
		if got != tt.want {
			t.Errorf("PageOutputPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestWriter_WritePages(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "dist")
	w := &Writer{OutputDir: outDir, ProjectDir: t.TempDir()}

	pages := []RenderedPage{
		{OutPath: "index.html", HTML: []byte("<html>Home</html>")},
		{OutPath: "docs/intro/index.html", HTML: []byte("<html>Intro</html>")},
	}

	if _, err := w.Write(pages, nil); err != nil {
		t.Fatal(err)
	}

	// Verify files exist.
	assertFileContains(t, filepath.Join(outDir, "index.html"), "Home")
	assertFileContains(t, filepath.Join(outDir, "docs", "intro", "index.html"), "Intro")
}

func TestWriter_Aliases(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "dist")
	w := &Writer{OutputDir: outDir, ProjectDir: t.TempDir()}

	aliases := map[string]string{
		"/old-path/": "/docs/new-path/",
	}

	if _, err := w.Write(nil, aliases); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, filepath.Join(outDir, "old-path", "index.html"))
	if !strings.Contains(content, "/docs/new-path/") {
		t.Error("alias redirect should contain target URL")
	}
	if !strings.Contains(content, "refresh") {
		t.Error("alias should contain meta refresh")
	}
}

func TestWriter_StaticCopy(t *testing.T) {
	projDir := t.TempDir()
	outDir := filepath.Join(t.TempDir(), "dist")

	// Create static files.
	staticDir := filepath.Join(projDir, "static", "fonts")
	os.MkdirAll(staticDir, 0o755)
	os.WriteFile(filepath.Join(staticDir, "inter.woff2"), []byte("font-data"), 0o644)
	os.WriteFile(filepath.Join(projDir, "static", "favicon.ico"), []byte("icon-data"), 0o644)

	w := &Writer{OutputDir: outDir, ProjectDir: projDir}
	if _, err := w.Write(nil, nil); err != nil {
		t.Fatal(err)
	}

	assertFileContains(t, filepath.Join(outDir, "fonts", "inter.woff2"), "font-data")
	assertFileContains(t, filepath.Join(outDir, "favicon.ico"), "icon-data")
}

func TestWriter_NoStaticDir(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "dist")
	w := &Writer{OutputDir: outDir, ProjectDir: t.TempDir()}

	// Should not error when static/ doesn't exist.
	if _, err := w.Write(nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestWriter_CleanFlag(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "dist")
	os.MkdirAll(outDir, 0o755)
	os.WriteFile(filepath.Join(outDir, "old-file.html"), []byte("stale"), 0o644)

	tracker := NewOutputTracker(8)
	w := &Writer{OutputDir: outDir, ProjectDir: t.TempDir(), Tracker: tracker}
	pages := []RenderedPage{
		{OutPath: "index.html", HTML: []byte("<html>Fresh</html>")},
	}

	if _, err := w.Write(pages, nil); err != nil {
		t.Fatal(err)
	}
	tracker.Prune(outDir)

	// Old file should be gone after prune.
	if _, err := os.Stat(filepath.Join(outDir, "old-file.html")); !os.IsNotExist(err) {
		t.Error("old file should have been removed by clean")
	}
	assertFileContains(t, filepath.Join(outDir, "index.html"), "Fresh")
}

func TestWriterRejectsUnsafeOutputPaths(t *testing.T) {
	parent := t.TempDir()
	outDir := filepath.Join(parent, "dist")
	w := &Writer{OutputDir: outDir, ProjectDir: t.TempDir()}

	if _, err := w.Write([]RenderedPage{{OutPath: "../escape/index.html", HTML: []byte("bad")}}, nil); err == nil {
		t.Fatal("expected unsafe rendered page path to fail")
	}
	if _, err := os.Stat(filepath.Join(parent, "escape", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("unsafe rendered page escaped output dir, stat err = %v", err)
	}

	if _, err := w.Write(nil, map[string]string{"/../alias/": "/target/"}); err == nil {
		t.Fatal("expected unsafe alias path to fail")
	}
	if _, err := os.Stat(filepath.Join(parent, "alias", "index.html")); !os.IsNotExist(err) {
		t.Fatalf("unsafe alias escaped output dir, stat err = %v", err)
	}
}

func assertFileContains(t *testing.T, path, substr string) {
	t.Helper()
	content := readFile(t, path)
	if !strings.Contains(content, substr) {
		t.Errorf("file %s does not contain %q", path, substr)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}
