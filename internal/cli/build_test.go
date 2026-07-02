package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/internal/buildlock"
	"github.com/getsarde/sarde/internal/consts"
)

func createBuildFixtureSite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "content"), 0o755)
	os.WriteFile(filepath.Join(dir, "sarde.yaml"),
		[]byte("site:\n  title: \"Lock Test\"\n  url: \"http://localhost:3000\"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "content", "_index.md"),
		[]byte("---\ntitle: Home\n---\n# Welcome\n"), 0o644)

	return dir
}

// sarde build must take the single-instance output lock for the duration of
// the build (lock file created in dist/) and fully release it afterwards.
func TestRunBuild_AcquiresAndReleasesOutputLock(t *testing.T) {
	dir := createBuildFixtureSite(t)

	cmd := rootCmd
	cmd.SetArgs([]string{"build", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("build: %v", err)
	}

	outDir := filepath.Join(dir, "dist")
	lockPath := filepath.Join(outDir, consts.FileOutputLock)
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("lock file not created during build: %v", err)
	}
	if n := buildlock.RefCount(outDir); n != 0 {
		t.Errorf("RefCount after build = %d, want 0", n)
	}

	// Released means re-acquirable.
	l, err := buildlock.Acquire(outDir, "test")
	if err != nil {
		t.Fatalf("Acquire after build: %v", err)
	}
	l.Release()
}
