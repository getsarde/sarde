package build

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// OutputTracker records every file written during a build so orphaned
// files from previous builds can be pruned without a full RemoveAll.
type OutputTracker struct {
	mu      sync.Mutex
	written map[string]struct{}
}

// NewOutputTracker creates an OutputTracker pre-sized for the expected
// number of output files.
func NewOutputTracker(sizeHint int) *OutputTracker {
	return &OutputTracker{
		written: make(map[string]struct{}, sizeHint),
	}
}

// Track records a path as written. Thread-safe.
func (t *OutputTracker) Track(path string) {
	cleaned := filepath.Clean(path)
	t.mu.Lock()
	t.written[cleaned] = struct{}{}
	t.mu.Unlock()
}

// Prune walks outputDir and removes any file not in the written set,
// then removes empty directories bottom-up.
func (t *OutputTracker) Prune(outputDir string) error {
	var emptyDirs []string
	err := filepath.WalkDir(outputDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			emptyDirs = append(emptyDirs, path)
			return nil
		}
		if _, ok := t.written[filepath.Clean(path)]; !ok {
			os.Remove(path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for i := len(emptyDirs) - 1; i >= 0; i-- {
		os.Remove(emptyDirs[i])
	}
	return nil
}
