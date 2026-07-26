package content

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/getsarde/sarde/internal/devlog"
)

// gitLogHardCommitCap bounds the walk when some requested paths are untracked
// and the early exit therefore never fires. Any path left unresolved falls back
// to mtime, exactly as an untracked file does.
const gitLogHardCommitCap = 50000

// gitCommitSentinel prefixes each commit's timestamp line in the log format.
// A file path can never contain this byte, so it separates commit headers from
// path lines without depending on git's blank-line placement, which varies
// between versions.
const gitCommitSentinel = '\x01'

// GitLastModIndex holds the most recent commit time for each content path,
// built from a single `git log` walk instead of one subprocess per file.
//
// A nil index means no index was built and callers should use the per-file
// path. A non-nil index with available == false means git was probed and found
// unusable; callers must fall straight back to mtime rather than retrying per
// file.
type GitLastModIndex struct {
	byPath map[string]int64
	// dead holds wanted paths whose newest git event is a deletion. The file
	// still exists on disk (it was re-created or kept untracked), so the
	// deletion time is not a last-updated time; the tombstone blocks older
	// commits from claiming the path and Lookup misses, giving mtime.
	dead      map[string]struct{}
	repoRoot  string
	headSHA   string
	available bool
	shallow   bool
}

// Available reports whether the index holds usable git data.
func (idx *GitLastModIndex) Available() bool { return idx != nil && idx.available }

// Shallow reports whether the repository is a shallow clone, in which case
// history predating the shallow boundary is missing and affected pages fall
// back to mtime.
func (idx *GitLastModIndex) Shallow() bool { return idx != nil && idx.shallow }

// Lookup returns the most recent commit time for absPath. The second result is
// false when the path is untracked, was deleted from git after its last
// commit (tombstoned), or no index is available. Renamed files resolve
// normally: the walk keys the rename destination, so they carry the
// rename-commit time.
func (idx *GitLastModIndex) Lookup(absPath string) (time.Time, bool) {
	if !idx.Available() {
		return time.Time{}, false
	}
	secs, ok := idx.byPath[foldGitPath(absPath)]
	if !ok {
		return time.Time{}, false
	}
	return time.Unix(secs, 0), true
}

// Stale reports whether HEAD moved since the index was built, which happens
// when a commit lands outside the dev server (for example in another terminal).
// One `git rev-parse`, no history walk.
func (idx *GitLastModIndex) Stale() bool {
	if !idx.Available() {
		return false
	}
	sha, err := gitRevParse(idx.repoRoot, "HEAD")
	return err == nil && sha != idx.headSHA
}

// BuildGitLastModIndex walks git history once for contentDir and records the
// most recent commit time for every path in wantedPaths that git knows about.
//
// It never fails the build: any git problem yields an index with
// available == false plus a descriptive error the caller folds into a single
// build warning.
func BuildGitLastModIndex(contentDir string, wantedPaths []string) (*GitLastModIndex, error) {
	repoRoot, err := gitRevParse(contentDir, "--show-toplevel")
	if err != nil {
		return &GitLastModIndex{}, fmt.Errorf("not a git repository, or git is not installed: %w", err)
	}
	headSHA, err := gitRevParse(repoRoot, "HEAD")
	if err != nil {
		return &GitLastModIndex{}, fmt.Errorf("repository has no commits: %w", err)
	}

	shallow, _ := gitRevParse(repoRoot, "--is-shallow-repository")

	relContentDir, err := filepath.Rel(repoRoot, contentDir)
	if err != nil {
		relContentDir = "."
	}

	idx := &GitLastModIndex{
		byPath:    make(map[string]int64, len(wantedPaths)),
		dead:      make(map[string]struct{}),
		repoRoot:  repoRoot,
		headSHA:   headSHA,
		available: true,
		shallow:   shallow == "true",
	}

	wanted := make(map[string]struct{}, len(wantedPaths))
	for _, p := range wantedPaths {
		wanted[foldGitPath(p)] = struct{}{}
	}
	if len(wanted) == 0 {
		return idx, nil
	}

	if err := idx.walk(relContentDir, wanted); err != nil {
		return &GitLastModIndex{}, err
	}
	return idx, nil
}

