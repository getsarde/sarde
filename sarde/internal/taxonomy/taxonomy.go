// Package taxonomy aggregates tags, categories, and other taxonomies across all pages.
package taxonomy

import (
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
)

// BuildTaxonomies walks all pages and aggregates taxonomy terms.
// Returns a map of taxonomy name → Taxonomy (e.g., "tags", "categories").
// The taxCfg map drives which taxonomies are built; if nil or empty, defaults
// to tags + categories for backward compatibility.
//
// lang scopes aggregation to a single language: when non-empty, pages whose
// Lang is set and differs from lang are skipped. This prevents a translated
// post (and its generated fallbacks) from being counted once per language on
// the same term page — taxonomy pages render once, at the default-language
// URLs. Pass "" to include every page (single-language sites, where Lang is
// empty).
func BuildTaxonomies(pages []*engine.Page, taxCfg map[string]config.TaxonomyConfig, lang string) map[string]*engine.Taxonomy {
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
		// Skip other-language translations so a post isn't listed once per
		// language on a term page. Pages with no detected language are always
		// included (single-language sites, undetected content).
		if lang != "" && page.Lang != "" && page.Lang != lang {
			continue
		}
		for name, tax := range taxonomies {
			var terms []string
			switch name {
			case "tags":
				terms = page.Tags
			case "categories":
				terms = page.Categories
			default:
				terms = page.Extra[name]
			}
			for _, term := range terms {
				addTerm(tax, term, page)
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
