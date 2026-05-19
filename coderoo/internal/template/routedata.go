package template

import (
	"fmt"
	"strings"

	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/internal/consts"
	"github.com/coderoo-dev/coderoo/internal/engine"
	"github.com/coderoo-dev/coderoo/internal/navigation"
)

// paginationCurrentKey is an internal Params sentinel set by the builder on
// synthesized pagination pages (e.g. /blog/page/2/) so BuildRouteData knows
// which page index to render.
const paginationCurrentKey = "__pagination_current"

// BuildRouteData constructs the unified RouteData context for a page render.
func BuildRouteData(page *engine.Page, site *engine.SiteContext, theme *engine.ThemeConfig) *engine.RouteData {
	rd := &engine.RouteData{
		Page:   page,
		Site:   site,
		Theme:  theme,
		Layout: engine.LayoutDefault,
		Lang:   resolveLang(page, site),
		Dir:    resolveDir(page, site),
	}

	// Build translation links from page.Translations
	if len(page.Translations) > 0 {
		rd.Translations = make([]engine.TranslationLink, 0, len(page.Translations)+1)
		// Include current page in translations list
		rd.Translations = append(rd.Translations, engine.TranslationLink{
			Lang:  page.Lang,
			URL:   page.RelPermalink,
			Title: page.Title,
		})
		for _, tr := range page.Translations {
			rd.Translations = append(rd.Translations, engine.TranslationLink{
				Lang:  tr.Lang,
				URL:   tr.RelPermalink,
				Title: tr.Title,
			})
		}
	}

	col := page.Collection
	if col != nil {
		rd.Collection = col

		// Layout from collection config
		if col.Config != nil {
			rd.Layout = col.Config.Layout
		}

		// Resolve template name
		rd.Template = resolveTemplateName(page, col)

		// Pagination from prev/next
		if page.PrevPage != nil || page.NextPage != nil {
			rd.Pagination = &engine.PaginationLinks{}
			if page.PrevPage != nil {
				rd.Pagination.Prev = &engine.PaginationLink{
					URL:   page.PrevPage.RelPermalink,
					Title: page.PrevPage.Title,
				}
			}
			if page.NextPage != nil {
				rd.Pagination.Next = &engine.PaginationLink{
					URL:   page.NextPage.RelPermalink,
					Title: page.NextPage.Title,
				}
			}
		}

		// Section detection for _index.md pages
		if page.Kind == engine.KindSection {
			rd.IsSection = true
			rd.Section = page.Section
		}

		// Numbered pagination for list pages (section index with Paginate > 0).
		if rd.IsSection && col.Config != nil && col.Config.Paginate > 0 {
			current := 1
			if page.Params != nil {
				if n, ok := page.Params[paginationCurrentKey].(int); ok && n > 0 {
					current = n
				}
			}
			rd.Paginator = buildPaginator(col, current)
		}

		// Sidebar and navigation for layouts with sidebar
		if engine.LayoutHasSidebar(rd.Layout) {
			rd.SidebarType = "nav"
			rd.HasSidebar = true
			// Use per-language nav tree if available, else fall back to default
			navTree := col.NavTree
			if col.NavTrees != nil && page.Lang != "" {
				if langTree, ok := col.NavTrees[page.Lang]; ok {
					navTree = langTree
				}
			}
			if navTree != nil {
				rd.Sidebar = navigation.MarkActive(navTree, page)
			}
			rd.Breadcrumbs = navigation.BuildBreadcrumbs(page, col)
		} else {
			rd.SidebarType = "none"
		}
	} else {
		// No collection — standalone or home page
		switch page.Kind {
		case engine.KindHome:
			rd.Template = "home"
			rd.Layout = engine.LayoutDefault
			if site != nil && site.Config != nil {
				if cfg, ok := site.Config.(*config.SiteConfig); ok {
					rd.Homepage = mapHomepageSettings(&cfg.Homepage)
				}
			}
		default:
			rd.Template = consts.DirDefault + "/single"
		}
		rd.SidebarType = "none"
	}

	// GlobalNav (collections + config header links)
	var headerLinks []config.NavLink
	if siteCfg, ok := site.Config.(*config.SiteConfig); ok && siteCfg != nil {
		headerLinks = siteCfg.Header.Links
	}
	rd.GlobalNav = navigation.BuildGlobalNav(site, col, headerLinks)

	return rd
}

