package content

import (
	"bytes"
	"os"
	"os/exec"
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
//   - "git"                    — `git log -1 --format=%ct` for the file, falls back to mtime on error
//   - "mtime" (default)        — file modification time
func GetLastUpdated(filePath, strategy string) *time.Time {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case "false", "off", "none":
		return nil
	case "git":
		if t, ok := gitLastCommitTime(filePath); ok {
			return &t
		}
		// Fall through to mtime
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return nil
	}
	t := info.ModTime()
	return &t
}

// gitLastCommitTime runs `git log -1 --format=%ct -- <filePath>` in the file's
// directory and returns the parsed commit time.
func gitLastCommitTime(filePath string) (time.Time, bool) {
	cmd := exec.Command("git", "log", "-1", "--format=%ct", "--", filePath)
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
