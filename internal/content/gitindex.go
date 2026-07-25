package content

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
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
	byPath    map[string]int64
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
// false when the path is untracked, was renamed, or no index is available.
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "log",
		"--format=%x01%ct", "--name-only", "--", filepath.ToSlash(relContentDir))
	cmd.Dir = idx.repoRoot

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("git log: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
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

		key := foldGitPath(filepath.Join(idx.repoRoot, filepath.FromSlash(line)))
		if _, want := wanted[key]; !want {
			continue
		}
		if _, seen := idx.byPath[key]; seen {
			continue
		}
		idx.byPath[key] = current
		resolved++
		if resolved == len(wanted) {
			break
		}
	}

	// Cancelling closes the pipe, so Wait reports the kill rather than a real
	// failure. Anything already recorded stays valid.
	cancel()
	_ = cmd.Wait()
	return nil
}

func gitRevParse(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"rev-parse"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.Output()
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
