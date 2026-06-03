package build

import (
	"strings"

	"github.com/frostybee/sarde/internal/engine"
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

// resolveRouteAssets applies the URL resolver to asset URLs, breadcrumbs,
// global nav, and pagination URLs in RouteData at render time. CDN URLs
// (with a scheme) are left untouched.
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
	for i := range rd.Translations {
		rd.Translations[i].URL = r.URL(rd.Translations[i].URL, rd.Translations[i].Lang, ver)
	}
	for i := range rd.AllTranslations {
		rd.AllTranslations[i].URL = r.URL(rd.AllTranslations[i].URL, rd.AllTranslations[i].Lang, ver)
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
		if rd.Pagination.Prev != nil {
			rd.Pagination.Prev.URL = r.URL(rd.Pagination.Prev.URL, lang, ver)
		}
		if rd.Pagination.Next != nil {
			rd.Pagination.Next.URL = r.URL(rd.Pagination.Next.URL, lang, ver)
		}
	}
	if rd.GlobalNav != nil {
		for i := range rd.GlobalNav.Items {
			item := &rd.GlobalNav.Items[i]
			if !item.External && !strings.Contains(item.URL, "://") {
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

func resolveNavNodes(r *engine.URLResolver, node *engine.NavNode, lang, version string) {
	if node.URL != "" {
		node.URL = r.URL(node.URL, lang, version)
	}
	for _, child := range node.Children {
		resolveNavNodes(r, child, lang, version)
	}
}

func resolveURLSlice(r *engine.URLResolver, urls []string) []string {
	for i, u := range urls {
		if strings.Contains(u, "://") {
			continue
		}
		urls[i] = r.URL(u, "", "")
	}
	return urls
}
