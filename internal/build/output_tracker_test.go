package build

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/getsarde/sarde/internal/consts"
)

// Prune must never delete the single-instance lock file at the output root
// (it is held by the running process and never tracked as written), but a
// same-named orphan in a nested directory is still an ordinary orphan.
func TestPrune_PreservesRootLockFile(t *testing.T) {
	outDir := t.TempDir()

	lockPath := filepath.Join(outDir, consts.FileOutputLock)
	if err := os.WriteFile(lockPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	nestedDir := filepath.Join(outDir, "sub")
	os.MkdirAll(nestedDir, 0o755)
	nestedLock := filepath.Join(nestedDir, consts.FileOutputLock)
	if err := os.WriteFile(nestedLock, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	kept := filepath.Join(outDir, "index.html")
	if err := os.WriteFile(kept, []byte("fresh"), 0o644); err != nil {
		t.Fatal(err)
	}

	tracker := NewOutputTracker(2)
	tracker.Track(kept)
	if err := tracker.Prune(outDir); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	if _, err := os.Stat(lockPath); err != nil {
		t.Errorf("root lock file should survive Prune: %v", err)
	}
	if _, err := os.Stat(nestedLock); !os.IsNotExist(err) {
		t.Error("nested lock-named orphan should still be pruned")
	}
	if _, err := os.Stat(kept); err != nil {
		t.Errorf("tracked file should survive Prune: %v", err)
	}
}

// Regression test: on case-insensitive filesystems (Windows/macOS), a build
// whose tracked paths differ from the on-disk casing (e.g. after renaming
// content/Docs/ to content/docs/, MkdirAll silently reuses the old-cased
// directory) must not prune the file it just wrote.
func TestPrune_CaseInsensitiveFilesystems(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		t.Skip("case-folding applies to Windows/macOS only")
	}

	outDir := t.TempDir()

	// The physical directory keeps the OLD casing.
	pageDir := filepath.Join(outDir, "MyPage")
	os.MkdirAll(pageDir, 0o755)
	page := filepath.Join(pageDir, "index.html")
	if err := os.WriteFile(page, []byte("fresh"), 0o644); err != nil {
		t.Fatal(err)
	}

	// The build tracked the NEW casing (the walk will report the old one).
	tracker := NewOutputTracker(1)
	tracker.Track(filepath.Join(outDir, "mypage", "index.html"))

	if err := tracker.Prune(outDir); err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if _, err := os.Stat(page); err != nil {
		t.Fatalf("file tracked under different casing was pruned: %v", err)
	}
}

// On case-sensitive platforms distinct casings are distinct paths, so a
// different-cased entry really is an orphan and must still be pruned.
func TestPrune_CaseSensitivePlatformsStillPrune(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		t.Skip("only meaningful on case-sensitive filesystems")
	}

	outDir := t.TempDir()
	orphanDir := filepath.Join(outDir, "MyPage")
	os.MkdirAll(orphanDir, 0o755)
	orphan := filepath.Join(orphanDir, "index.html")
	os.WriteFile(orphan, []byte("stale"), 0o644)

	keptDir := filepath.Join(outDir, "mypage")
	os.MkdirAll(keptDir, 0o755)
	kept := filepath.Join(keptDir, "index.html")
	os.WriteFile(kept, []byte("fresh"), 0o644)

	tracker := NewOutputTracker(1)
	tracker.Track(kept)
	if err := tracker.Prune(outDir); err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Error("different-cased orphan should be pruned on case-sensitive FS")
	}
	if _, err := os.Stat(kept); err != nil {
		t.Errorf("tracked file should survive: %v", err)
	}
}
