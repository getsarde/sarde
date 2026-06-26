package build

import (
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
	sardetemplate "github.com/getsarde/sarde/internal/template"
)

func collectionName(p *engine.Page) string {
	if p != nil && p.Collection != nil {
		return p.Collection.Name
	}
	return ""
}

func paramValue(params map[string]any, key string) any {
	if params == nil {
		return nil
	}
	return params[key]
}

func boolParam(params map[string]any, key string) bool {
	if v, ok := paramValue(params, key).(bool); ok {
		return v
	}
	return false
}

func preserveStablePageState(next, old *engine.Page) {
	next.Collection = old.Collection
	next.Section = old.Section
	next.PrevPage = old.PrevPage
	next.NextPage = old.NextPage
	next.Siblings = old.Siblings
	next.NavNode = old.NavNode
	next.Version = old.Version
	next.VersionRelPath = old.VersionRelPath
	next.VersionPeers = old.VersionPeers
	next.Translations = old.Translations
}

func replacePagePointer(pages *[]*engine.Page, old, next *engine.Page) {
	if pages == nil {
		return
	}
	for i, p := range *pages {
		if p == old || (p != nil && p.FilePath == old.FilePath && !p.IsFallback) {
			(*pages)[i] = next
		}
	}
}

func copyRenderedContentToFallbacks(pages []*engine.Page, changed map[string]*engine.Page, idx *content.PageIndex) {
	for _, p := range pages {
		if !p.IsFallback || p.FilePath == "" {
			continue
		}
		src := changed[p.FilePath]
		if src == nil {
			continue
		}
		p.Content = src.Content
		p.Headings = src.Headings
		p.HasCodeBlocks = src.HasCodeBlocks
		p.HasImages = src.HasImages
		setPageIndexHeadings(idx, p)
	}
}

func appendTaxonomyStubsForLang(pages []*engine.Page, taxonomies map[string]*engine.Taxonomy, cfg map[string]config.TaxonomyConfig, lang string) []*engine.Page {
	for taxName, tax := range taxonomies {
		if !cfg[taxName].ShouldRender() {
			continue
		}
		barePermalink := "/" + taxName + "/"
		pages = append(pages, &engine.Page{
			PageIdentity: engine.PageIdentity{
				Title: tax.Name, Kind: engine.KindTaxonomy,
				Permalink: tax.Permalink, RelPermalink: barePermalink,
			},
			PageI18n: engine.PageI18n{Lang: lang},
		})
		paginateBy := tax.PaginateBy
		if paginateBy <= 0 {
			paginateBy = consts.DefaultPaginateBy
		}
		for _, term := range tax.Terms {
			bareTermPermalink := barePermalink + term.Slug + "/"
			pages = append(pages, &engine.Page{
				PageIdentity: engine.PageIdentity{
					Title: term.Label, Kind: engine.KindTerm,
					Permalink: term.Permalink, RelPermalink: bareTermPermalink,
				},
				PageI18n: engine.PageI18n{Lang: lang},
			})
			total := (len(term.Pages) + paginateBy - 1) / paginateBy
			for n := 2; n <= total; n++ {
				paginatedPermalink := sardetemplate.PaginationURL(term.Permalink, n)
				pages = append(pages, &engine.Page{
					PageIdentity: engine.PageIdentity{
						Title: term.Label, Kind: engine.KindTerm,
						Permalink: paginatedPermalink, RelPermalink: bareTermPermalink,
					},
					PageI18n: engine.PageI18n{Lang: lang},
					Params:   map[string]any{consts.PaginationCurrentKey: n},
				})
			}
		}
	}
	return pages
}
