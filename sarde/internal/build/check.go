package build

import (
	"time"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/links"
)

// CheckOptions controls the behavior of the Check method.
type CheckOptions struct {
	External     bool   // enable external URL probing
	Strict       bool   // override all policies to "error"
	ReportFormat string // override report format ("pretty" | "json" | "github-actions")
}

// CheckResult holds the outcome of a link-check-only run.
type CheckResult struct {
	PageCount int
	LinkCount int
	Lanes     int
	HasErrors bool
	Findings  []links.Finding
	Warnings  []engine.ValidationWarning
	Output    string // formatted report output
	Summary   string
	Duration  time.Duration
}

// Check runs the build pipeline through link validation and returns the
// report without rendering templates or writing HTML to disk.
func (b *SiteBuilder) Check(opts CheckOptions) (*CheckResult, error) {
	// Snapshot the validation settings before applying the per-call
	// overrides; b.config is shared (e.g. with a reused dev-server builder),
	// so the overrides must not leak past this call. A value copy suffices:
	// the overrides below only REPLACE fields (strings and fresh BoolPtr
	// pointers), never write through existing pointers or into the slices.
	saved := b.config.LinkValidation

	if opts.External {
		b.config.LinkValidation.External.Check = config.BoolPtr(true)
	}
	if opts.Strict {
		b.config.LinkValidation.OnBroken = "error"
		b.config.LinkValidation.OnBrokenAnchor = "error"
		b.config.LinkValidation.OnRelativeLinks = "error"
		b.config.LinkValidation.OnLocalLinks = "error"
		b.config.LinkValidation.OnUnverifiedInternal = "error"
		b.config.LinkValidation.External.OnBroken = "error"
		b.config.LinkValidation.SameSitePolicy = "error"
	}
	if opts.ReportFormat != "" {
		b.config.LinkValidation.Report = opts.ReportFormat
	}
	b.config.LinkValidation.Enabled = config.BoolPtr(true)

	b.checkOnly = true
	defer func() {
		b.checkOnly = false
		b.config.LinkValidation = saved
	}()

	buildResult, buildErr := b.Build()

	result := &CheckResult{
		Duration: time.Duration(0),
	}
	if buildResult != nil {
		result.PageCount = buildResult.PageCount
		result.Duration = buildResult.Duration
		result.Warnings = buildResult.Warnings
	}
	result.LinkCount = b.lastCoverage.TotalLinks
	result.Lanes = b.lastCoverage.TotalLanes

	if rr := b.checkReportResult; rr != nil {
		result.HasErrors = rr.HasErrors
		result.Findings = rr.Findings
		result.Output = rr.Output
		result.Summary = rr.Summary
	}

	// Build() returns an error when link validation finds errors.
	// Don't propagate that as a Check() error — the caller uses HasErrors.
	if buildErr != nil && result.HasErrors {
		return result, nil
	}
	return result, buildErr
}
