package navigation

import (
	"github.com/coderoo-dev/coderoo/internal/engine"
)

// BuildBreadcrumbs constructs a breadcrumb trail for a docs-layout page.
// The trail starts with the collection root and ends with the current page.
func BuildBreadcrumbs(page *engine.Page, collection *engine.Collection) []engine.BreadcrumbItem {
	if page == nil || collection == nil {
		return nil
	}

	// Collect the section chain from current page up to root.
	var sections []*engine.Section
	sec := page.Section
	for sec != nil {
		sections = append(sections, sec)
		sec = sec.Parent
	}

	// Build breadcrumbs: collection root → sections (reversed) → current page.
	crumbs := make([]engine.BreadcrumbItem, 0, len(sections)+2)

	// Collection root.
	crumbs = append(crumbs, engine.BreadcrumbItem{
		Label: collection.Title,
		URL:   "/" + collection.Name + "/",
	})

	// Sections in root-to-leaf order.
	for i := len(sections) - 1; i >= 0; i-- {
		sec := sections[i]
		// Skip transparent sections — they shouldn't appear in breadcrumbs.
		if sec.Transparent {
			continue
		}
		crumbs = append(crumbs, engine.BreadcrumbItem{
			Label: sec.Title,
			URL:   sec.Permalink,
		})
	}

	// Current page (if it's not already the collection root or a section index).
	if page.Kind != engine.KindSection {
		crumbs = append(crumbs, engine.BreadcrumbItem{
			Label:   page.Title,
			URL:     page.RelPermalink,
			Current: true,
		})
	} else {
		// Mark the last crumb as current for section index pages.
		if len(crumbs) > 0 {
			crumbs[len(crumbs)-1].Current = true
		}
	}

	return crumbs
}

// BuildBreadcrumbsTabbed constructs breadcrumbs for a tabbed docs page.
// Adds the tab as a level between the collection root and section chain.
func BuildBreadcrumbsTabbed(page *engine.Page, col *engine.Collection, tab *engine.DocsTab) []engine.BreadcrumbItem {
	if page == nil || col == nil || tab == nil {
		return nil
	}

	crumbs := []engine.BreadcrumbItem{
		{Label: col.Title, URL: "/" + col.Name + "/"},
		{Label: tab.Title, URL: tab.Permalink},
	}

	// Section chain — skip the tab's root section (already represented by the tab crumb)
	var sections []*engine.Section
	sec := page.Section
	for sec != nil && sec != tab.Section {
		sections = append(sections, sec)
		sec = sec.Parent
	}
	for i := len(sections) - 1; i >= 0; i-- {
		s := sections[i]
		if s.Transparent {
			continue
		}
		crumbs = append(crumbs, engine.BreadcrumbItem{Label: s.Title, URL: s.Permalink})
	}

	if page.Kind != engine.KindSection {
		crumbs = append(crumbs, engine.BreadcrumbItem{
			Label:   page.Title,
			URL:     page.RelPermalink,
			Current: true,
		})
	} else if len(crumbs) > 0 {
		crumbs[len(crumbs)-1].Current = true
	}

	return crumbs
}
