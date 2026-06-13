package collection

import (
	"github.com/frostybee/sarde/internal/consts"
	"github.com/frostybee/sarde/internal/engine"
)

// ApplyCascade resolves frontmatter inheritance from section _index.md pages.
// Each section's `cascade` map is applied to all descendant pages that don't
// explicitly set those fields. Must be called after BuildSectionTree.
func ApplyCascade(pages []*engine.Page) {
	for _, page := range pages {
		if page.Section == nil {
			continue
		}
		merged := collectCascade(page)
		if len(merged) == 0 {
			continue
		}
		if page.Params == nil {
			page.Params = make(map[string]any)
		}
		for key, value := range merged {
			applyCascadeField(page, key, value)
		}
	}
}

// collectCascade walks the section ancestry from root to the page's direct
// parent, merging cascade maps. Inner sections override outer ones.
func collectCascade(page *engine.Page) map[string]any {
	var ancestors []*engine.Section
	sec := page.Section
	// For section _index.md pages, start from their parent (don't cascade to yourself).
	if page.Kind == engine.KindSection && sec.Parent != nil {
		sec = sec.Parent
	}
	for sec != nil {
		ancestors = append(ancestors, sec)
		sec = sec.Parent
	}
	if len(ancestors) == 0 {
		return nil
	}

	merged := make(map[string]any)
	// Apply outermost first so inner sections override outer.
	for i := len(ancestors) - 1; i >= 0; i-- {
		idx := ancestors[i].IndexPage
		if idx == nil || idx.Params == nil {
			continue
		}
		cascade, ok := idx.Params[consts.CascadeKey].(map[string]any)
		if !ok || len(cascade) == 0 {
			continue
		}
		for k, v := range cascade {
			merged[k] = v
		}
	}
	return merged
}

// applyCascadeField sets a single cascade value on a page, only if the page
// hasn't explicitly set that field itself.
func applyCascadeField(page *engine.Page, key string, value any) {
	switch key {
	case "layout", "template", "type",
		"pagefind", "icon", "edit_url", "show_updated":
		if _, exists := page.Params[key]; !exists {
			page.Params[key] = value
		}

	case "sidebar":
		if subMap, ok := value.(map[string]any); ok {
			if s, ok := subMap["label"].(string); ok && page.Sidebar.Label == "" {
				page.Sidebar.Label = s
			}
			if b, ok := subMap["hidden"].(bool); ok && !page.Sidebar.Hidden {
				page.Sidebar.Hidden = b
			}
			if s, ok := subMap["group"].(string); ok && page.Sidebar.Group == "" {
				page.Sidebar.Group = s
			}
		}

	case "toc":
		if subMap, ok := value.(map[string]any); ok {
			if b, ok := subMap["enabled"].(bool); ok && page.TOC.Enabled == nil {
				page.TOC.Enabled = &b
			}
			if n, ok := subMap["min_level"].(int); ok && page.TOC.MinLevel == 0 {
				page.TOC.MinLevel = n
			}
			if n, ok := subMap["max_level"].(int); ok && page.TOC.MaxLevel == 0 {
				page.TOC.MaxLevel = n
			}
		}

	case "banner":
		if _, exists := page.Params["banner"]; !exists {
			page.Params["banner"] = mapToPageBanner(value)
		}

	case "head":
		if _, exists := page.Params["head"]; !exists {
			page.Params["head"] = value
		}

	case "params":
		if subMap, ok := value.(map[string]any); ok {
			for k, v := range subMap {
				if _, exists := page.Params[k]; !exists {
					page.Params[k] = v
				}
			}
		}

	case "draft", "cascade":
		// draft: filtering already happened before cascade runs
		// cascade: don't cascade the cascade key itself

	default:
		if _, exists := page.Params[key]; !exists {
			page.Params[key] = value
		}
	}
}

func mapToPageBanner(v any) *engine.PageBanner {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	b := &engine.PageBanner{}
	if s, ok := m["content"].(string); ok {
		b.Content = s
	}
	if s, ok := m["variant"].(string); ok {
		b.Variant = s
	}
	if b.Content == "" {
		return nil
	}
	return b
}
