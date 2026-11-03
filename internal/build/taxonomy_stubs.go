package build

import (
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
	sardetemplate "github.com/getsarde/sarde/internal/template"
)

func buildTaxonomyIndexStub(tax *engine.Taxonomy, termEntries []*engine.TermEntry, lang string) *engine.Page {
	barePermalink := "/" + tax.Name + "/"
	return &engine.Page{
		PageIdentity: engine.PageIdentity{
			Title:        tax.Name,
			Kind:         engine.KindTaxonomy,
			Permalink:    tax.Permalink,
			RelPermalink: barePermalink,
		},
		PageI18n: engine.PageI18n{Lang: lang},
		Params: map[string]any{
			consts.TaxonomyKey:    tax,
			consts.TermEntriesKey: termEntries,
		},
	}
}

func buildTermStub(tax *engine.Taxonomy, term *engine.TaxonomyTerm, lang string) *engine.Page {
	barePermalink := "/" + tax.Name + "/" + term.Slug + "/"
	return &engine.Page{
		PageIdentity: engine.PageIdentity{
			Title:        term.Label,
			Kind:         engine.KindTerm,
			Permalink:    term.Permalink,
			RelPermalink: barePermalink,
		},
		PageI18n: engine.PageI18n{Lang: lang},
		Params: map[string]any{
			consts.TaxonomyKey:     tax,
			consts.TaxonomyTermKey: term,
		},
	}
}

func buildTermPaginatedStub(tax *engine.Taxonomy, term *engine.TaxonomyTerm, permalink string, n int, lang string) *engine.Page {
	barePermalink := sardetemplate.PaginationURL("/"+tax.Name+"/"+term.Slug+"/", n)
	return &engine.Page{
		PageIdentity: engine.PageIdentity{
			Title:        term.Label,
			Kind:         engine.KindTerm,
			Permalink:    permalink,
			RelPermalink: barePermalink,
		},
		PageI18n: engine.PageI18n{Lang: lang},
		Params: map[string]any{
			consts.TaxonomyKey:          tax,
			consts.TaxonomyTermKey:      term,
			consts.PaginationCurrentKey: n,
		},
	}
}

// crossLinkStubs returns the other languages' stubs for key, in langOrder
// (the configured language order), so the language switcher on a taxonomy
// page lists them in the same order on every build.
func crossLinkStubs(byLang map[string]map[string]*engine.Page, key, selfLang string, langOrder []string) []*engine.Page {
	var peers []*engine.Page
	for _, lang := range langOrder {
		if lang == selfLang {
			continue
		}
		if stub, ok := byLang[lang][key]; ok {
			peers = append(peers, stub)
		}
	}
	return peers
}
