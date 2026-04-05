package navigation

import (
	"sort"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

// BuildGlobalNav auto-generates the top-level site navigation from collections.
// Items are sorted alphabetically by collection name.
// The active item is determined by matching the current page's collection.
func BuildGlobalNav(site *engine.SiteContext, currentCollection *engine.Collection) *engine.GlobalNav {
	if site == nil || len(site.Collections) == 0 {
		return nil
	}

	// Sort collection names for deterministic ordering.
	names := make([]string, 0, len(site.Collections))
	for name := range site.Collections {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]engine.GlobalNavItem, 0, len(names))
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

	return &engine.GlobalNav{Items: items}
}
