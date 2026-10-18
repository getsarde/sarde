package cli

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/buildlock"
)

// buildableDeploySite is deploySite with content and no dist/, so only
// --build can produce the output.
func buildableDeploySite(t *testing.T, deployYAML string) string {
	t.Helper()
	dir := t.TempDir()
	yaml := "site:\n  title: Deploy Build Test\n  url: \"http://localhost:3000\"\n" + deployYAML
	if err := os.WriteFile(filepath.Join(dir, "sarde.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "content"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "content", "_index.md"), []byte("---\ntitle: Home\n---\n# Welcome\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDeployJSON_BuildFirst(t *testing.T) {
	dir := buildableDeploySite(t, "deploy:\n  provider: custom\n  command: echo shipped\n")
	lines, err := runDeployCLI(t, dir, "--format", "json", "--check=false", "--provider", "", "--build")
	if err != nil {
		t.Fatalf("deploy --build: %v\n%s", err, strings.Join(lines, "\n"))
	}
	if _, err := os.Stat(filepath.Join(dir, "dist", "index.html")); err != nil {
		t.Fatalf("--build did not write dist/index.html: %v", err)
	}
	// Every stdout line is a deploy event: the build reports nothing there.
	events := decodeEvents(t, lines)
	if events[0]["event"] != "start" {
		t.Errorf("first line = %v, want the start event", events[0])
	}
	if last := events[len(events)-1]; last["event"] != "result" || last["ok"] != true {
		t.Errorf("last event = %v", last)
	}
	if n := buildlock.RefCount(filepath.Join(dir, "dist")); n != 0 {
		t.Errorf("RefCount after deploy = %d, want 0", n)
	}
}

func TestDeployJSON_BuildFailureIsBuildFailed(t *testing.T) {
	dir := buildableDeploySite(t, "deploy:\n  provider: custom\n  command: echo shipped\n")
	// An unknown theme token fails the build (the deploy's own config read
	// does not check tokens), so the failure is the build's.
	cfg := filepath.Join(dir, "sarde.yaml")
	data, _ := os.ReadFile(cfg)
	os.WriteFile(cfg, append(data, []byte("theme:\n  overrides:\n    not-a-token: red\n")...), 0o644)

	lines, err := runDeployCLI(t, dir, "--format", "json", "--check=false", "--provider", "", "--build")
	if err == nil {
		t.Fatal("expected the build to fail")
	}
	events := decodeEvents(t, lines)
	env, ok := events[len(events)-1]["error"].(map[string]any)
	if !ok || env["kind"] != "build_failed" {
		t.Errorf("last line = %v, want a build_failed envelope", events[len(events)-1])
	}
}

func TestDeploy_BuildAndCheckRejected(t *testing.T) {
	dir := deploySite(t, "deploy:\n  provider: custom\n  command: echo shipped\n")
	_, err := runDeployCLI(t, dir, "--format", "pretty", "--provider", "", "--build", "--check")
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Errorf("err = %v, want --build and --check rejected", err)
	}
}

// TestHelperHoldDeployLock is not a real test: the contention test below
// re-executes the test binary to hold the output lock in another process
// (the lock is re-entrant inside one).
func TestHelperHoldDeployLock(t *testing.T) {
	if os.Getenv("SARDE_DEPLOY_LOCK_HELPER") != "1" {
		t.Skip("helper process only")
	}
	l, err := buildlock.Acquire(os.Getenv("SARDE_DEPLOY_LOCK_DIR"), "dev")
	if err != nil {
		os.Stdout.WriteString("ACQUIRE_FAILED: " + err.Error() + "\n")
		os.Exit(2)
	}
	os.Stdout.WriteString("HELPER_LOCKED\n")
	io.Copy(io.Discard, os.Stdin)
	l.Release()
	os.Exit(0)
}

func TestDeploy_FailsWhileAnotherProcessHoldsTheOutput(t *testing.T) {
	dir := deploySite(t, "deploy:\n  provider: custom\n  command: echo shipped\n")
	child := exec.Command(os.Args[0], "-test.run=^TestHelperHoldDeployLock$")
	child.Env = append(os.Environ(), "SARDE_DEPLOY_LOCK_HELPER=1", "SARDE_DEPLOY_LOCK_DIR="+filepath.Join(dir, "dist"))
	stdin, err := child.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		stdin.Close()
		child.Wait()
	}()
	ready := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			if line := sc.Text(); strings.HasPrefix(line, "HELPER_LOCKED") || strings.HasPrefix(line, "ACQUIRE_FAILED") {
				ready <- line
				return
			}
		}
		ready <- "EOF"
	}()
	select {
	case line := <-ready:
		if line != "HELPER_LOCKED" {
			t.Fatalf("helper did not take the lock: %s", line)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("helper never took the lock")
	}

	_, err = runDeployCLI(t, dir, "--format", "pretty", "--check=false", "--provider", "")
	if _, locked := buildlock.IsLocked(err); !locked {
		t.Errorf("deploy err = %v, want the output lock error", err)
	}
}
