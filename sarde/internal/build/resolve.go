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
		page.Permalink = r.URL(page.RelPermalink, "", "")
	}
}

// resolveRouteAssets applies the URL resolver to asset URLs, breadcrumbs,
// global nav, and pagination URLs in RouteData at render time. CDN URLs
// (with a scheme) are left untouched.
func resolveRouteAssets(r *engine.URLResolver, rd *engine.RouteData) {
	if r == nil || r.BasePath == "/" {
		return
	}
	rd.Styles = resolveURLSlice(r, rd.Styles)
	rd.Scripts = resolveURLSlice(r, rd.Scripts)
	rd.ModuleScripts = resolveURLSlice(r, rd.ModuleScripts)

	// Sidebar nav tree (rd.Sidebar is a deep clone from MarkActive, safe to mutate).
	if rd.Sidebar != nil && rd.Sidebar.Root != nil {
		resolveNavNodes(r, rd.Sidebar.Root)
	}

	for i := range rd.Translations {
		rd.Translations[i].URL = r.URL(rd.Translations[i].URL, "", "")
	}

	for i := range rd.Breadcrumbs {
		rd.Breadcrumbs[i].URL = r.URL(rd.Breadcrumbs[i].URL, "", "")
	}
	if rd.Pagination != nil {
		if rd.Pagination.Prev != nil {
			rd.Pagination.Prev.URL = r.URL(rd.Pagination.Prev.URL, "", "")
		}
		if rd.Pagination.Next != nil {
			rd.Pagination.Next.URL = r.URL(rd.Pagination.Next.URL, "", "")
		}
	}
	if rd.GlobalNav != nil {
		for i := range rd.GlobalNav.Items {
			item := &rd.GlobalNav.Items[i]
			if !item.External && !strings.Contains(item.URL, "://") {
				item.URL = r.URL(item.URL, "", "")
			}
		}
	}
	if rd.Paginator != nil {
		for i := range rd.Paginator.Pages {
			rd.Paginator.Pages[i].URL = r.URL(rd.Paginator.Pages[i].URL, "", "")
		}
		rd.Paginator.PrevURL = r.URL(rd.Paginator.PrevURL, "", "")
		rd.Paginator.NextURL = r.URL(rd.Paginator.NextURL, "", "")
		rd.Paginator.BaseURL = r.URL(rd.Paginator.BaseURL, "", "")
		rd.Paginator.FirstURL = r.URL(rd.Paginator.FirstURL, "", "")
		rd.Paginator.LastURL = r.URL(rd.Paginator.LastURL, "", "")
	}
}

func resolveNavNodes(r *engine.URLResolver, node *engine.NavNode) {
	if node.URL != "" {
		node.URL = r.URL(node.URL, "", "")
	}
	for _, child := range node.Children {
		resolveNavNodes(r, child)
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
