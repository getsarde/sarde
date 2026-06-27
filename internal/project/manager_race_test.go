package project

import (
	"net/http"
	"sync"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/build"
)

// stubPreview satisfies PreviewServer; Start blocks like the real dev server.
type stubPreview struct {
	stop  chan struct{}
	ready chan int
}

// newStubPreview returns a stub whose Ready() immediately yields port.
func newStubPreview(port int) *stubPreview {
	s := &stubPreview{stop: make(chan struct{}), ready: make(chan int, 1)}
	s.ready <- port
	close(s.ready)
	return s
}

func (s *stubPreview) Start() error      { <-s.stop; return http.ErrServerClosed }
func (s *stubPreview) Stop() error       { close(s.stop); return nil }
func (s *stubPreview) Ready() <-chan int { return s.ready }

// TestProjectManager_PreviewFactory_LifecycleRace exercises the builder
// factory closure (as the dev server's Rebuilder would, from watcher timer
// goroutines) while the project is updated, closed, and re-opened. Before the
// deps-snapshot fix, this raced on pm.config and panicked on nil config when
// a straggler factory call ran after CloseProject. Run with -race.
func TestProjectManager_PreviewFactory_LifecycleRace(t *testing.T) {
	dir := createTestProject(t)
	hub := NewEventHub()

	var captured func() *build.SiteBuilder
	factory := func(projectDir, outputDir string, port int, liveReload bool, builderFactory func() *build.SiteBuilder) PreviewServer {
		captured = builderFactory
		return newStubPreview(port)
	}

	pm := NewProjectManager(hub, embedded.ThemeFS(), factory)
	if _, err := pm.OpenProject(dir); err != nil {
		t.Fatalf("OpenProject failed: %v", err)
	}
	if _, err := pm.StartPreview(0); err != nil {
		t.Fatalf("StartPreview failed: %v", err)
	}
	if captured == nil {
		t.Fatal("preview factory did not receive a builder factory")
	}

	// Hammer the factory closure from a background goroutine, simulating
	// watcher-triggered rebuilds racing with lifecycle operations.
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
			}
			if b := captured(); b == nil {
				t.Error("builder factory returned nil")
				return
			}
		}
	}()

	title1, title2 := "Race Title A", "Race Title B"
	for i := 0; i < 25; i++ {
		titles := []*string{&title1, &title2}
		if err := pm.UpdateSettings(SettingsInput{Title: titles[i%2]}); err != nil {
			t.Fatalf("UpdateSettings failed: %v", err)
		}
	}

	if err := pm.CloseProject(); err != nil {
		t.Fatalf("CloseProject failed: %v", err)
	}

	// Straggler case: factory calls after close must not panic and must
	// return a builder constructed from the last-known-good snapshot.
	for i := 0; i < 10; i++ {
		if b := captured(); b == nil {
			t.Fatal("straggler factory call returned nil after CloseProject")
		}
	}

	// Re-open and confirm the factory picks up the new project.
	if _, err := pm.OpenProject(dir); err != nil {
		t.Fatalf("re-OpenProject failed: %v", err)
	}
	if b := captured(); b == nil {
		t.Fatal("factory returned nil after re-open")
	}

	close(done)
	wg.Wait()
	pm.CloseProject()
}

// TestProjectManager_ValidateBuild_CloseOpenRace hammers Validate/Build while
// the project is closed and re-opened. Asserts no race report and no panic.
func TestProjectManager_ValidateBuild_CloseOpenRace(t *testing.T) {
	dir := createTestProject(t)
	hub := NewEventHub()
	pm := NewProjectManager(hub, embedded.ThemeFS(), nil)

	if _, err := pm.OpenProject(dir); err != nil {
		t.Fatalf("OpenProject failed: %v", err)
	}

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
			}
			// "no project open" errors are expected mid-close; the test
			// only asserts the absence of races and panics.
			_, _ = pm.Validate()
		}
	}()

	for i := 0; i < 20; i++ {
		if err := pm.CloseProject(); err != nil {
			t.Fatalf("CloseProject failed: %v", err)
		}
		if _, err := pm.OpenProject(dir); err != nil {
			t.Fatalf("OpenProject failed: %v", err)
		}
	}
	if _, err := pm.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	close(done)
	wg.Wait()
}

// TestProjectManager_DepsSnapshotInvariants checks the snapshot stays in
// lockstep with the authoritative fields and survives CloseProject.
func TestProjectManager_DepsSnapshotInvariants(t *testing.T) {
	dir := createTestProject(t)
	hub := NewEventHub()
	pm := NewProjectManager(hub, embedded.ThemeFS(), nil)

	if pm.deps.Load() != nil {
		t.Error("snapshot should be nil before any project is opened")
	}

	if _, err := pm.OpenProject(dir); err != nil {
		t.Fatalf("OpenProject failed: %v", err)
	}
	d := pm.deps.Load()
	if d == nil {
		t.Fatal("snapshot nil after OpenProject")
	}
	if d.config != pm.GetConfig() {
		t.Error("snapshot config does not match GetConfig() after open")
	}
	if d.projectDir != pm.ProjectDir() {
		t.Errorf("snapshot dir = %q, want %q", d.projectDir, pm.ProjectDir())
	}

	newTitle := "Snapshot Title"
	if err := pm.UpdateSettings(SettingsInput{Title: &newTitle}); err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}
	d = pm.deps.Load()
	if d.config != pm.GetConfig() {
		t.Error("snapshot config does not track GetConfig() after UpdateSettings")
	}
	if d.config.Site.Title != newTitle {
		t.Errorf("snapshot title = %q, want %q", d.config.Site.Title, newTitle)
	}

	if err := pm.CloseProject(); err != nil {
		t.Fatalf("CloseProject failed: %v", err)
	}
	d = pm.deps.Load()
	if d == nil {
		t.Fatal("snapshot must survive CloseProject (last-known-good)")
	}
	if d.config == nil || d.projectDir == "" {
		t.Error("snapshot must retain last-known-good deps after close")
	}
	if pm.ProjectDir() != "" {
		t.Errorf("ProjectDir() = %q, want empty after close", pm.ProjectDir())
	}
}
