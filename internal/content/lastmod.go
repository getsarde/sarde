package content

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// GetLastUpdated returns the "last updated" timestamp for a file according
// to the configured strategy. It returns nil when disabled or when no
// timestamp can be determined.
//
// Strategies:
//   - "false" / "off" / "none" — disabled, returns nil
//   - "git"                    — commit time, falls back to mtime when unavailable
//   - "mtime" (default)        — file modification time
//
// The git strategy has three cases depending on idx:
//   - nil index: resolve per file (one subprocess), for callers with no index
//   - index present but unavailable: git was already probed and found unusable,
//     so fall straight back to mtime rather than retrying per file
//   - index available: O(1) map lookup, no subprocess
func GetLastUpdated(filePath, strategy string, idx *GitLastModIndex) *time.Time {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case "false", "off", "none":
		return nil
	case "git":
		switch {
		case idx == nil:
			if t, ok := gitLastCommitTime(filePath); ok {
				return &t
			}
		case idx.Available():
			if t, ok := idx.Lookup(filePath); ok {
				return &t
			}
		}
		// Untracked, unresolved, or git unusable: fall through to mtime.
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return nil
	}
	t := info.ModTime()
	return &t
}

// gitLastCommitTime runs `git log -1 --format=%ct -- <filePath>` in the file's
// own directory and returns the parsed commit time. Setting cmd.Dir matters:
// without it git runs in the process working directory and reports the file as
// outside the repository whenever the build runs from elsewhere.
func gitLastCommitTime(filePath string) (time.Time, bool) {
	cmd := exec.Command("git", "log", "-1", "--format=%ct", "--", filePath)
	cmd.Dir = filepath.Dir(filePath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return time.Time{}, false
	}
	raw := strings.TrimSpace(stdout.String())
	if raw == "" {
		return time.Time{}, false
	}
	secs, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.Unix(secs, 0), true
}
