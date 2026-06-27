package deploy

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitHubPagesDeployer deploys to GitHub Pages by force-pushing to a branch.
type GitHubPagesDeployer struct {
	Branch string
}

func (d *GitHubPagesDeployer) Name() string { return "github-pages" }

func (d *GitHubPagesDeployer) Deploy(distDir string) error {
	// Get the remote origin URL from the current repo.
	remote, err := gitOutput(".", "remote", "get-url", "origin")
	if err != nil {
		return fmt.Errorf("no git remote origin found: %w", err)
	}

	// Create a temp directory and copy dist files into it.
	tmpDir, err := os.MkdirTemp("", "sarde-deploy-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := copyDir(distDir, tmpDir); err != nil {
		return fmt.Errorf("copying dist files: %w", err)
	}

	// Initialize git repo, commit, and push.
	if err := gitRun(tmpDir, "init"); err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	// Fresh CI environments have no committer identity configured and
	// `git commit` would fail with exit status 128 ("Please tell me who
	// you are"). Probe after init so local/global config both resolve.
	email, _ := gitOutput(tmpDir, "config", "user.email")
	hasIdentity := strings.TrimSpace(email) != ""

	for _, args := range deployCommands(remote, d.Branch, hasIdentity) {
		if err := gitRun(tmpDir, args...); err != nil {
			return fmt.Errorf("git %s: %w", gitCmdLabel(args), err)
		}
	}

	return nil
}

// deployCommands returns the git commands to run after `git init`, in order.
// Without a configured identity, the commit carries one-shot -c overrides.
func deployCommands(remote, branch string, hasIdentity bool) [][]string {
	commit := []string{"commit", "-m", "deploy site"}
	if !hasIdentity {
		commit = append([]string{"-c", "user.name=sarde-deploy", "-c", "user.email=sarde-deploy@localhost"}, commit...)
	}
	return [][]string{
		{"add", "."},
		commit,
		{"remote", "add", "origin", remote},
		{"push", "--force", "origin", "HEAD:" + branch},
	}
}

// gitCmdLabel returns the git subcommand name, skipping any leading -c flags.
func gitCmdLabel(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "-c" {
			i++ // skip the -c value
			continue
		}
		return args[i]
	}
	return "git"
}

func gitRun(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// Trim trailing newline.
	s := string(out)
	if len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return s, nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
