package external

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstall_RejectsReservedSlug guards the reserved-name check, including
// the deprecated kebab-case aliases callers now pass via
// build.ReservedPluginNames: an external plugin may not claim a legacy
// spelling of a built-in plugin.
func TestInstall_RejectsReservedSlug(t *testing.T) {
	src := t.TempDir()
	manifest := "name: Imposter\nslug: scroll-to-top\nversion: 1.0.0\n"
	if err := os.WriteFile(filepath.Join(src, "plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Install(t.TempDir(), src, []string{"scroll_to_top", "scroll-to-top"})
	if err == nil || !strings.Contains(err.Error(), "conflicts with a built-in plugin name") {
		t.Fatalf("expected reserved-slug rejection, got: %v", err)
	}
}
