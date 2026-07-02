package project

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/build"
	"github.com/getsarde/sarde/internal/buildlock"
)

// A preview holds the output lock for its lifetime; a Build during the
// preview re-enters the same lock (refcount) instead of failing, and the
// lock fully releases only when the preview stops.
func TestProjectManager_PreviewAndBuild_LockReentrant(t *testing.T) {
	dir := createTestProject(t)
	outDir := filepath.Join(dir, "dist")
	hub := NewEventHub()
	factory := func(projectDir, outputDir string, port int, liveReload bool, bf func() *build.SiteBuilder) PreviewServer {
		return newStubPreview(9999)
	}
	pm := NewProjectManager(hub, embedded.ThemeFS(), factory)
	if _, err := pm.OpenProject(dir); err != nil {
		t.Fatalf("OpenProject: %v", err)
	}

	if n := buildlock.RefCount(outDir); n != 0 {
		t.Fatalf("RefCount before preview = %d, want 0", n)
	}
	if _, err := pm.StartPreview(0); err != nil {
		t.Fatalf("StartPreview: %v", err)
	}
	if n := buildlock.RefCount(outDir); n != 1 {
		t.Errorf("RefCount during preview = %d, want 1", n)
	}

	if _, err := pm.Build(); err != nil {
		t.Fatalf("Build during preview: %v", err)
	}
	if n := buildlock.RefCount(outDir); n != 1 {
		t.Errorf("RefCount after build during preview = %d, want 1 (build released its ref)", n)
	}

	if err := pm.StopPreview(); err != nil {
		t.Fatalf("StopPreview: %v", err)
	}
	if n := buildlock.RefCount(outDir); n != 0 {
		t.Errorf("RefCount after StopPreview = %d, want 0", n)
	}
}

// Regression: the async bind-failure teardown in StartPreview bypasses
// stopPreviewLocked, and must still release the lock reference. A leak here
// would keep the output dir locked for the process lifetime.
func TestProjectManager_FailedPreviewReleasesLock(t *testing.T) {
	dir := createTestProject(t)
	outDir := filepath.Join(dir, "dist")
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

	if _, err := pm.StartPreview(0); err == nil || !strings.Contains(err.Error(), "bind failed") {
		t.Fatalf("first StartPreview err = %v, want bind failure", err)
	}
	if n := buildlock.RefCount(outDir); n != 0 {
		t.Fatalf("RefCount after failed preview = %d, want 0 (leaked reference)", n)
	}

	if _, err := pm.StartPreview(0); err != nil {
		t.Fatalf("retry StartPreview: %v", err)
	}
	if err := pm.StopPreview(); err != nil {
		t.Fatalf("StopPreview: %v", err)
	}
	if n := buildlock.RefCount(outDir); n != 0 {
		t.Errorf("RefCount after retry and stop = %d, want 0", n)
	}
}

// Build without a preview acquires and fully releases the lock.
func TestProjectManager_BuildAcquiresAndReleasesLock(t *testing.T) {
	dir := createTestProject(t)
	outDir := filepath.Join(dir, "dist")
	hub := NewEventHub()
	pm := NewProjectManager(hub, embedded.ThemeFS(), nil)
	if _, err := pm.OpenProject(dir); err != nil {
		t.Fatalf("OpenProject: %v", err)
	}

	if _, err := pm.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if n := buildlock.RefCount(outDir); n != 0 {
		t.Errorf("RefCount after Build = %d, want 0", n)
	}
}
