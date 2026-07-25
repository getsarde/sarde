package content

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFileTest(t *testing.T, dir, name, contents string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestGetLastUpdated_Disabled(t *testing.T) {
	p := writeFileTest(t, t.TempDir(), "x.md", "hello")
	for _, strategy := range []string{"false", "off", "none", "FALSE", " off "} {
		if got := GetLastUpdated(p, strategy, nil); got != nil {
			t.Errorf("GetLastUpdated(%q) = %v, want nil", strategy, got)
		}
	}
}

func TestGetLastUpdated_Mtime(t *testing.T) {
	p := writeFileTest(t, t.TempDir(), "x.md", "hello")
	expected := time.Now()
	// Back-date the mtime so we can compare.
	stamp := expected.Add(-time.Hour)
	if err := os.Chtimes(p, stamp, stamp); err != nil {
		t.Fatal(err)
	}

	got := GetLastUpdated(p, "mtime", nil)
	if got == nil {
		t.Fatal("GetLastUpdated(mtime) returned nil")
	}
	if !got.Equal(stamp) {
		t.Errorf("GetLastUpdated(mtime) = %v, want %v", got, stamp)
	}
}

func TestGetLastUpdated_DefaultFallsBackToMtime(t *testing.T) {
	// Empty/unknown strategy falls through to mtime.
	p := writeFileTest(t, t.TempDir(), "x.md", "hello")
	got := GetLastUpdated(p, "", nil)
	if got == nil {
		t.Error("GetLastUpdated(\"\") returned nil, want mtime")
	}
}

func TestGetLastUpdated_GitFallsBackOnMissingGit(t *testing.T) {
	// Running "git" in a tempdir outside any repo returns an error; we fall
	// through to mtime. This validates the fallback path without requiring
	// an actual git-initialised repository.
	p := writeFileTest(t, t.TempDir(), "x.md", "hello")
	got := GetLastUpdated(p, "git", nil)
	if got == nil {
		t.Error("GetLastUpdated(git) returned nil, expected mtime fallback")
	}
}
