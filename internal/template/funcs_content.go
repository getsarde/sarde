package template

import (
	htmltemplate "html/template"
	"sort"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
)

// buildContentFuncs returns cross-collection content lookup functions: ref,
// relref, recentEntries, findEntry, allCollections, and topTerms.
func buildContentFuncs(pageIndexPtr **content.PageIndex, sitePtr **engine.SiteContext) htmltemplate.FuncMap {
	return htmltemplate.FuncMap{
		"ref": func(slug string) string {
			if p := lookupPage(*pageIndexPtr, *sitePtr, slug); p != nil {
				return p.Permalink
			}
			return slug
		},
		"relref": func(slug string) string {
			if p := lookupPage(*pageIndexPtr, *sitePtr, slug); p != nil {
				return p.RelPermalink
			}
			return slug
		},
		"recentEntries": func(colName string, n int) []*engine.Page {
			s := *sitePtr
			if s == nil {
				return nil
			}
			col, ok := s.Collections[colName]
			if !ok || col == nil {
				return nil
			}
			pages := col.Pages
			// Clamp both ends: a negative n from template arithmetic would
			// panic with a runtime error that escapes template recovery and
			// kills the whole build.
			if n < 0 {
				n = 0
			}
			if n > len(pages) {
				n = len(pages)
			}
			return pages[:n]
		},
		"findEntry": func(colName, slug string) *engine.Page {
			s := *sitePtr
			if s == nil {
				return nil
			}
			col, ok := s.Collections[colName]
			if !ok || col == nil {
				return nil
			}
			for _, p := range col.Pages {
				if p.Slug == slug {
					return p
				}
			}
			return nil
		},
		"allCollections": func() map[string]*engine.Collection {
			s := *sitePtr
			if s == nil {
				return nil
			}
			return s.Collections
		},

		// termURL is overridden in funcMapForLang to inject the rendering page's language.
		"topTerms": func(taxonomyName string, n int) []*engine.TaxonomyTerm {
			s := *sitePtr
			if s == nil {
				return nil
			}
			tax, ok := s.Taxonomies[taxonomyName]
			if !ok || tax == nil {
				return nil
			}
			terms := make([]*engine.TaxonomyTerm, 0, len(tax.Terms))
			for _, t := range tax.Terms {
				if !t.Hidden {
					terms = append(terms, t)
				}
			}
			sort.Slice(terms, func(i, j int) bool {
				if len(terms[i].Pages) != len(terms[j].Pages) {
					return len(terms[i].Pages) > len(terms[j].Pages)
				}
				return terms[i].Slug < terms[j].Slug
			})
			if n > 0 && n < len(terms) {
				terms = terms[:n]
			}
			return terms
		},
	}
}

// fnVersionOf returns the version peer of page matching versionID, or nil if
// page is nil or no matching peer exists.
func fnVersionOf(page *engine.Page, versionID string) *engine.Page {
	if page == nil {
		return nil
	}
	for _, peer := range page.VersionPeers {
		if peer.Version == versionID {
			return peer
		}
	}
	return nil
}
