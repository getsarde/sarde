package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestWriteEmbeddedAssets_CopiesAssetFiles(t *testing.T) {
	fs := fstest.MapFS{
		"assets/js/foo.js":      {Data: []byte("console.log('foo');")},
		"assets/css/theme.css":  {Data: []byte("body{}")},
		"theme/_default/ignore": {Data: []byte("ignored")},
	}

	out := t.TempDir()
	if err := WriteEmbeddedAssets(fs, out, nil, nil); err != nil {
		t.Fatalf("WriteEmbeddedAssets: %v", err)
	}

	if data, err := os.ReadFile(filepath.Join(out, "assets", "js", "foo.js")); err != nil || string(data) != "console.log('foo');" {
		t.Errorf("assets/js/foo.js missing or wrong contents (err=%v, data=%q)", err, data)
	}
	if _, err := os.Stat(filepath.Join(out, "assets", "css", "theme.css")); err != nil {
		t.Errorf("assets/css/theme.css not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "theme", "_default", "ignore")); !os.IsNotExist(err) {
		t.Errorf("non-assets path was copied: err=%v", err)
	}
}

func TestWriteEmbeddedAssets_SkipPrefixes(t *testing.T) {
	fs := fstest.MapFS{
		"assets/js/foo.js":     {Data: []byte("console.log('foo');")},
		"assets/css/theme.css": {Data: []byte("body{}")},
	}

	out := t.TempDir()
	if err := WriteEmbeddedAssets(fs, out, nil, []string{"assets/js/"}); err != nil {
		t.Fatalf("WriteEmbeddedAssets: %v", err)
	}

	if _, err := os.Stat(filepath.Join(out, "assets", "js", "foo.js")); !os.IsNotExist(err) {
		t.Errorf("assets/js/foo.js should be skipped but was written")
	}
	if _, err := os.Stat(filepath.Join(out, "assets", "css", "theme.css")); err != nil {
		t.Errorf("assets/css/theme.css not written: %v", err)
	}
}

func TestWriteEmbeddedAssets_NoAssetsDir(t *testing.T) {
	fs := fstest.MapFS{"other/file.txt": {Data: []byte("x")}}
	out := t.TempDir()
	if err := WriteEmbeddedAssets(fs, out, nil, nil); err != nil {
		t.Errorf("WriteEmbeddedAssets on fs without assets/: %v", err)
	}
}

func TestWriteEmbeddedAssets_NilFS(t *testing.T) {
	out := t.TempDir()
	if err := WriteEmbeddedAssets(nil, out, nil, nil); err != nil {
		t.Errorf("WriteEmbeddedAssets(nil, ...): %v", err)
	}
}

func TestBundleEmbeddedJS(t *testing.T) {
	fs := fstest.MapFS{
		"assets/js/a.js": {Data: []byte("var a = 1;")},
		"assets/js/b.js": {Data: []byte("var b = 2;")},
	}

	content, filename, err := BundleEmbeddedJS(fs, true)
	if err != nil {
		t.Fatalf("BundleEmbeddedJS: %v", err)
	}
	if filename != "sarde.js" {
		t.Errorf("expected sarde.js in dev mode, got %q", filename)
	}
	s := string(content)
	if !strings.Contains(s, "var a = 1;") || !strings.Contains(s, "var b = 2;") {
		t.Errorf("bundle missing expected content: %q", s)
	}
	if !strings.Contains(s, "(function()") {
		t.Errorf("bundle should wrap in IIFE: %q", s)
	}
}

func TestBundleEmbeddedJS_Production(t *testing.T) {
	fs := fstest.MapFS{
		"assets/js/a.js": {Data: []byte("var longVariableName = 1;")},
	}

	_, filename, err := BundleEmbeddedJS(fs, false)
	if err != nil {
		t.Fatalf("BundleEmbeddedJS: %v", err)
	}
	if !strings.Contains(filename, "sarde.") || !strings.HasSuffix(filename, ".js") {
		t.Errorf("expected fingerprinted filename, got %q", filename)
	}
}

func TestBundleEmbeddedJS_NilFS(t *testing.T) {
	content, filename, err := BundleEmbeddedJS(nil, true)
	if err != nil || content != nil || filename != "" {
		t.Errorf("expected nil return for nil FS, got (%v, %q, %v)", content, filename, err)
	}
}
