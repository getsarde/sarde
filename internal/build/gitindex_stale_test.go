package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

// initGitTestRepo creates a git repository with local identity config so
// commits work regardless of the machine's global git setup.
func initGitTestRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	gitRunT(t, dir, "init", "-q")
	gitRunT(t, dir, "config", "user.email", "test@example.com")
	gitRunT(t, dir, "config", "user.name", "Test")
	gitRunT(t, dir, "config", "commit.gpgsign", "false")
	return dir
}

func gitRunT(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// commitGitAt commits everything staged with a fixed author and committer date.
func commitGitAt(t *testing.T, dir, message string, when time.Time) {
	t.Helper()
	stamp := when.Format(time.RFC3339)
	cmd := exec.Command("git", "commit", "-q", "-m", message, "--date", stamp)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_COMMITTER_DATE="+stamp)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

// TestContentRebuild_RefreshesStaleGitIndex guards the dev-server git-date
// freeze: a commit landing from outside the dev server (another terminal)
// must be picked up by the next ContentRebuild, not served from the git index
// snapshot captured at the last full build's HEAD.
func TestContentRebuild_RefreshesStaleGitIndex(t *testing.T) {
	repo := initGitTestRepo(t)
	t1 := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)

	writeFixture(t, repo, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, repo, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, repo, "content/docs/guide.md", "---\ntitle: Guide\nweight: 1\n---\n# Guide\nOriginal body.\n")
	gitRunT(t, repo, "add", "-A")
	commitGitAt(t, repo, "initial", t1)

	cfg := config.Defaults() // build.last_updated: git (embedded default)
	cfg.Build.Minify = config.BoolPtr(false)
	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  repo,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	if _, err := builder.Build(); err != nil {
		t.Fatalf("full build failed: %v", err)
	}
	if !builder.lastGitIndex.Available() {
		t.Fatal("expected an available git index after the full build")
	}

	guidePath := filepath.Join(repo, "content", "docs", "guide.md")
	if got, ok := builder.lastGitIndex.Lookup(guidePath); !ok || !got.Equal(t1) {
		t.Fatalf("index after full build: got %v (ok=%v), want %v", got, ok, t1)
	}

	// Simulate an edit plus commit from outside the dev server.
	writeFixture(t, repo, "content/docs/guide.md", "---\ntitle: Guide\nweight: 1\n---\n# Guide\nUpdated body.\n")
	gitRunT(t, repo, "add", "content/docs/guide.md")
	commitGitAt(t, repo, "external edit", t2)

	if !builder.lastGitIndex.Stale() {
		t.Fatal("index should report stale once HEAD moves past the full build's snapshot")
	}

	result, err := builder.ContentRebuild([]string{guidePath})
	if err != nil {
		t.Fatalf("ContentRebuild failed: %v", err)
	}
	if result.PageCount < 1 {
		t.Errorf("PageCount = %d, want at least the rebuilt page", result.PageCount)
	}

	if builder.lastGitIndex.Stale() {
		t.Error("index should be fresh immediately after ContentRebuild")
	}
	if got, ok := builder.lastGitIndex.Lookup(guidePath); !ok || !got.Equal(t2) {
		t.Errorf("index after ContentRebuild: got %v (ok=%v), want %v", got, ok, t2)
	}

	var page *engine.Page
	for _, p := range builder.lastAllPages {
		if p.FilePath == guidePath {
			page = p
			break
		}
	}
	if page == nil {
		t.Fatal("rebuilt guide page missing from lastAllPages")
	}
	if !page.Updated.Equal(t2) {
		t.Errorf("page.Updated = %v, want %v (new commit time)", page.Updated, t2)
	}

	data, err := os.ReadFile(filepath.Join(repo, "dist", "docs", "guide", "index.html"))
	if err != nil {
		t.Fatalf("reading rebuilt output: %v", err)
	}
	wantDatetime := `<time datetime="` + t2.Format("2006-01-02") + `"`
	if !strings.Contains(string(data), wantDatetime) {
		t.Errorf("rendered output missing %q", wantDatetime)
	}
}
