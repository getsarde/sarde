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
		page.Permalink = r.URL(page.RelPermalink, page.Lang, "")
	}
}

// resolveRouteAssets applies the URL resolver to asset URLs, breadcrumbs,
// global nav, and pagination URLs in RouteData at render time. CDN URLs
// (with a scheme) are left untouched.
func resolveRouteAssets(r *engine.URLResolver, rd *engine.RouteData) {
	if r == nil {
		return
	}

	// Assets: basePath only, no lang (assets are shared across languages).
	if r.BasePath != "/" {
		rd.Styles = resolveURLSlice(r, rd.Styles)
		rd.Scripts = resolveURLSlice(r, rd.Scripts)
		rd.ModuleScripts = resolveURLSlice(r, rd.ModuleScripts)
	}

	lang := ""
	if rd.Page != nil {
		lang = rd.Page.Lang
	}

	// Sidebar nav tree (rd.Sidebar is a deep clone from MarkActive, safe to mutate).
	if rd.Sidebar != nil && rd.Sidebar.Root != nil {
		resolveNavNodes(r, rd.Sidebar.Root, lang)
	}

	for i := range rd.Translations {
		rd.Translations[i].URL = r.URL(rd.Translations[i].URL, rd.Translations[i].Lang, "")
	}
	for i := range rd.AllTranslations {
		rd.AllTranslations[i].URL = r.URL(rd.AllTranslations[i].URL, rd.AllTranslations[i].Lang, "")
	}

	for i := range rd.Breadcrumbs {
		rd.Breadcrumbs[i].URL = r.URL(rd.Breadcrumbs[i].URL, lang, "")
	}
	if rd.Pagination != nil {
		if rd.Pagination.Prev != nil {
			rd.Pagination.Prev.URL = r.URL(rd.Pagination.Prev.URL, lang, "")
		}
		if rd.Pagination.Next != nil {
			rd.Pagination.Next.URL = r.URL(rd.Pagination.Next.URL, lang, "")
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
			rd.Paginator.Pages[i].URL = r.URL(rd.Paginator.Pages[i].URL, lang, "")
		}
		rd.Paginator.PrevURL = r.URL(rd.Paginator.PrevURL, lang, "")
		rd.Paginator.NextURL = r.URL(rd.Paginator.NextURL, lang, "")
		rd.Paginator.BaseURL = r.URL(rd.Paginator.BaseURL, lang, "")
		rd.Paginator.FirstURL = r.URL(rd.Paginator.FirstURL, lang, "")
		rd.Paginator.LastURL = r.URL(rd.Paginator.LastURL, lang, "")
	}
}

func resolveNavNodes(r *engine.URLResolver, node *engine.NavNode, lang string) {
	if node.URL != "" {
		node.URL = r.URL(node.URL, lang, "")
	}
	for _, child := range node.Children {
		resolveNavNodes(r, child, lang)
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
