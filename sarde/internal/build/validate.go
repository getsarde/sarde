package build

import (
	"fmt"
	"time"

	"github.com/frostybee/sarde/internal/collection"
	"github.com/frostybee/sarde/internal/engine"
)

// Validate runs phases 1-4 (Initialize, Discover, Parse, Assemble) without rendering or writing.
func (b *SiteBuilder) Validate() (*ValidateResult, error) {
	start := time.Now()

	contentDir := b.resolveContentDir()

	// i18n: configure scanner for multi-language detection
	if b.config.I18n.IsMultiLang() {
		langCodes := make(map[string]bool)
		for code := range b.config.I18n.Languages {
			langCodes[code] = true
		}
		b.scanner.Languages = langCodes
		b.scanner.DefaultLang = b.config.I18n.GetDefaultLanguage()
	}

	files, err := b.scanner.DiscoverFiles(contentDir)
	if err != nil {
		return nil, fmt.Errorf("discovering content: %w", err)
	}

	collections, warnings, err := collection.BuildCollections(files, b.config, contentDir)
	if err != nil {
		return nil, fmt.Errorf("building collections: %w", err)
	}

	standalones, err := collection.BuildStandalonePages(files, contentDir, b.config.Content.SummaryLength, string(b.config.Build.LastUpdated))
	if err != nil {
		return nil, fmt.Errorf("building standalone pages: %w", err)
	}

	var allPages []*engine.Page
	for _, col := range collections {
		allPages = append(allPages, col.Pages...)
	}
	allPages = append(allPages, standalones...)

	return &ValidateResult{
		PageCount:   len(allPages),
		Collections: len(collections),
		Warnings:    warnings,
		Pages:       allPages,
		Duration:    time.Since(start),
	}, nil
}

// ValidateResult holds the outcome of a validation run (no rendering or writing).
type ValidateResult struct {
	PageCount   int
	Collections int
	Warnings    []engine.ValidationWarning
	Pages       []*engine.Page
	Duration    time.Duration
}
