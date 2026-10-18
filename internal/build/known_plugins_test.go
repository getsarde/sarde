package build

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// The list feeds the "must be one of" message for plugins.enabled, so it
// must be the same on every run and must not repeat a name, even when a
// project plugin folder shares a built-in's name.
func TestKnownPluginNames_SortedAndUnique(t *testing.T) {
	dir := t.TempDir()
	ext := filepath.Join(dir, "plugins", "search")
	if err := os.MkdirAll(ext, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "name: Search clone\nslug: search\nversion: 1.0.0\n"
	if err := os.WriteFile(filepath.Join(ext, "plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	names := KnownPluginNames(dir)
	if !slices.IsSorted(names) {
		t.Errorf("names are not sorted: %v", names)
	}
	if len(slices.Compact(slices.Clone(names))) != len(names) {
		t.Errorf("names repeat a value: %v", names)
	}
	if !slices.Contains(names, "search") || !slices.Contains(names, "sitemap") {
		t.Errorf("names miss a built-in: %v", names)
	}
}