// resolveTemplateName determines which template to use for a page.
// Priority: frontmatter "template" field > collection/kind convention.
func resolveTemplateName(page *engine.Page, col *engine.Collection) string {
	// Check frontmatter template override via Params
	if page.Params != nil {
		if tmpl, ok := page.Params["template"].(string); ok && tmpl != "" {
			return tmpl
		}
	}

	prefix := col.Name
	switch page.Kind {
	case engine.KindSection:
		return prefix + "/list"
	case engine.KindHome:
		return "home"
	default:
		return prefix + "/single"
	}
}

// resolveLang determines the language code for a page's RouteData.
func resolveLang(page *engine.Page, site *engine.SiteContext) string {
	if page.Lang != "" {
		return page.Lang
	}
	if site != nil && site.Language != "" {
		return site.Language
	}
	return "en"
}

// resolveDir determines the text direction for a page's RouteData.
func resolveDir(page *engine.Page, site *engine.SiteContext) string {
	lang := resolveLang(page, site)
	if site != nil && site.Config != nil {
		if cfg, ok := site.Config.(*config.SiteConfig); ok {
			if lc, ok := cfg.I18n.Languages[lang]; ok && lc.Dir != "" {
				return lc.Dir
			}
		}
	}
	return "ltr"
}

// buildPaginator computes the Paginator for a list page of a paginated collection.
// current is the 1-based index of the page being rendered.
func buildPaginator(col *engine.Collection, current int) *engine.Paginator {
	perPage := col.Config.Paginate
	// Only include rendered pages (exclude section index).
	var pages []*engine.Page
	for _, p := range col.Pages {
		if p.Kind != engine.KindSection {
			pages = append(pages, p)
		}
	}
	total := (len(pages) + perPage - 1) / perPage
	if total < 1 {
		total = 1
	}
	if current < 1 {
		current = 1
	}
	if current > total {
		current = total
	}
	start := (current - 1) * perPage
	end := start + perPage
	if end > len(pages) {
		end = len(pages)
	}

	base := paginationBaseURL(col)
	p := &engine.Paginator{
		CurrentPages: pages[start:end],
		Current:      current,
		Total:        total,
	}
	p.Pages = make([]engine.PaginationLink, 0, total)
	for i := 1; i <= total; i++ {
		p.Pages = append(p.Pages, engine.PaginationLink{
			URL:   paginationURL(base, i),
			Title: fmt.Sprintf("%d", i),
		})
	}
	if current > 1 {
		p.HasPrev = true
		p.PrevURL = paginationURL(base, current-1)
	}
	if current < total {
		p.HasNext = true
		p.NextURL = paginationURL(base, current+1)
	}
	return p
}

// paginationBaseURL returns the index URL for a collection, e.g. "/blog/".
func paginationBaseURL(col *engine.Collection) string {
	if col == nil {
		return "/"
	}
	if col.IndexPage != nil && col.IndexPage.RelPermalink != "" {
		return col.IndexPage.RelPermalink
	}
	return "/" + col.Name + "/"
}

// paginationURL returns the URL for the Nth pagination page of a collection.
// Page 1 maps to the collection's base URL; N>1 maps to "<base>page/N/".
func paginationURL(base string, n int) string {
	if n <= 1 {
		return base
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return fmt.Sprintf("%spage/%d/", base, n)
}

// mapHomepageSettings converts config.HomepageSettings to engine.HomepageData.
func mapHomepageSettings(s *config.HomepageSettings) *engine.HomepageData {
	d := &engine.HomepageData{
		Template: s.Template,
		Hero: engine.HeroData{
			Title:      s.Hero.Title,
			Subtitle:   s.Hero.Subtitle,
			Background: s.Hero.Background,
		},
	}
	if s.Hero.CTA != nil {
		d.Hero.CTA = &engine.HeroCTAData{
			Label: s.Hero.CTA.Label,
			URL:   s.Hero.CTA.URL,
		}
	}
	return d
}
