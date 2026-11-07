package build

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/config"
)

// TestResolveGitIndex_NoHistory covers a site that is not under version
// control yet: a local build falls back to mtime silently, and a CI build
// warns without the raw exec error in the message.
func TestResolveGitIndex_NoHistory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	cfg := &config.SiteConfig{}
	cfg.Build.LastUpdated = "git"
	b := &SiteBuilder{config: cfg}

	// t.TempDir is outside any repository, unless GIT_DIR or a parent repo
	// says otherwise; skip rather than fail in that unusual setup.
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").CombinedOutput(); err == nil {
		t.Skipf("temp dir is inside a git repository: %s", out)
	}

	t.Run("local", func(t *testing.T) {
		t.Setenv("CI", "")
		if _, warn := b.resolveGitIndex(nil, dir); warn != nil {
			t.Errorf("expected no warning outside CI, got %q", warn.Message)
		}
	})

	t.Run("ci", func(t *testing.T) {
		t.Setenv("CI", "true")
		_, warn := b.resolveGitIndex(nil, dir)
		if warn == nil {
			t.Fatal("expected a warning in CI")
		}
		if !strings.Contains(warn.Message, "not a git repository") {
			t.Errorf("warning should name the cause: %q", warn.Message)
		}
		if strings.Contains(warn.Message, "exit status") {
			t.Errorf("warning should not include the raw exec error: %q", warn.Message)
		}
	})
}

func TestRunningInCI(t *testing.T) {
	for v, want := range map[string]bool{"": false, "0": false, "false": false, "true": true, "1": true} {
		t.Setenv("CI", v)
		if got := runningInCI(); got != want {
			t.Errorf("CI=%q: runningInCI() = %v, want %v", v, got, want)
		}
	}
}
