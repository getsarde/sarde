package build

import (
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/frostybee/sarde/internal/outputpath"
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
	outputRoot, err := filepath.Abs(outputDir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(outputRoot); err != nil {
		return err
	}
	t.mu.Lock()
	written := make(map[string]struct{}, len(t.written))
	for path := range t.written {
		if safe, err := outputpath.EnsureWithinOutputDir(outputRoot, path); err == nil {
			written[filepath.Clean(safe)] = struct{}{}
		}
	}
	t.mu.Unlock()

	var emptyDirs []string
	err = filepath.WalkDir(outputRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if _, err := outputpath.EnsureWithinOutputDir(outputRoot, path); err != nil {
			return err
		}
		if d.IsDir() {
			emptyDirs = append(emptyDirs, path)
			return nil
		}
		if _, ok := written[filepath.Clean(path)]; !ok {
			if err := outputpath.RemoveIfWithin(outputRoot, path); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for i := len(emptyDirs) - 1; i >= 0; i-- {
		_ = outputpath.RemoveIfWithin(outputRoot, emptyDirs[i])
	}
	return nil
}
