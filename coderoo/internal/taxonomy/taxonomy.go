// Package taxonomy aggregates tags, categories, and other taxonomies across all pages.
package taxonomy

import (
	"github.com/coderoo-dev/coderoo/internal/content"
	"github.com/coderoo-dev/coderoo/internal/engine"
)

// BuildTaxonomies walks all pages and aggregates taxonomy terms.
// Returns a map of taxonomy name → Taxonomy (e.g., "tags", "categories").
func BuildTaxonomies(pages []*engine.Page) map[string]*engine.Taxonomy {
	taxonomies := map[string]*engine.Taxonomy{
		"tags": {
			Name:      "tags",
			Singular:  "tag",
			Terms:     make(map[string]*engine.TaxonomyTerm),
			Permalink: "/tags/",
		},
		"categories": {
			Name:      "categories",
			Singular:  "category",
			Terms:     make(map[string]*engine.TaxonomyTerm),
			Permalink: "/categories/",
		},
	}

	for _, page := range pages {
		for _, tag := range page.Tags {
			addTerm(taxonomies["tags"], tag, page)
		}
		for _, cat := range page.Categories {
			addTerm(taxonomies["categories"], cat, page)
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
