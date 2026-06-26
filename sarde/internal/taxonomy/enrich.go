package taxonomy

import (
	"fmt"
	"sort"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

// EnrichTaxonomies applies term metadata from data/*.yml, validates undefined
// tags per the configured strictness mode, and sorts term pages by date.
// When lang is non-empty, per-language overlays (data/<name>.<lang>.yml) are
// loaded on top of the base file.
// Returns a list of warnings and an error (for "error" strictness mode).
func EnrichTaxonomies(
	taxonomies map[string]*engine.Taxonomy,
	taxCfg map[string]config.TaxonomyConfig,
	projectDir string,
	lang string,
) ([]string, error) {
	var warnings []string

	for name, tax := range taxonomies {
		meta, err := LoadTermMetadataForLang(projectDir, name, lang)
		if err != nil {
			return warnings, fmt.Errorf("loading term metadata for %q: %w", name, err)
		}

		// Apply metadata to terms.
		for slug, term := range tax.Terms {
			if meta != nil {
				if m, ok := meta[slug]; ok {
					if m.Label != "" {
						term.Label = m.Label
					}
					term.Description = m.Description
					term.Color = m.Color
					term.Icon = m.Icon
					term.Hidden = m.Hidden
					term.Priority = m.Priority
					term.Difficulty = m.Difficulty
					term.ContentType = m.ContentType
					if m.Permalink != "" {
						term.CustomSlug = m.Permalink
					}
				}
			}
			if term.Label == "" {
				term.Label = term.Name
			}

			// Sort pages within each term by date descending.
			sort.Slice(term.Pages, func(i, j int) bool {
				return term.Pages[i].Date.After(term.Pages[j].Date)
			})
		}

		// Re-key terms that have a custom slug from the permalink field.
		warnings = append(warnings, reKeyTerms(tax)...)

		// Validate undefined tags against metadata definitions.
		cfg := taxCfg[name]
		mode := cfg.UndefinedTags
		if mode == "" {
			mode = "warn"
		}
		if meta == nil && mode == "error" {
			warnings = append(warnings, fmt.Sprintf("taxonomy %q: undefined_tags is \"error\" but no data/%s.yml file exists", name, name))
		}
		if meta != nil && mode != "ignore" {
			w, err := validateUndefinedTerms(name, tax, meta, mode)
			warnings = append(warnings, w...)
			if err != nil {
				return warnings, err
			}
		}
	}

	return warnings, nil
}

// reKeyTerms updates terms that have a CustomSlug set from the permalink
// metadata field. The term is removed from the old key and re-inserted
// under the new slug, and its Slug/Permalink are updated. When two terms
// resolve to the same slug, the displaced term's pages are merged into the
// surviving term (deduped) and a warning is returned instead of silently
// dropping it.
func reKeyTerms(tax *engine.Taxonomy) []string {
	var warnings []string
	var toReKey []struct {
		oldSlug string
		newSlug string
	}
	for slug, term := range tax.Terms {
		if term.CustomSlug != "" && term.CustomSlug != slug {
			toReKey = append(toReKey, struct {
				oldSlug string
				newSlug string
			}{slug, term.CustomSlug})
		}
	}
	// Map iteration order is random; sort so collision outcomes are
	// deterministic across builds.
	sort.Slice(toReKey, func(i, j int) bool { return toReKey[i].oldSlug < toReKey[j].oldSlug })

	// Phase 1: remove every re-keyed term first, so a term vacating a slug
	// does not falsely collide with another term moving into it.
	moved := make(map[string]*engine.TaxonomyTerm, len(toReKey))
	for _, rk := range toReKey {
		moved[rk.oldSlug] = tax.Terms[rk.oldSlug]
		delete(tax.Terms, rk.oldSlug)
	}

	// Phase 2: re-insert under the new slugs, merging pages on collision.
	for _, rk := range toReKey {
		term := moved[rk.oldSlug]
		if existing, ok := tax.Terms[rk.newSlug]; ok {
			existing.Pages = mergeTermPages(existing.Pages, term.Pages)
			warnings = append(warnings, fmt.Sprintf(
				"taxonomy %q: permalink %q of %q collides with an existing slug — merged %d page(s) into the existing entry (check data/%s.yml)",
				tax.Name, rk.newSlug, term.Name, len(term.Pages), tax.Name))
			continue
		}
		term.Slug = rk.newSlug
		term.Permalink = "/" + tax.Name + "/" + rk.newSlug + "/"
		tax.Terms[rk.newSlug] = term
	}
	return warnings
}

// mergeTermPages unions two page lists, deduped by page identity, sorted by
// date descending (the order EnrichTaxonomies establishes per term).
func mergeTermPages(a, b []*engine.Page) []*engine.Page {
	seen := make(map[*engine.Page]struct{}, len(a)+len(b))
	merged := make([]*engine.Page, 0, len(a)+len(b))
	for _, list := range [][]*engine.Page{a, b} {
		for _, p := range list {
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			merged = append(merged, p)
		}
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].Date.After(merged[j].Date) })
	return merged
}

// validateUndefinedTerms checks for terms found in pages but not in metadata.
func validateUndefinedTerms(
	taxName string,
	tax *engine.Taxonomy,
	meta map[string]TermMetadata,
	mode string,
) ([]string, error) {
	var warnings []string

	for slug, term := range tax.Terms {
		if _, defined := meta[slug]; defined {
			continue
		}
		if term.CustomSlug != "" {
			if _, defined := meta[term.CustomSlug]; defined {
				continue
			}
		}
		msg := fmt.Sprintf("taxonomy %q: term %q is not defined in data/%s.yml", taxName, term.Name, taxName)
		switch mode {
		case "error":
			return warnings, fmt.Errorf("%s", msg)
		case "warn":
			warnings = append(warnings, msg)
		case "create":
			// Auto-created terms keep their defaults (label = name).
		}
	}

	return warnings, nil
}
