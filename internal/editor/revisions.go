package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const revisionsDirName = ".revisions"

// RevisionInfo describes one saved snapshot of a file.
type RevisionInfo struct {
	Path      string    // absolute path to the snapshot file
	Timestamp time.Time // moment the snapshot was taken
	Size      int64     // bytes
}

// CreateRevision copies the current contents of filePath into a sibling
// ".revisions/" directory under a timestamped name. It is a no-op if the
// source file cannot be read.
func CreateRevision(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file for revision: %w", err)
	}

	revDir := filepath.Join(filepath.Dir(filePath), revisionsDirName)
	if err := os.MkdirAll(revDir, 0o755); err != nil {
		return fmt.Errorf("creating revisions dir: %w", err)
	}

	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	revName := fmt.Sprintf("%s.%d%s", name, time.Now().UnixNano(), ext)

	return os.WriteFile(filepath.Join(revDir, revName), data, 0o644)
}

// ListRevisions returns all revisions of filePath ordered newest first.
// Returns an empty slice when no revisions exist.
func ListRevisions(filePath string) []RevisionInfo {
	revDir := filepath.Join(filepath.Dir(filePath), revisionsDirName)
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	prefix := name + "."

	entries, err := os.ReadDir(revDir)
	if err != nil {
		return nil
	}

	var out []RevisionInfo
	for _, entry := range entries {
		n := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(n, prefix) || !strings.HasSuffix(n, ext) {
			continue
		}
		middle := strings.TrimSuffix(strings.TrimPrefix(n, prefix), ext)
		var ts int64
		if _, err := fmt.Sscanf(middle, "%d", &ts); err != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		// Timestamps written by CreateRevision are unix-nanoseconds; older
		// callers may have used unix-seconds. Detect which by magnitude.
		stamp := time.Unix(0, ts)
		if ts < 1e12 {
			stamp = time.Unix(ts, 0)
		}
		out = append(out, RevisionInfo{
			Path:      filepath.Join(revDir, n),
			Timestamp: stamp,
			Size:      info.Size(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.After(out[j].Timestamp) })
	return out
}

// RestoreRevision overwrites filePath with the contents of revisionPath.
// The current contents of filePath are snapshotted first so a restore is
// itself reversible.
func RestoreRevision(filePath, revisionPath string) error {
	if err := CreateRevision(filePath); err != nil {
		return fmt.Errorf("snapshotting current before restore: %w", err)
	}
	data, err := os.ReadFile(revisionPath)
	if err != nil {
		return fmt.Errorf("reading revision: %w", err)
	}
	return SafeWrite(filePath, data, SafeWriteOptions{})
}

// PruneRevisions deletes all but the newest maxCount revisions.
func PruneRevisions(filePath string, maxCount int) error {
	revs := ListRevisions(filePath)
	if len(revs) <= maxCount {
		return nil
	}
	for _, r := range revs[maxCount:] {
		if err := os.Remove(r.Path); err != nil {
			return err
		}
	}
	return nil
}
