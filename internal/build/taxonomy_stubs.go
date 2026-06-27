package build

import (
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
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
	barePermalink := "/" + tax.Name + "/" + term.Slug + "/"
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

func crossLinkStubs(byLang map[string]map[string]*engine.Page, key, selfLang string) []*engine.Page {
	var peers []*engine.Page
	for lang, stubs := range byLang {
		if lang == selfLang {
			continue
		}
		if stub, ok := stubs[key]; ok {
			peers = append(peers, stub)
		}
	}
	return peers
}
