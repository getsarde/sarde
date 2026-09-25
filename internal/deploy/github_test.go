package deploy

import (
	"os"
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
