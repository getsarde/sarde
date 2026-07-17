package external

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscoverSlugs(t *testing.T) {
	project := t.TempDir()
	pluginsDir := filepath.Join(project, "plugins")

	// Two valid plugins.
	for _, slug := range []string{"beta", "alpha"} {
		dir := filepath.Join(pluginsDir, slug)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte("name: X\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Directory without a manifest: skipped.
	if err := os.MkdirAll(filepath.Join(pluginsDir, "nomanifest"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Dot-directory: skipped.
	if err := os.MkdirAll(filepath.Join(pluginsDir, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Plain file: skipped.
	if err := os.WriteFile(filepath.Join(pluginsDir, "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := DiscoverSlugs(project)
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DiscoverSlugs() = %v, want %v", got, want)
	}
}

func TestDiscoverSlugsNoPluginsDir(t *testing.T) {
	if got := DiscoverSlugs(t.TempDir()); got != nil {
		t.Errorf("expected nil for missing plugins dir, got %v", got)
	}
}
