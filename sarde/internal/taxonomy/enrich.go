package taxonomy

import (
	"fmt"
	"sort"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/engine"
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
		reKeyTerms(tax)

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
// under the new slug, and its Slug/Permalink are updated.
func reKeyTerms(tax *engine.Taxonomy) {
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
	for _, rk := range toReKey {
		term := tax.Terms[rk.oldSlug]
		delete(tax.Terms, rk.oldSlug)
		term.Slug = rk.newSlug
		term.Permalink = "/" + tax.Name + "/" + rk.newSlug + "/"
		tax.Terms[rk.newSlug] = term
	}
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
