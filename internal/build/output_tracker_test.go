package build

import (
	"os"
	"path/filepath"
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
