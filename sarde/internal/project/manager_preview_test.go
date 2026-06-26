package project

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/build"
)

// failingPreview fails before binding: Start returns an error immediately
// and Ready closes without a value.
type failingPreview struct{ ready chan int }

func newFailingPreview() *failingPreview { return &failingPreview{ready: make(chan int)} }

func (f *failingPreview) Start() error      { close(f.ready); return errors.New("bind failed") }
func (f *failingPreview) Stop() error       { return nil }
func (f *failingPreview) Ready() <-chan int { return f.ready }

// blockingPreview never becomes ready until it is stopped.
type blockingPreview struct {
	stop  chan struct{}
	ready chan int
}

func newBlockingPreview() *blockingPreview {
	return &blockingPreview{stop: make(chan struct{}), ready: make(chan int)}
}

func (b *blockingPreview) Start() error      { <-b.stop; close(b.ready); return http.ErrServerClosed }
func (b *blockingPreview) Stop() error       { close(b.stop); return nil }
func (b *blockingPreview) Ready() <-chan int { return b.ready }

// StartPreview must return the port the server actually bound, not the
// requested one.
func TestProjectManager_StartPreview_ReturnsActualPort(t *testing.T) {
	dir := createTestProject(t)
	hub := NewEventHub()
	factory := func(projectDir, outputDir string, port int, liveReload bool, bf func() *build.SiteBuilder) PreviewServer {
		return newStubPreview(9999)
	}
	pm := NewProjectManager(hub, embedded.ThemeFS(), factory)
	if _, err := pm.OpenProject(dir); err != nil {
		t.Fatalf("OpenProject: %v", err)
	}

	got, err := pm.StartPreview(0)
	if err != nil {
		t.Fatalf("StartPreview: %v", err)
	}
	if got != 9999 {
		t.Errorf("port = %d, want 9999 (actual bound port)", got)
	}
	if pm.State() != StatePreviewing {
		t.Errorf("state = %v, want Previewing", pm.State())
	}
	if err := pm.StopPreview(); err != nil {
		t.Fatalf("StopPreview: %v", err)
	}
}

// A pre-bind failure must surface synchronously, clean up, and allow a retry.
func TestProjectManager_StartPreview_FailsBeforeReady(t *testing.T) {
	dir := createTestProject(t)
	hub := NewEventHub()
	calls := 0
	factory := func(projectDir, outputDir string, port int, liveReload bool, bf func() *build.SiteBuilder) PreviewServer {
		calls++
		if calls == 1 {
			return newFailingPreview()
		}
		return newStubPreview(8080)
	}
	pm := NewProjectManager(hub, embedded.ThemeFS(), factory)
	if _, err := pm.OpenProject(dir); err != nil {
		t.Fatalf("OpenProject: %v", err)
	}

	_, err := pm.StartPreview(0)
	if err == nil || !strings.Contains(err.Error(), "bind failed") {
		t.Fatalf("err = %v, want the server's bind error", err)
	}
	if pm.State() != StateOpen {
		t.Errorf("state after failure = %v, want Open", pm.State())
	}

	// The failed server must have been deregistered, so a retry works.
	got, err := pm.StartPreview(0)
	if err != nil {
		t.Fatalf("retry StartPreview: %v", err)
	}
	if got != 8080 {
		t.Errorf("retry port = %d, want 8080", got)
	}
	pm.StopPreview()
}

// Stopping the preview while StartPreview is still waiting for the bind must
// yield a clean error, not a started event for a dead server.
func TestProjectManager_StopPreviewWhileStarting(t *testing.T) {
	dir := createTestProject(t)
	hub := NewEventHub()
	factory := func(projectDir, outputDir string, port int, liveReload bool, bf func() *build.SiteBuilder) PreviewServer {
		return newBlockingPreview()
	}
	pm := NewProjectManager(hub, embedded.ThemeFS(), factory)
	if _, err := pm.OpenProject(dir); err != nil {
		t.Fatalf("OpenProject: %v", err)
	}

	resCh := make(chan error, 1)
	go func() {
		_, err := pm.StartPreview(0)
		resCh <- err
	}()

	// Wait until the preview is installed before stopping it.
	deadline := time.Now().Add(5 * time.Second)
	for pm.State() != StatePreviewing {
		if time.Now().After(deadline) {
			t.Fatal("preview was never installed")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := pm.StopPreview(); err != nil {
		t.Fatalf("StopPreview: %v", err)
	}

	err := <-resCh
	if err == nil || !strings.Contains(err.Error(), "stopped before it became ready") {
		t.Errorf("StartPreview err = %v, want 'stopped before it became ready'", err)
	}
	if pm.State() != StateOpen {
		t.Errorf("final state = %v, want Open", pm.State())
	}
}
