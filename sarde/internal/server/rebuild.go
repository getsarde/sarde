package server

import (
	"log"
	"sync"
	"time"

	"github.com/frostybee/sarde/internal/build"
	"github.com/frostybee/sarde/internal/engine"
)

// RebuildResult holds the outcome of a rebuild attempt.
type RebuildResult struct {
	Success   bool
	Duration  time.Duration
	PageCount int
	Warnings  []engine.ValidationWarning
	Error     error
}

// Rebuilder wraps SiteBuilder for dev-mode rebuilds.
// It persists the builder across content/static changes and only creates
// a new one when config or templates change.
type Rebuilder struct {
	builderFactory func() *build.SiteBuilder
	projectDir     string
	builder        *build.SiteBuilder
	mu             sync.Mutex
}

// NewRebuilder creates a Rebuilder with the given factory function.
// The factory is called on the first build and whenever config/template changes
// require a fresh SiteBuilder.
func NewRebuilder(factory func() *build.SiteBuilder, projectDir string) *Rebuilder {
	return &Rebuilder{builderFactory: factory, projectDir: projectDir}
}

// Rebuild runs a site build and returns the result.
// Config or template changes create a fresh builder (full re-init).
// Content or static changes reuse the existing builder (template engine skips Load).
func (r *Rebuilder) Rebuild(change FileChange) *RebuildResult {
	r.mu.Lock()
	defer r.mu.Unlock()

	start := time.Now()

	switch change.Kind {
	case ChangeConfig, ChangeTemplate:
		log.Printf("Full rebuild (%s change) — new builder", change.Kind)
		r.builder = r.builderFactory()

	case ChangeContent:
		if r.builder != nil {
			paths := change.Paths
			if len(paths) == 0 {
				paths = []string{change.Path}
			}
			log.Printf("Incremental content rebuild — %d file(s)", len(paths))
			result, err := r.builder.ContentRebuild(paths)
			if err != nil {
				return &RebuildResult{
					Success:  false,
					Duration: time.Since(start),
					Error:    err,
				}
			}
			return &RebuildResult{
				Success:   true,
				Duration:  result.Duration,
				PageCount: result.PageCount,
				Warnings:  result.Warnings,
			}
		}
		r.builder = r.builderFactory()

	default:
		if r.builder == nil {
			r.builder = r.builderFactory()
		} else {
			log.Printf("Incremental rebuild (%s change) — reusing builder", change.Kind)
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
		Success:   true,
		Duration:  result.Duration,
		PageCount: result.PageCount,
		Warnings:  result.Warnings,
	}
}

// ToReloadMessage converts a file change and rebuild result into a ReloadMessage
// suitable for broadcasting to connected browsers.
func ToReloadMessage(change FileChange, result *RebuildResult, projectDir string) ReloadMessage {
	if result.Error != nil {
		msg := ReloadMessage{
			Type:  ReloadError,
			Error: result.Error.Error(),
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
			Type: ReloadCSS,
			Path: change.Path,
		}
	}

	if len(result.Warnings) > 0 {
		// Send reload with a follow-up warning.
		return ReloadMessage{
			Type: ReloadFull,
		}
	}

	return ReloadMessage{
		Type: ReloadFull,
	}
}
