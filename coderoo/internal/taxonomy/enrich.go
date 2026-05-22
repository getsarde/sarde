package taxonomy

import (
	"fmt"
	"sort"

	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/internal/engine"
)

// EnrichTaxonomies applies term metadata from data/*.yml, validates undefined
// tags per the configured strictness mode, and sorts term pages by date.
// Returns a list of warnings and an error (for "error" strictness mode).
func EnrichTaxonomies(
	taxonomies map[string]*engine.Taxonomy,
	taxCfg map[string]config.TaxonomyConfig,
	projectDir string,
) ([]string, error) {
	var warnings []string

	for name, tax := range taxonomies {
		meta, err := LoadTermMetadata(projectDir, name)
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
