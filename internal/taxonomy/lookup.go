package taxonomy

import (
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
)

// TermSlug is the key a term name is stored under in Taxonomy.Terms before
// any custom permalink from data/<taxonomy>.yml re-keys it: the name's slug,
// or a short hash of a name with nothing URL-safe in it ("日本語").
func TermSlug(name string) string {
	return content.SlugOrHash(name)
}

// LookupTerm finds the term a page names. The term stored under the name's
// slug is the usual answer; when its display name differs, a term re-keyed
// under a custom permalink may carry the name instead ("C++" moved to "cpp"
// while "C" owns "c"), so the display names are searched before settling
// for the slug's term (two names that collide on one slug share a term).
// Nil when the taxonomy has no such term.
func LookupTerm(tax *engine.Taxonomy, name string) *engine.TaxonomyTerm {
	if tax == nil {
		return nil
	}
	bySlug := tax.Terms[TermSlug(name)]
	if bySlug != nil && bySlug.Name == name {
		return bySlug
	}
	for _, term := range tax.Terms {
		if term.Name == name {
			return term
		}
	}
	return bySlug
}
