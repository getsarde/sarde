package build

import (
	"strings"

	"github.com/getsarde/sarde/internal/engine"
)

// resolvePermalinks resolves Page.Permalink through the URL resolver.
// Only pages are resolved — sections, nav trees, tabs, and taxonomies
// stay prefix-free. Their URLs are resolved at render time through the
// relURL template function (the single chokepoint).
func resolvePermalinks(r *engine.URLResolver, pages []*engine.Page) {
	for _, page := range pages {
		page.Permalink = r.URL(page.RelPermalink, page.Lang, resolvePageVersion(page))
	}
}

// resolvePageVersion delegates to engine.ResolvePageVersion.
func resolvePageVersion(page *engine.Page) string {
	return engine.ResolvePageVersion(page)
}

// assetURLResolver returns a function that prefixes a root-relative asset
// URL with the base path only (no language or version, assets are shared).
// It returns nil when no resolver is configured, which leaves URLs unchanged.
func (b *SiteBuilder) assetURLResolver() func(string) string {
	if b.urlResolver == nil {
		return nil
	}
	return func(u string) string { return b.urlResolver.URL(u, "", "") }
}

// resolveRouteAssets applies the URL resolver to asset URLs, breadcrumbs,
// global nav, and pagination URLs in RouteData at render time. A URL that
// names another location (a scheme such as https: or mailto:, or //host) is
// left untouched; see engine.IsAbsoluteURL.
func resolveRouteAssets(r *engine.URLResolver, rd *engine.RouteData) {
	if r == nil {
		return
	}

	// Assets: basePath only, no lang/version (assets are shared).
	if r.BasePath != "/" {
		rd.Styles = resolveURLSlice(r, rd.Styles)
		rd.Scripts = resolveURLSlice(r, rd.Scripts)
		rd.ModuleScripts = resolveURLSlice(r, rd.ModuleScripts)
	}

	lang := ""
	ver := ""
	if rd.Page != nil {
		lang = rd.Page.Lang
		ver = resolvePageVersion(rd.Page)
	}

	// Sidebar nav tree (rd.Sidebar is a deep clone from MarkActive, safe to mutate).
	if rd.Sidebar != nil && rd.Sidebar.Root != nil {
		resolveNavNodes(r, rd.Sidebar.Root, lang, ver)
	}

	// DocsTabs: shallow-copy to avoid mutating shared col.Tabs, then resolve permalinks.
	if len(rd.DocsTabs) > 0 && lang != "" {
		copied := make([]*engine.DocsTab, len(rd.DocsTabs))
		for i, tab := range rd.DocsTabs {
			cp := *tab
			cp.Permalink = r.URL(tab.Permalink, lang, ver)
			copied[i] = &cp
		}
		rd.DocsTabs = copied
	}

	// Translations: each link carries its own lang; version stays the same.
	// AbsURL is resolved here, in the link's own language: the absURL template
	// function adds the page's language, which is wrong for the other links.
	origin := strings.TrimRight(r.BaseURL, "/")
	for i := range rd.Translations {
		link := &rd.Translations[i]
		link.URL = r.URL(link.URL, link.Lang, ver)
		link.AbsURL = origin + link.URL
	}
	for i := range rd.AllTranslations {
		link := &rd.AllTranslations[i]
		link.URL = r.URL(link.URL, link.Lang, ver)
		link.AbsURL = origin + link.URL
	}

	// Version links: each link carries its own version ID.
	for i := range rd.Versions {
		linkVer := rd.Versions[i].ID
		if rd.Versions[i].IsLatest {
			linkVer = ""
		}
		rd.Versions[i].URL = r.URL(rd.Versions[i].URL, lang, linkVer)
	}

	for i := range rd.Breadcrumbs {
		// Skip empty URLs (inferred/phantom sections render as plain text);
		// resolving "" would expand to the base path. Mirrors resolveNavNodes.
		if rd.Breadcrumbs[i].URL != "" {
			rd.Breadcrumbs[i].URL = r.URL(rd.Breadcrumbs[i].URL, lang, ver)
		}
	}
	if rd.Pagination != nil {
		// A frontmatter prev or next link can name another site.
		if p := rd.Pagination.Prev; p != nil && !engine.IsAbsoluteURL(p.URL) {
			p.URL = r.URL(p.URL, lang, ver)
		}
		if n := rd.Pagination.Next; n != nil && !engine.IsAbsoluteURL(n.URL) {
			n.URL = r.URL(n.URL, lang, ver)
		}
	}
	if rd.GlobalNav != nil {
		for i := range rd.GlobalNav.Items {
			item := &rd.GlobalNav.Items[i]
			if !item.External && !engine.IsAbsoluteURL(item.URL) {
				item.URL = r.URL(item.URL, lang, "")
			}
		}
	}
	if rd.Paginator != nil {
		for i := range rd.Paginator.Pages {
			rd.Paginator.Pages[i].URL = r.URL(rd.Paginator.Pages[i].URL, lang, ver)
		}
		rd.Paginator.PrevURL = r.URL(rd.Paginator.PrevURL, lang, ver)
		rd.Paginator.NextURL = r.URL(rd.Paginator.NextURL, lang, ver)
		rd.Paginator.BaseURL = r.URL(rd.Paginator.BaseURL, lang, ver)
		rd.Paginator.FirstURL = r.URL(rd.Paginator.FirstURL, lang, ver)
		rd.Paginator.LastURL = r.URL(rd.Paginator.LastURL, lang, ver)
	}
}

// resolveNavNodes resolves the site paths in a sidebar tree. A nav.yaml url
// can name another site, which keeps its URL.
func resolveNavNodes(r *engine.URLResolver, node *engine.NavNode, lang, version string) {
	if node.URL != "" && !engine.IsAbsoluteURL(node.URL) {
		node.URL = r.URL(node.URL, lang, version)
	}
	for _, child := range node.Children {
		resolveNavNodes(r, child, lang, version)
	}
}

func resolveURLSlice(r *engine.URLResolver, urls []string) []string {
	for i, u := range urls {
		if engine.IsAbsoluteURL(u) {
			continue
		}
		urls[i] = r.URL(u, "", "")
	}
	return urls
}
