package server

import (
	"time"

	"github.com/coderoo-dev/coderoo/internal/build"
	"github.com/coderoo-dev/coderoo/internal/engine"
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
type Rebuilder struct {
	builderFactory func() *build.SiteBuilder
	projectDir     string
}

// NewRebuilder creates a Rebuilder with the given factory function.
// The factory is called on each rebuild to get a fresh SiteBuilder instance.
func NewRebuilder(factory func() *build.SiteBuilder, projectDir string) *Rebuilder {
	return &Rebuilder{builderFactory: factory, projectDir: projectDir}
}

// Rebuild runs a full site build and returns the result.
func (r *Rebuilder) Rebuild() *RebuildResult {
	start := time.Now()

	builder := r.builderFactory()
	result, err := builder.Build()

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
