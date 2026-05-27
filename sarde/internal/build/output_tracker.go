package build

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/frostybee/sarde/internal/outputpath"
	"github.com/frostybee/sarde/internal/workers"
	"golang.org/x/sync/errgroup"
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

// TrackBatch records multiple paths as written in a single lock acquisition.
func (t *OutputTracker) TrackBatch(paths []string) {
	t.mu.Lock()
	for _, p := range paths {
		if p != "" {
			t.written[filepath.Clean(p)] = struct{}{}
		}
	}
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

	// Snapshot the written set under a single lock, using the pre-resolved
	// outputRoot to avoid per-path filepath.Abs calls.
	t.mu.Lock()
	written := make(map[string]struct{}, len(t.written))
	for path := range t.written {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if outputpath.IsWithin(outputRoot, absPath) {
			written[filepath.Clean(absPath)] = struct{}{}
		}
	}
	t.mu.Unlock()

	// Walk to collect orphans and directories. The walk itself is
	// inherently serial, but we defer deletions for parallel execution.
	var emptyDirs []string
	var toDelete []string
	err = filepath.WalkDir(outputRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil
		}
		if !outputpath.IsWithin(outputRoot, absPath) {
			return fmt.Errorf("path escapes output directory: %s", path)
		}
		if d.IsDir() {
			emptyDirs = append(emptyDirs, path)
			return nil
		}
		if _, ok := written[filepath.Clean(absPath)]; !ok {
			toDelete = append(toDelete, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Delete orphaned files in parallel.
	if len(toDelete) > 0 {
		g := new(errgroup.Group)
		g.SetLimit(workers.IOLimit(len(toDelete)))
		for _, p := range toDelete {
			g.Go(func() error {
				return outputpath.RemoveIfWithinAbs(outputRoot, p)
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}
	}

	// Remove empty directories bottom-up (ordering matters).
	for i := len(emptyDirs) - 1; i >= 0; i-- {
		_ = outputpath.RemoveIfWithinAbs(outputRoot, emptyDirs[i])
	}
	return nil
}
