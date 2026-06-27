package navigation

import (
	"sort"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

// BuildGlobalNav builds the top-level site navigation from collections and
// optional header links from the site config. Collection items are sorted
// alphabetically; header links are appended after collection items.
func BuildGlobalNav(site *engine.SiteContext, currentCollection *engine.Collection, headerLinks []config.NavLink) *engine.GlobalNav {
	if site == nil {
		return nil
	}
	if len(site.Collections) == 0 && len(headerLinks) == 0 {
		return nil
	}

	var items []engine.GlobalNavItem

	// Collection-generated items.
	names := make([]string, 0, len(site.Collections))
	for name := range site.Collections {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		col := site.Collections[name]
		item := engine.GlobalNavItem{
			Label:      col.Title,
			URL:        "/" + col.Name + "/",
			Collection: col.Name,
		}
		if currentCollection != nil && col.Name == currentCollection.Name {
			item.IsActive = true
		}
		items = append(items, item)
	}

	// Config header links appended after collections.
	for _, link := range headerLinks {
		items = append(items, engine.GlobalNavItem{
			Label:    link.Label,
			URL:      link.URL,
			External: link.External,
		})
	}

	return &engine.GlobalNav{Items: items}
}
