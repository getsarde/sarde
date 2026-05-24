// Package taxonomy aggregates tags, categories, and other taxonomies across all pages.
package taxonomy

import (
	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/engine"
)

// BuildTaxonomies walks all pages and aggregates taxonomy terms.
// Returns a map of taxonomy name → Taxonomy (e.g., "tags", "categories").
// The taxCfg map drives which taxonomies are built; if nil or empty, defaults
// to tags + categories for backward compatibility.
func BuildTaxonomies(pages []*engine.Page, taxCfg map[string]config.TaxonomyConfig) map[string]*engine.Taxonomy {
	if len(taxCfg) == 0 {
		taxCfg = map[string]config.TaxonomyConfig{
			"tags":       {Singular: "tag"},
			"categories": {Singular: "category"},
		}
	}

	taxonomies := make(map[string]*engine.Taxonomy, len(taxCfg))
	for name, tc := range taxCfg {
		singular := tc.Singular
		if singular == "" {
			singular = name
		}
		taxonomies[name] = &engine.Taxonomy{
			Name:       name,
			Singular:   singular,
			Terms:      make(map[string]*engine.TaxonomyTerm),
			Permalink:  "/" + name + "/",
			PaginateBy: tc.PaginateBy,
		}
	}

	for _, page := range pages {
		if tax, ok := taxonomies["tags"]; ok {
			for _, tag := range page.Tags {
				addTerm(tax, tag, page)
			}
		}
		if tax, ok := taxonomies["categories"]; ok {
			for _, cat := range page.Categories {
				addTerm(tax, cat, page)
			}
		}
	}

	// Remove empty taxonomies.
	for name, tax := range taxonomies {
		if len(tax.Terms) == 0 {
			delete(taxonomies, name)
		}
	}

	return taxonomies
}

// addTerm adds a page to a taxonomy term, creating the term if needed.
func addTerm(tax *engine.Taxonomy, termName string, page *engine.Page) {
	slug := content.Slugify(termName)
	term, ok := tax.Terms[slug]
	if !ok {
		term = &engine.TaxonomyTerm{
			Name:      termName,
			Slug:      slug,
			Permalink: tax.Permalink + slug + "/",
		}
		tax.Terms[slug] = term
	}
	term.Pages = append(term.Pages, page)
}
