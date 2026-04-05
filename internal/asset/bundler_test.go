package asset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupBundlerTest(t *testing.T) (projectDir string, bundler *Bundler) {
	t.Helper()
	projectDir = t.TempDir()

	assetsDir := filepath.Join(projectDir, "assets", "css")
	os.MkdirAll(assetsDir, 0o755)

	jsDir := filepath.Join(projectDir, "assets", "js")
	os.MkdirAll(jsDir, 0o755)

	resolver := &Resolver{ProjectDir: projectDir}
	bundler = &Bundler{
		Resolver:  resolver,
		DevMode:   false,
		OutputDir: t.TempDir(),
	}
	return
}

func TestBundler_BundleCSS(t *testing.T) {
	projectDir, bundler := setupBundlerTest(t)

	// Create a CSS file.
	cssPath := filepath.Join(projectDir, "assets", "css", "main.css")
	os.WriteFile(cssPath, []byte(`
body { color: red; }
h1 { font-size: 2rem; }
`), 0o644)

	result, err := bundler.BundleCSS("css/main.css")
	if err != nil {
		t.Fatalf("BundleCSS failed: %v", err)
	}

	if len(result.OutputFiles) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(result.OutputFiles))
	}

	f := result.OutputFiles[0]
	if !strings.HasPrefix(f.OutputURL, "/assets/css/") {
		t.Errorf("OutputURL = %q, want /assets/css/ prefix", f.OutputURL)
	}
	if f.OriginalPath != "css/main.css" {
		t.Errorf("OriginalPath = %q, want css/main.css", f.OriginalPath)
	}

	// Content should be minified (no whitespace).
	content := string(f.Content)
	if strings.Contains(content, "  ") {
		t.Error("expected minified CSS (production mode)")
	}
}

func TestBundler_BundleCSS_DevMode(t *testing.T) {
	projectDir, bundler := setupBundlerTest(t)
	bundler.DevMode = true

	cssPath := filepath.Join(projectDir, "assets", "css", "style.css")
	os.WriteFile(cssPath, []byte("body { color: blue; }"), 0o644)

	result, err := bundler.BundleCSS("css/style.css")
	if err != nil {
		t.Fatalf("BundleCSS failed: %v", err)
	}

	if len(result.OutputFiles) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(result.OutputFiles))
	}

	f := result.OutputFiles[0]
	// In dev mode, filename should NOT be fingerprinted.
	if strings.Count(f.Name, ".") > 1 {
		t.Errorf("dev mode should not fingerprint: %q", f.Name)
	}

	// Content should be in memory.
	if len(f.Content) == 0 {
		t.Error("expected non-empty content")
	}
}

func TestBundler_BundleJS(t *testing.T) {
	projectDir, bundler := setupBundlerTest(t)

	jsPath := filepath.Join(projectDir, "assets", "js", "app.js")
	os.WriteFile(jsPath, []byte(`
const greeting = "hello";
console.log(greeting);
`), 0o644)

	result, err := bundler.BundleJS("js/app.js")
	if err != nil {
		t.Fatalf("BundleJS failed: %v", err)
	}

	if len(result.OutputFiles) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(result.OutputFiles))
	}

	f := result.OutputFiles[0]
	if !strings.HasPrefix(f.OutputURL, "/assets/js/") {
		t.Errorf("OutputURL = %q, want /assets/js/ prefix", f.OutputURL)
	}

	// Content should be in memory.
	if len(f.Content) == 0 {
		t.Error("expected non-empty content")
	}
}

func TestBundler_BundleCSS_WithImport(t *testing.T) {
	projectDir, bundler := setupBundlerTest(t)

	// Create two CSS files with @import.
	baseCSS := filepath.Join(projectDir, "assets", "css", "base.css")
	os.WriteFile(baseCSS, []byte("html { box-sizing: border-box; }"), 0o644)

	mainCSS := filepath.Join(projectDir, "assets", "css", "main.css")
	os.WriteFile(mainCSS, []byte(`@import "./base.css";
body { margin: 0; }
`), 0o644)

	result, err := bundler.BundleCSS("css/main.css")
	if err != nil {
		t.Fatalf("BundleCSS failed: %v", err)
	}

	if len(result.OutputFiles) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(result.OutputFiles))
	}

	// Bundled output should contain content from both files.
	content := string(result.OutputFiles[0].Content)
	if !strings.Contains(content, "box-sizing") {
		t.Error("bundled CSS should contain imported base.css content")
	}
	if !strings.Contains(content, "margin") {
		t.Error("bundled CSS should contain main.css content")
	}
}

func TestBundler_NotFound(t *testing.T) {
	_, bundler := setupBundlerTest(t)

	_, err := bundler.BundleCSS("css/nonexistent.css")
	if err == nil {
		t.Error("expected error for missing entry point")
	}
}

func TestBundler_Fingerprinting(t *testing.T) {
	projectDir, bundler := setupBundlerTest(t)

	cssPath := filepath.Join(projectDir, "assets", "css", "style.css")
	os.WriteFile(cssPath, []byte("body { color: green; }"), 0o644)

	result, err := bundler.BundleCSS("css/style.css")
	if err != nil {
		t.Fatalf("BundleCSS failed: %v", err)
	}

	f := result.OutputFiles[0]
	// In production mode, filename should contain a hash.
	parts := strings.Split(f.Name, ".")
	if len(parts) != 3 { // "style.HASH.css"
		t.Errorf("expected fingerprinted name with 3 parts, got %q (%d parts)", f.Name, len(parts))
	}
}
