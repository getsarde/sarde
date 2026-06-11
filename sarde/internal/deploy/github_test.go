package deploy

import (
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

func TestGitCmdLabel(t *testing.T) {
	if got := gitCmdLabel([]string{"commit", "-m", "x"}); got != "commit" {
		t.Errorf("got %q, want commit", got)
	}
	if got := gitCmdLabel([]string{"-c", "user.name=x", "-c", "user.email=y", "commit", "-m", "z"}); got != "commit" {
		t.Errorf("got %q, want commit (skipping -c pairs)", got)
	}
}
