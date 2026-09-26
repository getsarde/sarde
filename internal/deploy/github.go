package deploy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// GitHubPagesDeployer deploys to GitHub Pages by force-pushing the output to
// a branch of the project's origin remote, using the git credentials already
// configured on the machine.
type GitHubPagesDeployer struct {
	Branch string
	// CNAME is the custom domain written to the published CNAME file. Empty
	// keeps whatever CNAME the output already contains (for example one
	// copied from static/).
	CNAME string
	opts  Options
}

func (d *GitHubPagesDeployer) Name() string { return "github-pages" }

func (d *GitHubPagesDeployer) projectDir() string {
	if d.opts.ProjectDir != "" {
		return plainDir(d.opts.ProjectDir)
	}
	return "."
}

// Check verifies the origin remote is reachable with the current git
// credentials (`git ls-remote`, read-only).
func (d *GitHubPagesDeployer) Check(ctx context.Context, rep Reporter) (*CheckResult, error) {
	rep = orNop(rep)
	remote, err := gitOutput(ctx, d.projectDir(), "remote", "get-url", "origin")
	if err != nil {
		return nil, configErrorf("github-pages: no git remote origin found in %s: %v", d.projectDir(), err)
	}
	rep.Step(StepPrepare, "Contacting "+remote)
	if _, err := gitOutput(ctx, d.projectDir(), "ls-remote", "--heads", "origin"); err != nil {
		return nil, withCode(CodeAuth, fmt.Errorf("github-pages: cannot read %s: %w", remote, err))
	}
	return &CheckResult{Provider: d.Name(), Target: remote, TargetID: d.Branch}, nil
}

func (d *GitHubPagesDeployer) Deploy(ctx context.Context, distDir string, rep Reporter) (*Result, error) {
	rep = orNop(rep)
	remote, err := gitOutput(ctx, d.projectDir(), "remote", "get-url", "origin")
	if err != nil {
		return nil, configErrorf("github-pages: no git remote origin found in %s: %v", d.projectDir(), err)
	}

	rep.Step(StepCollect, "Collecting files")
	files, err := collectFiles(distDir, rep)
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "sarde-deploy-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := copyFiles(files, tmpDir); err != nil {
		return nil, fmt.Errorf("copying output files: %w", err)
	}
	if err := writeNoJekyll(tmpDir); err != nil {
		return nil, fmt.Errorf("writing .nojekyll: %w", err)
	}
	if err := writeCNAME(tmpDir, d.CNAME, rep); err != nil {
		return nil, fmt.Errorf("writing CNAME: %w", err)
	}

	rep.Step(StepUpload, fmt.Sprintf("Pushing %d files to %s (%s)", len(files), remote, d.Branch))
	if err := gitRun(ctx, tmpDir, rep, "init"); err != nil {
		return nil, fmt.Errorf("git init: %w", err)
	}

	// Fresh CI environments have no committer identity configured and
	// `git commit` would fail with exit status 128 ("Please tell me who
	// you are"). Probe after init so local/global config both resolve.
	email, _ := gitOutput(ctx, tmpDir, "config", "user.email")
	hasIdentity := strings.TrimSpace(email) != ""

	for _, args := range deployCommands(remote, d.Branch, hasIdentity) {
		if err := gitRun(ctx, tmpDir, rep, args...); err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("git %s: %w", gitCmdLabel(args), err)
		}
	}

	return &Result{
		Provider:      d.Name(),
		DeployID:      d.Branch,
		FilesTotal:    len(files),
		FilesUploaded: len(files),
		BytesUploaded: totalSize(files),
	}, nil
}

// writeNoJekyll creates an empty .nojekyll file in dir unless one exists.
// Branch-based GitHub Pages runs Jekyll over the pushed files, and Jekyll
// drops every file and directory whose name starts with an underscore.
func writeNoJekyll(dir string) error {
	path := filepath.Join(dir, ".nojekyll")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, nil, 0o644)
}

// writeCNAME writes the custom domain into dir/CNAME. GitHub Pages reads the
// domain from that file on every push, so without it each force-push would
// remove a custom domain set in the repository settings. An empty domain
// leaves an existing CNAME untouched.
func writeCNAME(dir, domain string, rep Reporter) error {
	if domain == "" {
		return nil
	}
	path := filepath.Join(dir, "CNAME")
	if existing, err := os.ReadFile(path); err == nil {
		if prev := strings.TrimSpace(string(existing)); prev != "" && !strings.EqualFold(prev, domain) {
			orNop(rep).Log(LevelWarn, fmt.Sprintf("replacing CNAME %q from the output with deploy.cname %q", prev, domain))
		}
	}
	return os.WriteFile(path, []byte(domain+"\n"), 0o644)
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

// gitRun runs git in dir and forwards its output to rep line by line.
func gitRun(ctx context.Context, dir string, rep Reporter, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	return runForwarding(cmd, rep)
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

// runForwarding runs cmd with stdout and stderr sent to rep.Log, one call
// per line, so a child process never writes to our stdout directly (which
// would corrupt the CLI's JSON event stream).
func runForwarding(cmd *exec.Cmd, rep Reporter) error {
	rep = orNop(rep)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	var wg sync.WaitGroup
	forward := func(r io.Reader) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			if line := strings.TrimRight(sc.Text(), "\r"); line != "" {
				rep.Log(LevelInfo, line)
			}
		}
	}
	wg.Add(2)
	go forward(stdout)
	go forward(stderr)
	wg.Wait()
	return cmd.Wait()
}

// copyFiles copies the collected files into dst, recreating directories.
func copyFiles(files []File, dst string) error {
	for _, f := range files {
		target := filepath.Join(dst, filepath.FromSlash(f.Rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := copyFile(f.Abs, target); err != nil {
			return err
		}
	}
	return nil
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
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