// walk streams `git log`, recording the first timestamp seen for each wanted
// path. Git emits newest-first, so the first hit is the most recent commit.
func (idx *GitLastModIndex) walk(relContentDir string, wanted map[string]struct{}) error {
	// Timeout, not just cancel: WaitDelay bounds I/O after the process exits,
	// but only a deadline bounds a git that never exits at all.
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	// core.quotepath=off makes git emit non-ASCII paths as raw UTF-8 bytes.
	// The default quoting octal-escapes them ("docs/caf\303\251.md", in double
	// quotes), which can never match a foldGitPath key.
	//
	// --name-status rather than --name-only, for the status letter: a D line
	// tombstones the path (see the dead map), and rename lines carry the
	// destination as their last field. Shapes seen in the wild: "M\tpath",
	// "A\tpath", "D\tpath", "R100\told\tnew" (rename inside the pathspec),
	// plain "A\tnew" for a rename whose source lies outside the pathspec, and
	// separate A + D lines when diff.renames is disabled.
	cmd := exec.CommandContext(ctx, "git", "-c", "core.quotepath=off", "log",
		"--format=%x01%ct", "--name-status", "--", filepath.ToSlash(relContentDir))
	cmd.Dir = idx.repoRoot

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("git log: %w", err)
	}
	// Leave Stderr nil so os/exec wires it to the null device directly. Giving
	// it an in-process writer instead makes os/exec run a copy goroutine, and
	// Wait blocks on that goroutine until every writer of the pipe is closed.
	// A handle inherited by an unrelated process therefore hung the whole build
	// after git had already exited. WaitDelay bounds the same risk for the
	// stdout pipe, which the early exits below deliberately stop reading.
	cmd.WaitDelay = 5 * time.Second
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("git log: %w", err)
	}

	var current int64
	commits := 0
	resolved := 0

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if line[0] == gitCommitSentinel {
			commits++
			if commits > gitLogHardCommitCap {
				break
			}
			if secs, err := strconv.ParseInt(line[1:], 10, 64); err == nil {
				current = secs
			}
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		status := fields[0]
		// The last field is the path, or the destination for renames/copies.
		key := foldGitPath(filepath.Join(idx.repoRoot, filepath.FromSlash(fields[len(fields)-1])))
		if _, want := wanted[key]; !want {
			continue
		}
		if _, seen := idx.byPath[key]; seen {
			continue
		}
		if _, seen := idx.dead[key]; seen {
			continue
		}
		if strings.HasPrefix(status, "D") {
			idx.dead[key] = struct{}{}
		} else {
			idx.byPath[key] = current
		}
		resolved++
		if resolved == len(wanted) {
			break
		}
	}

	// A scan error (a line past the buffer cap, a read failure) ends the loop
	// exactly like history running out; unresolved paths degrade to mtime, but
	// the truncation should at least be visible.
	if scanErr := scanner.Err(); scanErr != nil {
		devlog.Warn("content", "git log scan: %v", scanErr)
	}

	// Cancelling closes the pipe, so Wait reports the kill rather than a real
	// failure. Anything already recorded stays valid.
	cancel()
	_ = cmd.Wait()
	return nil
}

// gitCommandTimeout bounds every git subprocess the build starts. Commit dates
// are an optimization with an mtime fallback, so a git that never returns (an
// index lock, a credential prompt, an unresponsive network mount) must degrade
// to that fallback rather than stall the build.
const gitCommandTimeout = 30 * time.Second

// gitWaitDelay bounds how long Wait blocks on I/O once the process has exited.
// os/exec runs a copy goroutine for any in-process stdout/stderr writer, and
// Wait blocks on that goroutine until every writer of the pipe is closed. A
// handle inherited by an unrelated process can keep it open indefinitely, which
// is what deadlocked whole builds before these bounds existed.
const gitWaitDelay = 5 * time.Second

// runGitCapture runs git in dir and returns its stdout. Stderr is deliberately
// left nil so os/exec wires it to the null device rather than starting a second
// copy goroutine; no caller reads it. Prefer this over exec.Command().Output(),
// which captures both streams into buffers and waits without any bound.
func runGitCapture(dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.WaitDelay = gitWaitDelay
	if err := cmd.Run(); err != nil {
		// ErrWaitDelay means the process exited with success and only the
		// stdout copy was cut off by WaitDelay, which happens when an
		// inherited handle keeps the pipe open after git is gone. Git already
		// wrote everything before exiting, so the captured output is complete
		// and discarding it would silently disable the git strategy in the
		// very scenario these bounds exist for.
		if errors.Is(err, exec.ErrWaitDelay) {
			return stdout.Bytes(), nil
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}

func gitRevParse(dir string, args ...string) (string, error) {
	out, err := runGitCapture(dir, append([]string{"rev-parse"}, args...)...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// foldGitPath canonicalizes a path for map keys, matching foldPath in
// internal/build/output_tracker.go.
func foldGitPath(path string) string {
	p := filepath.Clean(path)
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.ToLower(p)
	}
	return p
}
