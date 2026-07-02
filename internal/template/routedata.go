package template

import (
	htmltemplate "html/template"

	"github.com/getsarde/sarde/internal/collection"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/navigation"
)

// BuildRouteData constructs the unified RouteData context for a page render.
func BuildRouteData(page *engine.Page, site *engine.SiteContext, theme *engine.ThemeConfig) *engine.RouteData {
	rd := &engine.RouteData{
		Page:   page,
		Site:   site,
		Theme:  theme,
		Layout: engine.LayoutDefault,
		RouteI18n: engine.RouteI18n{
			Lang: resolveLang(page, site),
			Dir:  resolveDir(page, site),
		},
	}

	if len(page.Translations) > 0 {
		rd.Translations = buildTranslationLinks(page, buildLangMap(site), page.Translations)
	}
	if len(page.AllTranslations) > 0 {
		rd.AllTranslations = buildTranslationLinks(page, buildLangMap(site), page.AllTranslations)
	}

	col := page.Collection
	if col != nil {
		buildCollectionRouteData(rd, page, col)
	} else {
		buildStandaloneRouteData(rd, page, site)
	}

	// Per-page banner from frontmatter.
	if page.Params != nil {
		if b, ok := page.Params["banner"].(*engine.PageBanner); ok {
			rd.PageBanner = b
		}
	}

	// GlobalNav (collections + config header links)
	var headerLinks []config.NavLink
	if site != nil && site.Config != nil {
		if siteCfg, ok := site.Config.(*config.SiteConfig); ok {
			headerLinks = siteCfg.Header.Links
		}
	}
	rd.GlobalNav = navigation.BuildGlobalNav(site, col, headerLinks)

	return rd
}

// buildCollectionRouteData populates rd for a page that belongs to a
// collection: layout, template name, pagination, section detection,
// versioning, and sidebar/nav resolution.
func buildCollectionRouteData(rd *engine.RouteData, page *engine.Page, col *engine.Collection) {
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

	// Apply NavOverride link/label/disabled overrides from frontmatter.
	applyNavOverride(page.Params, "prev", &rd.Pagination, true)
	applyNavOverride(page.Params, "next", &rd.Pagination, false)

	// Section detection for _index.md pages
	if page.Kind == engine.KindSection {
		rd.IsSection = true
		rd.Section = page.Section
	}

	// Numbered pagination for list pages (section index with Paginate > 0).
	if rd.IsSection && col.Config != nil && col.Config.Paginate > 0 {
		current := 1
		if page.Params != nil {
			if n, ok := page.Params[consts.PaginationCurrentKey].(int); ok && n > 0 {
				current = n
			}
		}
		rd.Paginator = buildPaginator(col, current)
	}

	if col.Config != nil {
		if vc := col.Config.Versioning; vc != nil && vc.Enabled {
			rd.Version = page.Version
			rd.IsLatest = isLastVersion(page.Version, vc)
			rd.VersionLabel, rd.VersionBanner = versionLabelAndBanner(page.Version, vc)
			rd.Versions = buildVersionLinks(page, vc, col.Name)
		}
	}

	resolveSidebar(rd, col, page)
}

