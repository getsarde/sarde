package deploy

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/consts"
)

// fakeEnv returns a Getenv backed by a map, so tests never touch the real
// environment and can run in parallel.
func fakeEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

// writeDist creates an output directory with the given files and always a
// root .sarde.lock, which no deployer may ship.
func writeDist(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, consts.FileOutputLock), []byte("lock"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// recordingReporter keeps everything a deployer reported.
type recordingReporter struct {
	mu       sync.Mutex
	steps    []Step
	progress []Progress
	logs     []string
}

func (r *recordingReporter) Step(s Step, _ string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.steps = append(r.steps, s)
}

func (r *recordingReporter) Progress(p Progress) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.progress = append(r.progress, p)
}

func (r *recordingReporter) Log(level, msg string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, level+": "+msg)
}

// stubSleep makes retries and polling instant and records the waits.
func stubSleep(t *testing.T) *[]time.Duration {
	t.Helper()
	var mu sync.Mutex
	var waits []time.Duration
	prev := apiSleep
	apiSleep = func(ctx context.Context, d time.Duration) error {
		mu.Lock()
		waits = append(waits, d)
		mu.Unlock()
		return ctx.Err()
	}
	t.Cleanup(func() { apiSleep = prev })
	return &waits
}
