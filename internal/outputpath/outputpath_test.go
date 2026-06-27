package outputpath

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveOutputDir(t *testing.T) {
	projectDir := t.TempDir()
	outside := t.TempDir()

	rejects := []struct {
		name       string
		configured string
	}{
		{"parent traversal", "../dist"},
		{"bare dotdot", ".."},
		// Cleans to a safe path, but the raw value contains a ".." segment and
		// must still be rejected (regression for removing the wrong duplicate
		// traversal check).
		{"traversal that cleans away", "foo/../bar"},
		{"project root", "."},
		{"content source dir", "content"},
		{"git dir", ".git"},
		{"inside content", "content/sub"},
	}
	for _, tt := range rejects {
		if _, err := ResolveOutputDir(projectDir, tt.configured); err == nil {
			t.Errorf("%s: ResolveOutputDir(%q) should be rejected", tt.name, tt.configured)
		}
	}

	accepts := []struct {
		name       string
		configured string
		want       string
	}{
		{"empty defaults to dist", "", filepath.Join(projectDir, "dist")},
		{"relative dist", "dist", filepath.Join(projectDir, "dist")},
		{"nested relative", "build/out", filepath.Join(projectDir, "build", "out")},
		{"absolute outside project", outside, outside},
	}
	for _, tt := range accepts {
		got, err := ResolveOutputDir(projectDir, tt.configured)
		if err != nil {
			t.Errorf("%s: ResolveOutputDir(%q) unexpected error: %v", tt.name, tt.configured, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s: ResolveOutputDir(%q) = %q, want %q", tt.name, tt.configured, got, tt.want)
		}
	}

	if _, err := ResolveOutputDir("", "dist"); err == nil {
		t.Error("empty project dir should be rejected")
	}
}

func TestSafeJoin(t *testing.T) {
	out := t.TempDir()

	if _, err := SafeJoin(out, "../x"); err == nil {
		t.Error("traversal relPath should be rejected")
	}
	if _, err := SafeJoin(out, ""); err == nil {
		t.Error("empty relPath should be rejected")
	}
	if runtime.GOOS == "windows" {
		if _, err := SafeJoin(out, `C:\x`); err == nil {
			t.Error("volume-prefixed relPath should be rejected")
		}
	}

	got, err := SafeJoin(out, "a/b.html")
	if err != nil {
		t.Fatalf("SafeJoin(%q, %q): %v", out, "a/b.html", err)
	}
	if want := filepath.Join(out, "a", "b.html"); got != want {
		t.Errorf("SafeJoin = %q, want %q", got, want)
	}
}

func TestSamePath(t *testing.T) {
	dir := t.TempDir()

	if !samePath(dir, dir) {
		t.Error("identical paths must compare equal")
	}
	if !samePath(dir, dir+string(os.PathSeparator)) {
		t.Error("trailing separator must not affect equality")
	}

	upper := strings.ToUpper(dir)
	if upper == dir {
		t.Skip("temp dir has no letters to vary by case")
	}
	wantEqual := runtime.GOOS == "windows" || runtime.GOOS == "darwin"
	if got := samePath(dir, upper); got != wantEqual {
		t.Errorf("samePath case-variant on %s = %v, want %v", runtime.GOOS, got, wantEqual)
	}
}