// buildStandaloneRouteData populates rd for a page with no collection:
// home, language-root section, taxonomy, term, or a generic standalone page.
func buildStandaloneRouteData(rd *engine.RouteData, page *engine.Page, site *engine.SiteContext) {
	// No collection — standalone, home, or taxonomy page
	switch page.Kind {
	case engine.KindHome:
		rd.Template = "home"
		rd.Layout = engine.LayoutDefault
		if site != nil && site.Config != nil {
			if cfg, ok := site.Config.(*config.SiteConfig); ok {
				rd.Homepage = mapHomepageSettings(&cfg.Homepage)
			}
		}
	case engine.KindSection:
		// Language-root _index.md (e.g. content/fr/_index.md) is KindSection
		// because its dir != "", but it should render identically to KindHome.
		if page.LangRelPath == "_index.md" {
			rd.Template = "home"
			rd.Layout = engine.LayoutDefault
			if site != nil && site.Config != nil {
				if cfg, ok := site.Config.(*config.SiteConfig); ok {
					rd.Homepage = mapHomepageSettings(&cfg.Homepage)
				}
			}
		} else {
			rd.Template = consts.DirDefault + "/single"
		}
	case engine.KindTaxonomy:
		rd.Template = consts.DirTaxonomy + "/list"
		rd.Layout = engine.LayoutDefault
		if tax, ok := page.Params[consts.TaxonomyKey].(*engine.Taxonomy); ok {
			rd.Taxonomy = tax
		}
		if entries, ok := page.Params[consts.TermEntriesKey].([]*engine.TermEntry); ok {
			rd.TermEntries = entries
		}
	case engine.KindTerm:
		rd.Template = consts.DirTaxonomy + "/term"
		rd.Layout = engine.LayoutDefault
		if tax, ok := page.Params[consts.TaxonomyKey].(*engine.Taxonomy); ok {
			rd.Taxonomy = tax
		}
		if term, ok := page.Params[consts.TaxonomyTermKey].(*engine.TaxonomyTerm); ok {
			rd.TaxonomyTerm = term
		}
		if rd.TaxonomyTerm != nil && rd.Taxonomy != nil {
			current := 1
			if n, ok := page.Params[consts.PaginationCurrentKey].(int); ok && n > 0 {
				current = n
			}
			paginateBy := rd.Taxonomy.PaginateBy
			if paginateBy <= 0 {
				paginateBy = consts.DefaultPaginateBy
			}
			rd.Paginator = buildTermPaginator(rd.TaxonomyTerm, paginateBy, current)
		}
	default:
		rd.Template = consts.DirDefault + "/single"
	}
	rd.SidebarType = "none"
}

func buildTranslationLinks(page *engine.Page, langMap map[string]engine.Language, sources []*engine.Page) []engine.TranslationLink {
	links := make([]engine.TranslationLink, 0, len(sources)+1)
	links = append(links, engine.TranslationLink{
		Lang:       page.Lang,
		Name:       langName(langMap, page.Lang),
		Dir:        langDir(langMap, page.Lang),
		URL:        page.RelPermalink,
		Title:      page.Title,
		IsFallback: page.IsFallback,
	})
	for _, tr := range sources {
		links = append(links, engine.TranslationLink{
			Lang:       tr.Lang,
			Name:       langName(langMap, tr.Lang),
			Dir:        langDir(langMap, tr.Lang),
			URL:        tr.RelPermalink,
			Title:      tr.Title,
			IsFallback: tr.IsFallback,
		})
	}
	return links
}

func resolveSidebar(rd *engine.RouteData, col *engine.Collection, page *engine.Page) {
	if !engine.LayoutHasSidebar(rd.Layout) {
		rd.SidebarType = "none"
		return
	}
	rd.SidebarType = "nav"
	rd.HasSidebar = true
	if col.Config != nil && col.Config.Sidebar != nil {
		rd.SidebarCollapsedByDefault = col.Config.Sidebar.CollapsedByDefault
	}

	if col.IsTabbed && col.CompositeTabSets != nil {
		key := collection.LangVersionKey(page.Lang, page.Version)
		tabs := col.CompositeTabSets[key]
		if len(tabs) > 0 {
			rd.IsTabbed = true
			rd.DocsTabs = tabs
			rd.ActiveTab = collection.FindTabForPage(tabs, page)
			if rd.ActiveTab == nil && len(tabs) > 0 {
				rd.ActiveTab = tabs[0]
			}
			if rd.ActiveTab != nil {
				if rd.ActiveTab.NavTree != nil {
					rd.Sidebar = navigation.MarkActive(rd.ActiveTab.NavTree, page)
				}
				rd.Breadcrumbs = navigation.BuildBreadcrumbsTabbed(page, col, rd.ActiveTab)
			}
		} else {
			rd.Breadcrumbs = navigation.BuildBreadcrumbsVersioned(page, col)
		}
	} else if col.IsTabbed && len(col.Tabs) > 0 {
		rd.IsTabbed = true
		rd.DocsTabs = col.Tabs
		rd.ActiveTab = collection.FindTabForPage(col.Tabs, page)
		if rd.ActiveTab == nil && len(col.Tabs) > 0 {
			rd.ActiveTab = col.Tabs[0]
		}
		if rd.ActiveTab != nil {
			navTree := rd.ActiveTab.NavTree
			if rd.ActiveTab.NavTrees != nil && page.Lang != "" {
				if langTree, ok := rd.ActiveTab.NavTrees[page.Lang]; ok {
					navTree = langTree
				}
			}
			if navTree != nil {
				rd.Sidebar = navigation.MarkActive(navTree, page)
			}
			rd.Breadcrumbs = navigation.BuildBreadcrumbsTabbed(page, col, rd.ActiveTab)
		}
	} else if col.CompositeNavTrees != nil {
		key := collection.LangVersionKey(page.Lang, page.Version)
		if navTree, ok := col.CompositeNavTrees[key]; ok && navTree != nil {
			rd.Sidebar = navigation.MarkActive(navTree, page)
		}
		rd.Breadcrumbs = navigation.BuildBreadcrumbsVersioned(page, col)
	} else {
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
	}
}

