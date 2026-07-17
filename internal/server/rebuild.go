package server

import (
	"sync"
	"time"

	"github.com/getsarde/sarde/internal/build"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/engine"
)

// RebuildResult holds the outcome of a rebuild attempt.
type RebuildResult struct {
	Success      bool
	Duration     time.Duration
	PageCount    int
	Warnings     []engine.ValidationWarning
	PhaseTimings []engine.PhaseTiming
	LogMessages  []engine.BuildLogEntry
	Error        error
}

// Rebuilder wraps SiteBuilder for dev-mode rebuilds.
// It persists the builder across content/public changes and only creates
// a new one when config or templates change.
//
// Coalescing: if a rebuild is already running, incoming requests are merged
// into a single pending change via mergeChanges (kinds escalate, content
// paths union), so no change is lost while at most one follow-up rebuild
// runs. This prevents cascading queued rebuilds when editors emit rapid
// successive save events.
type Rebuilder struct {
	builderFactory func() *build.SiteBuilder
	projectDir     string
	builder        *build.SiteBuilder

	mu      sync.Mutex
	running bool
	pending *FileChange
}

// NewRebuilder creates a Rebuilder with the given factory function.
// The factory is called on the first build and whenever config/template changes
// require a fresh SiteBuilder.
func NewRebuilder(factory func() *build.SiteBuilder, projectDir string) *Rebuilder {
	return &Rebuilder{builderFactory: factory, projectDir: projectDir}
}

// Rebuild runs a site build and returns the executed change with its result.
// Config or template changes create a fresh builder (full re-init).
// Content or public changes reuse the existing builder (template engine skips Load).
//
// If a rebuild is already in progress, the change is merged into the pending
// slot and this call returns a nil result (the caller should skip
// logging/broadcasting). When the active rebuild finishes, it picks up the
// pending change automatically and runs it before returning; the returned
// change is the one the final result actually belongs to.
func (r *Rebuilder) Rebuild(change FileChange) (FileChange, *RebuildResult) {
	r.mu.Lock()
	if r.running {
		if r.pending != nil {
			merged := mergeChanges([]FileChange{*r.pending, change})
			r.pending = &merged
		} else {
			r.pending = &change
		}
		r.mu.Unlock()
		return change, nil
	}
	r.running = true
	r.mu.Unlock()

	result := r.executeBuild(change)

	for {
		r.mu.Lock()
		next := r.pending
		r.pending = nil
		if next == nil {
			r.running = false
			r.mu.Unlock()
			return change, result
		}
		r.mu.Unlock()

		change = *next
		result = r.executeBuild(change)
	}
}

// executeBuild performs the actual build under the assumption that the caller
// has set r.running = true.
func (r *Rebuilder) executeBuild(change FileChange) *RebuildResult {
	start := time.Now()

	switch change.Kind {
	case ChangeConfig, ChangeTemplate, ChangePlugin:
		devlog.Log("build", "Full rebuild (%s change): new builder", change.Kind)
		r.builder = r.builderFactory()

	case ChangeContent:
		if r.builder != nil {
			paths := change.Paths
			if len(paths) == 0 {
				paths = []string{change.Path}
			}
			devlog.Log("build", "Incremental content rebuild: %d file(s)", len(paths))
			result, err := r.builder.ContentRebuild(paths)
			if err != nil {
				return &RebuildResult{
					Success:  false,
					Duration: time.Since(start),
					Error:    err,
				}
			}
			return &RebuildResult{
				Success:      true,
				Duration:     result.Duration,
				PageCount:    result.PageCount,
				Warnings:     result.Warnings,
				PhaseTimings: result.PhaseTimings,
				LogMessages:  result.LogMessages,
			}
		}
		r.builder = r.builderFactory()

	default:
		if r.builder == nil {
			r.builder = r.builderFactory()
		} else {
			devlog.Log("build", "Incremental rebuild (%s change): reusing builder", change.Kind)
		}
	}

	result, err := r.builder.Build()

	if err != nil {
		return &RebuildResult{
			Success:  false,
			Duration: time.Since(start),
			Error:    err,
		}
	}

	return &RebuildResult{
		Success:      true,
		Duration:     result.Duration,
		PageCount:    result.PageCount,
		Warnings:     result.Warnings,
		PhaseTimings: result.PhaseTimings,
		LogMessages:  result.LogMessages,
	}
}

// ToReloadMessage converts a file change and rebuild result into a ReloadMessage
// suitable for broadcasting to connected browsers.
func ToReloadMessage(change FileChange, result *RebuildResult, projectDir string) ReloadMessage {
	var changedAt int64
	if !change.DetectedAt.IsZero() {
		changedAt = change.DetectedAt.UnixMilli()
	}

	if result.Error != nil {
		msg := ReloadMessage{
			Type:      ReloadError,
			Error:     result.Error.Error(),
			ChangedAt: changedAt,
		}
		// Try to extract structured error info for the overlay.
		if be := build.ParseBuildError(result.Error, projectDir); be != nil {
			if be.File != "" {
				msg.File = be.File
			}
			if be.Line > 0 {
				msg.Line = be.Line
			}
			if be.Col > 0 {
				msg.Col = be.Col
			}
			if be.Frame != "" {
				msg.Frame = be.Frame
			}
			if be.Message != "" {
				msg.Error = be.Message
			}
		}
		return msg
	}

	if change.Kind == ChangeCSS {
		return ReloadMessage{
			Type:      ReloadCSS,
			Path:      change.Path,
			ChangedAt: changedAt,
		}
	}

	return ReloadMessage{
		Type:      ReloadFull,
		ChangedAt: changedAt,
	}
}
