package navigation

import (
	"github.com/frostybee/sarde/internal/engine"
)

// WirePrevNextFromTree rewires Page.PrevPage/NextPage based on the nav tree's
// flattened DFS order. This replaces simple sort-order prev/next for
// docs-layout collections, enabling cross-section navigation.
func WirePrevNextFromTree(tree *engine.NavTree) {
	if tree == nil || len(tree.Flat) == 0 {
		return
	}

	// Collect pages from flat list (some nodes may lack a Page reference).
	var pages []*engine.Page
	for _, node := range tree.Flat {
		if node.Page != nil {
			pages = append(pages, node.Page)
		}
	}

	// Wire sequential prev/next.
	for i, page := range pages {
		page.PrevPage = nil
		page.NextPage = nil
		if i > 0 {
			page.PrevPage = pages[i-1]
		}
		if i < len(pages)-1 {
			page.NextPage = pages[i+1]
		}
	}

	// Apply manual overrides from frontmatter.
	lookup := make(map[string]*engine.Page, len(pages))
	for _, p := range pages {
		lookup[p.Slug] = p
	}

	for _, page := range pages {
		if page.Params == nil {
			continue
		}
		if nav := asNavOverride(page.Params["prev"]); nav != nil {
			if nav.Disabled {
				page.PrevPage = nil
			} else if nav.Slug != "" {
				if target := lookup[nav.Slug]; target != nil {
					page.PrevPage = target
				}
			}
		}
		if nav := asNavOverride(page.Params["next"]); nav != nil {
			if nav.Disabled {
				page.NextPage = nil
			} else if nav.Slug != "" {
				if target := lookup[nav.Slug]; target != nil {
					page.NextPage = target
				}
			}
		}
	}
}

func asNavOverride(v any) *engine.NavOverride {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case *engine.NavOverride:
		return val
	case string:
		if val != "" {
			return &engine.NavOverride{Slug: val}
		}
	}
	return nil
}
