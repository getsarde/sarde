package deploy

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeployCommands_WithIdentity(t *testing.T) {
	cmds := deployCommands("https://example.com/repo.git", "gh-pages", true)

	want := [][]string{
		{"add", "."},
		{"commit", "-m", "deploy site"},
		{"remote", "add", "origin", "https://example.com/repo.git"},
		{"push", "--force", "origin", "HEAD:gh-pages"},
	}
	if len(cmds) != len(want) {
		t.Fatalf("got %d commands, want %d", len(cmds), len(want))
	}
	for i := range want {
		if strings.Join(cmds[i], " ") != strings.Join(want[i], " ") {
			t.Errorf("cmd %d = %v, want %v", i, cmds[i], want[i])
		}
	}
}

func TestDeployCommands_WithoutIdentity(t *testing.T) {
	cmds := deployCommands("https://example.com/repo.git", "gh-pages", false)

	commit := strings.Join(cmds[1], " ")
	if !strings.Contains(commit, "-c user.name=sarde-deploy") ||
		!strings.Contains(commit, "-c user.email=sarde-deploy@localhost") {
		t.Errorf("commit command missing -c identity overrides: %v", cmds[1])
	}
	if !strings.HasSuffix(commit, "commit -m deploy site") {
		t.Errorf("commit subcommand malformed: %v", cmds[1])
	}
}

func TestWriteNoJekyll(t *testing.T) {
	dir := t.TempDir()
	if err := writeNoJekyll(dir); err != nil {
		t.Fatalf("writeNoJekyll: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, ".nojekyll"))
	if err != nil {
		t.Fatalf(".nojekyll not created: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf(".nojekyll size = %d, want 0", info.Size())
	}
}

func TestWriteNoJekyll_KeepsExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nojekyll")
	if err := os.WriteFile(path, []byte("user"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeNoJekyll(dir); err != nil {
		t.Fatalf("writeNoJekyll: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "user" {
		t.Errorf(".nojekyll content = %q, want existing file left untouched", got)
	}
}

func TestGitCmdLabel(t *testing.T) {
	if got := gitCmdLabel([]string{"commit", "-m", "x"}); got != "commit" {
		t.Errorf("got %q, want commit", got)
	}
	if got := gitCmdLabel([]string{"-c", "user.name=x", "-c", "user.email=y", "commit", "-m", "z"}); got != "commit" {
		t.Errorf("got %q, want commit (skipping -c pairs)", got)
	}
}

func TestWriteCNAME(t *testing.T) {
	dir := t.TempDir()
	if err := writeCNAME(dir, "docs.example.com", nil); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "CNAME"))
	if string(got) != "docs.example.com\n" {
		t.Errorf("CNAME = %q", got)
	}
}

func TestWriteCNAME_EmptyKeepsExisting(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CNAME"), []byte("from-static.example.com\n"), 0o644)
	if err := writeCNAME(dir, "", nil); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "CNAME"))
	if string(got) != "from-static.example.com\n" {
		t.Errorf("CNAME = %q, want the existing file untouched", got)
	}
}

func TestWriteCNAME_OverrideWarns(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "CNAME"), []byte("old.example.com\n"), 0o644)
	rep := &recordingReporter{}
	if err := writeCNAME(dir, "new.example.com", rep); err != nil {
		t.Fatal(err)
	}
	if len(rep.logs) != 1 || !strings.Contains(rep.logs[0], "old.example.com") {
		t.Errorf("logs = %v, want one warning naming the replaced domain", rep.logs)
	}
}

func TestCopyFiles_ExcludesLock(t *testing.T) {
	dist := writeDist(t, map[string]string{"index.html": "home", "sub/page.html": "page"})
	files, err := collectFiles(dist, nil)
	if err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	if err := copyFiles(files, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, ".sarde.lock")); !os.IsNotExist(err) {
		t.Error(".sarde.lock was copied into the published tree")
	}
	if got, _ := os.ReadFile(filepath.Join(dst, "sub", "page.html")); string(got) != "page" {
		t.Errorf("sub/page.html = %q", got)
	}
}

// End to end against a local bare repository: the pushed branch carries
// the site, .nojekyll and CNAME, and never the lock file.
func TestGitHubPagesDeploy_LocalBareRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	ctx := context.Background()
	bare := t.TempDir()
	mustGit(t, bare, "init", "--bare", "-q")
	project := t.TempDir()
	mustGit(t, project, "init", "-q")
	mustGit(t, project, "remote", "add", "origin", bare)

	dist := writeDist(t, map[string]string{"index.html": "home", "_astro/x.css": "css"})
	d := &GitHubPagesDeployer{Branch: "gh-pages", CNAME: "docs.example.com", opts: Options{ProjectDir: project}}
	res, err := d.Deploy(ctx, dist, &recordingReporter{})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if res.FilesTotal != 2 {
		t.Errorf("FilesTotal = %d, want 2 (lock excluded)", res.FilesTotal)
	}
	out, err := exec.Command("git", "-C", bare, "ls-tree", "-r", "--name-only", "gh-pages").Output()
	if err != nil {
		t.Fatalf("ls-tree: %v", err)
	}
	tree := strings.Fields(string(out))
	want := []string{".nojekyll", "CNAME", "_astro/x.css", "index.html"}
	if strings.Join(tree, ",") != strings.Join(want, ",") {
		t.Errorf("pushed tree = %v, want %v", tree, want)
	}
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
