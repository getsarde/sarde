package project

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/getsarde/sarde/internal/build"
	"github.com/getsarde/sarde/internal/buildlock"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/outputpath"
)

// Sentinel errors for preview lifecycle, so API handlers can map failures to
// distinct error codes instead of guessing from free-text messages.
var (
	ErrNoProjectOpen      = errors.New("no project open")
	ErrPreviewRunning     = errors.New("preview already running")
	ErrPreviewNotRunning  = errors.New("preview not running")
	ErrPreviewUnavailable = errors.New("preview not available (no preview factory configured)")
)

// PreviewServer is the interface for the dev server to avoid circular imports.
type PreviewServer interface {
	Start() error
	Stop() error
	// Ready reports the actual bound port: sent once after the listener
	// binds, then the channel is closed. Closed without a value if the
	// server fails or is stopped before binding.
	Ready() <-chan int
}

// PreviewFactory creates a PreviewServer. Set by the caller to break the import cycle.
type PreviewFactory func(projectDir, outputDir string, port int, liveReload bool, builderFactory func() *build.SiteBuilder) PreviewServer

// StartPreview starts the dev server and returns the actual bound port.
func (pm *ProjectManager) StartPreview(port int) (int, error) {
	ds, err := pm.installPreview(port)
	if err != nil {
		return 0, err
	}

	startErr := make(chan error, 1)
	go func() {
		err := ds.Start()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			pm.mu.Lock()
			// Only tear down if this server is still the active one; a
			// Stop + restart may have installed a new devServer since.
			// This teardown path bypasses stopPreviewLocked, so the output
			// lock must be released here too or its reference leaks for the
			// process lifetime.
			if pm.devServer == ds {
				pm.devServer = nil
				pm.releasePreviewLockLocked()
				pm.state = StateOpen
			}
			pm.mu.Unlock()
			pm.eventHub.Broadcast(Event{Type: "preview:error", Data: map[string]any{"error": err.Error()}})
		}
		startErr <- err // sent after teardown so the waiter observes cleaned-up state
	}()

	// Wait for the bind LOCK-FREE: the initial build runs inside Start and
	// must not block other ProjectManager operations.
	actualPort, ok := <-ds.Ready()
	if !ok {
		// Start returned before binding; its error is authoritative.
		err := <-startErr
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return 0, fmt.Errorf("preview stopped before it became ready")
		}
		return 0, err
	}

	pm.mu.Lock()
	if pm.devServer != ds {
		pm.mu.Unlock()
		return 0, fmt.Errorf("preview was stopped while starting")
	}
	pm.mu.Unlock()

	pm.eventHub.Broadcast(Event{Type: "preview:started", Data: map[string]any{"port": actualPort}})
	return actualPort, nil
}

// installPreview validates state, constructs the dev server via the factory,
// and registers it as the active preview — all under pm.mu. The blocking
// wait for the bound port happens in StartPreview, outside the lock.
func (pm *ProjectManager) installPreview(port int) (PreviewServer, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.state == StateClosed {
		return nil, ErrNoProjectOpen
	}
	if pm.devServer != nil {
		return nil, ErrPreviewRunning
	}

	if port == 0 {
		port = pm.config.Server.Port
	}
	if port == 0 {
		port = consts.DefaultPort
	}

	// Checked before acquiring the output lock: everything after Acquire
	// must be infallible or the lock reference would leak.
	if pm.previewFactory == nil {
		return nil, ErrPreviewUnavailable
	}

	outputDir, err := outputpath.ResolveOutputDir(pm.projectDir, pm.config.Build.Output)
	if err != nil {
		return nil, err
	}

	// Hold the single-instance output lock for the preview's lifetime so a
	// concurrent sarde process targeting the same dist/ fails fast instead
	// of silently corrupting fingerprinted assets.
	lock, err := buildlock.Acquire(outputDir, "sidecar-preview")
	if err != nil {
		return nil, err
	}
	pm.previewLock = lock

	ds := pm.previewFactory(pm.projectDir, outputDir, port, config.BoolVal(pm.config.Server.LiveReload, true), func() *build.SiteBuilder {
		// Freshness barrier: UpdateSettings writes sarde.yaml and publishes
		// the re-resolved snapshot inside one pm.mu critical section. The
		// watcher's config-change rebuild can fire (debounce) before that
		// section ends; loading under RLock orders this rebuild after the
		// publish so it builds with the settings that triggered it. No
		// deadlock: Stop() never joins the watcher timer goroutine, so a
		// writer holding pm.mu always completes.
		pm.mu.RLock()
		d := pm.deps.Load()
		pm.mu.RUnlock()
		return pm.newBuilder(d)
	})

	pm.devServer = ds
	pm.state = StatePreviewing
	return ds, nil
}

// StopPreview stops the dev server.
func (pm *ProjectManager) StopPreview() error {
	pm.mu.Lock()
	if pm.devServer == nil {
		pm.mu.Unlock()
		return ErrPreviewNotRunning
	}
	err := pm.stopPreviewLocked()
	pm.mu.Unlock()

	pm.eventHub.Broadcast(Event{Type: "preview:stopped"})
	return err
}

// stopPreviewLocked stops the dev server and clears preview state. Caller must
// hold pm.mu and is responsible for broadcasting "preview:stopped" after unlocking.
func (pm *ProjectManager) stopPreviewLocked() error {
	err := pm.devServer.Stop()
	pm.devServer = nil
	pm.releasePreviewLockLocked()
	pm.state = StateOpen
	return err
}

// releasePreviewLockLocked releases the preview's output lock reference if
// one is held. Caller must hold pm.mu. Idempotent: safe when no lock is set.
func (pm *ProjectManager) releasePreviewLockLocked() {
	if pm.previewLock != nil {
		pm.previewLock.Release()
		pm.previewLock = nil
	}
}
