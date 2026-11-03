package build

import (
	"strings"

	"github.com/getsarde/sarde/internal/engine"
	sardetemplate "github.com/getsarde/sarde/internal/template"
)

// knownOutputURLs returns a lookup of every page URL the build writes, base
// path included: the pages themselves and their aliases, the taxonomy and
// term pages with their numbered pages, the numbered list pages of paginated
// collections, the latest-version copies a versioned collection publishes,
// and folders under public/ that hold an index.html. The link report runs
// before the taxonomy and list pages are rendered, so it uses this to tell a
// link to an output-only page from one to nothing.
func (b *SiteBuilder) knownOutputURLs(s *buildState) func(string) bool {
	r := b.urlResolver
	if r == nil {
		return nil
	}
	known := make(map[string]struct{}, len(s.allPages)*2)
	add := func(u string) {
		known[strings.TrimSuffix(u, "/")] = struct{}{}
	}

	for _, p := range s.allPages {
		if !p.Renders() {
			continue
		}
		add(r.URL(p.RelPermalink, p.Lang, resolvePageVersion(p)))
		for _, alias := range p.Aliases {
			add(r.URL(alias, "", ""))
		}
		if col := p.Collection; col != nil && col.Config != nil {
			if vc := col.Config.Versioning; vc != nil && vc.Enabled && vc.PublishLatestAtVersionURL && p.Version == vc.LastVersion {
				add(r.URL(p.RelPermalink, p.Lang, vc.LastVersion))
			}
		}
	}

	for _, name := range sortedKeys(s.collections) {
		col := s.collections[name]
		for _, lang := range listLangs(col) {
			for _, stub := range paginationStubs(col, lang) {
				add(r.URL(stub.RelPermalink, stub.Lang, ""))
			}
		}
	}

	taxByLang := s.taxByLang
	if taxByLang == nil && s.siteCtx != nil {
		taxByLang = map[string]map[string]*engine.Taxonomy{"": s.siteCtx.Taxonomies}
	}
	for _, langTax := range taxByLang {
		for name, tax := range langTax {
			if tax == nil || !b.config.Taxonomies[name].ShouldRender() {
				continue
			}
			add(tax.Permalink)
			for _, term := range tax.Terms {
				add(term.Permalink)
				for n := 2; n <= tax.TermPageCount(term); n++ {
					add(sardetemplate.PaginationURL(term.Permalink, n))
				}
			}
		}
	}

	basePath := strings.TrimSuffix(r.BasePath, "/")
	return func(urlPath string) bool {
		if _, ok := known[strings.TrimSuffix(urlPath, "/")]; ok {
			return true
		}
		// A folder under public/ served by its index.html.
		rel := strings.TrimPrefix(urlPath, basePath)
		if !strings.HasPrefix(rel, "/") {
			rel = "/" + rel
		}
		return s.pageIndex != nil && s.pageIndex.HasAsset(strings.TrimSuffix(rel, "/")+"/index.html")
	}
}