func applyNavOverride(params map[string]any, key string, pagination **engine.PaginationLinks, isPrev bool) {
	if params == nil {
		return
	}
	nav, ok := params[key].(*engine.NavOverride)
	if !ok || nav == nil {
		return
	}
	if nav.Disabled {
		if *pagination != nil {
			if isPrev {
				(*pagination).Prev = nil
			} else {
				(*pagination).Next = nil
			}
		}
		return
	}
	if nav.Link != "" {
		if *pagination == nil {
			*pagination = &engine.PaginationLinks{}
		}
		link := &engine.PaginationLink{URL: nav.Link, Title: nav.Label}
		if isPrev {
			(*pagination).Prev = link
		} else {
			(*pagination).Next = link
		}
		return
	}
	if nav.Label != "" && *pagination != nil {
		if isPrev && (*pagination).Prev != nil {
			(*pagination).Prev.Title = nav.Label
		} else if !isPrev && (*pagination).Next != nil {
			(*pagination).Next.Title = nav.Label
		}
	}
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

func buildLangMap(site *engine.SiteContext) map[string]engine.Language {
	if site == nil {
		return nil
	}
	m := make(map[string]engine.Language, len(site.Languages))
	for _, l := range site.Languages {
		m[l.Code] = l
	}
	return m
}

func langName(m map[string]engine.Language, code string) string {
	if l, ok := m[code]; ok && l.Name != "" {
		return l.Name
	}
	return code
}

func langDir(m map[string]engine.Language, code string) string {
	if l, ok := m[code]; ok && l.Dir != "" {
		return l.Dir
	}
	return "ltr"
}

// mapHomepageSettings converts config.HomepageSettings to engine.HomepageData.
func mapHomepageSettings(s *config.HomepageSettings) *engine.HomepageData {
	d := &engine.HomepageData{
		Template: s.Template,
		Hero: engine.HeroData{
			Eyebrow:    s.Hero.Eyebrow,
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
	if s.Hero.SecondaryCTA != nil {
		d.Hero.SecondaryCTA = &engine.HeroCTAData{
			Label: s.Hero.SecondaryCTA.Label,
			URL:   s.Hero.SecondaryCTA.URL,
		}
	}
	if len(s.Hero.Stats) > 0 {
		d.Hero.Stats = make([]engine.HeroStatData, 0, len(s.Hero.Stats))
		for _, stat := range s.Hero.Stats {
			d.Hero.Stats = append(d.Hero.Stats, engine.HeroStatData{
				Value: stat.Value,
				Label: stat.Label,
			})
		}
	}
	if s.Hero.Code != nil {
		d.Hero.Code = &engine.HeroCodeData{
			Title:    s.Hero.Code.Title,
			Language: s.Hero.Code.Language,
			Body:     s.Hero.Code.Body,
		}
	}
	if s.Hero.Image != nil {
		d.Hero.Image = &engine.HeroImageData{
			Src:   s.Hero.Image.Src,
			Light: s.Hero.Image.Light,
			Dark:  s.Hero.Image.Dark,
			Alt:   s.Hero.Image.Alt,
			HTML:  htmltemplate.HTML(s.Hero.Image.HTML),
		}
	}
	return d
}
