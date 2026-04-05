package template

import (
	"github.com/coderoo-dev/coderoo/internal/engine"
	"github.com/coderoo-dev/coderoo/internal/navigation"
)

// BuildRouteData constructs the unified RouteData context for a page render.
// Translation fields are left nil — populated in Phase 13.
func BuildRouteData(page *engine.Page, site *engine.SiteContext, theme *engine.ThemeConfig) *engine.RouteData {
	rd := &engine.RouteData{
		Page:   page,
		Site:   site,
		Theme:  theme,
		Layout: engine.LayoutDefault,
		Lang:   "en",
		Dir:    "ltr",
	}

	if site != nil && site.Language != "" {
		rd.Lang = site.Language
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

		// Sidebar and navigation for docs-layout
		if rd.Layout == engine.LayoutDocs {
			rd.SidebarType = "nav"
			rd.HasSidebar = true
			if col.NavTree != nil {
				rd.Sidebar = navigation.MarkActive(col.NavTree, page)
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
		default:
			rd.Template = "_default/single"
		}
		rd.SidebarType = "none"
	}

	// GlobalNav (always populated when site has collections)
	rd.GlobalNav = navigation.BuildGlobalNav(site, col)

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
