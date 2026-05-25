package asset

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestPipeline_WriteBundledFilesWithOptions_ParallelTracksOutputs(t *testing.T) {
	outDir := t.TempDir()
	p := &Pipeline{}
	for i := 0; i < 12; i++ {
		name := "bundle-" + string(rune('a'+i)) + ".css"
		p.bundledFiles = append(p.bundledFiles, BundledFile{
			Name:      name,
			OutputURL: "/assets/css/" + name,
			Content:   []byte("body { color: red; }\n"),
		})
	}

	var mu sync.Mutex
	tracked := make(map[string]struct{})
	err := p.WriteBundledFilesWithOptions(outDir, func(path string) {
		mu.Lock()
		tracked[path] = struct{}{}
		mu.Unlock()
	}, WriteOptions{Parallel: true, WorkerCount: 4})
	if err != nil {
		t.Fatalf("WriteBundledFilesWithOptions failed: %v", err)
	}
	if len(tracked) != len(p.bundledFiles) {
		t.Fatalf("tracked %d files, want %d", len(tracked), len(p.bundledFiles))
	}
	for path := range tracked {
		rel, err := filepath.Rel(outDir, path)
		if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			t.Fatalf("tracked path escaped output dir: %s", path)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("tracked file missing: %s: %v", path, err)
		}
	}
}

func TestPipeline_WriteBundledFilesWithOptions_RejectsEscapingPath(t *testing.T) {
	outDir := t.TempDir()
	p := &Pipeline{
		bundledFiles: []BundledFile{{
			Name:      "escape.css",
			OutputURL: "../escape.css",
			Content:   []byte("bad"),
		}},
	}
	if err := p.WriteBundledFilesWithOptions(outDir, nil, WriteOptions{Parallel: true, WorkerCount: 2}); err == nil {
		t.Fatal("expected escaping bundled path to fail")
	}
	if _, err := os.Stat(filepath.Join(outDir, "..", "escape.css")); !os.IsNotExist(err) {
		t.Fatalf("escaping file was created, stat err=%v", err)
	}
}
