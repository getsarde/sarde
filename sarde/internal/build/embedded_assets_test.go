package build

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestWriteEmbeddedAssets_CopiesAssetFiles(t *testing.T) {
	fs := fstest.MapFS{
		"assets/js/foo.js":       {Data: []byte("console.log('foo');")},
		"assets/css/theme.css":   {Data: []byte("body{}")},
		"theme/_default/ignore": {Data: []byte("ignored")},
	}

	out := t.TempDir()
	if err := WriteEmbeddedAssets(fs, out, nil); err != nil {
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

func TestWriteEmbeddedAssets_NoAssetsDir(t *testing.T) {
	// FS without a top-level "assets" directory should be a no-op.
	fs := fstest.MapFS{"other/file.txt": {Data: []byte("x")}}
	out := t.TempDir()
	if err := WriteEmbeddedAssets(fs, out, nil); err != nil {
		t.Errorf("WriteEmbeddedAssets on fs without assets/: %v", err)
	}
}

func TestWriteEmbeddedAssets_NilFS(t *testing.T) {
	out := t.TempDir()
	if err := WriteEmbeddedAssets(nil, out, nil); err != nil {
		t.Errorf("WriteEmbeddedAssets(nil, ...): %v", err)
	}
}
