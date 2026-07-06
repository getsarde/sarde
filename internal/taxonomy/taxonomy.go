// Package taxonomy aggregates tags, categories, and other taxonomies across all pages.
package taxonomy

import (
	"crypto/sha256"
	"fmt"

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
//
// The second return value is a list of warnings collected while adding terms
// (e.g. two differently-named terms colliding on the same slug).
func BuildTaxonomies(pages []*engine.Page, taxCfg map[string]config.TaxonomyConfig, lang string) (map[string]*engine.Taxonomy, []string) {
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

	var warnings []string
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
				if w := addTerm(tax, term, page); w != "" {
					warnings = append(warnings, w)
				}
			}
		}
	}

	// Remove empty taxonomies.
	for name, tax := range taxonomies {
		if len(tax.Terms) == 0 {
			delete(taxonomies, name)
		}
	}

	return taxonomies, warnings
}

// addTerm adds a page to a taxonomy term, creating the term if needed.
// Returns a non-empty warning when termName slugifies to the same slug as an
// already-registered term with a different display name (e.g. "C++" and "C#"
// both slugify to "c") — in that case the existing term silently absorbs the
// page instead of the collision being reported, unless we surface it here.
func addTerm(tax *engine.Taxonomy, termName string, page *engine.Page) string {
	slug := content.Slugify(termName)
	if slug == "" {
		// termName has no ASCII alphanumeric characters (all punctuation, CJK,
		// Cyrillic, etc.), so Slugify collapses it to "". Left as-is, this
		// would key the term under "" and its permalink (tax.Permalink + "/")
		// would collide with the taxonomy index page. Fall back to a short
		// hash of the original name instead.
		slug = fmt.Sprintf("%x", sha256.Sum256([]byte(termName)))[:8]
	}

	var warning string
	term, ok := tax.Terms[slug]
	if !ok {
		term = &engine.TaxonomyTerm{
			Name:      termName,
			Slug:      slug,
			Permalink: tax.Permalink + slug + "/",
		}
		tax.Terms[slug] = term
	} else if term.Name != termName {
		warning = fmt.Sprintf("taxonomy %q: terms %q and %q collide on slug %q", tax.Name, term.Name, termName, slug)
	}
	term.Pages = append(term.Pages, page)
	return warning
}
