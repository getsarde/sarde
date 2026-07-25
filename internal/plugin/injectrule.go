package plugin

import "github.com/getsarde/sarde/internal/engine"

// MatchesInjectRule evaluates a declarative inject-condition rule against a
// page and its route data. Shared by the clientplugins manifest loader and
// the external plugin loader. Unknown rules never match.
func MatchesInjectRule(rule string, page *engine.Page, rd *engine.RouteData) bool {
	switch rule {
	case "always":
		return true
	case "has_sidebar":
		return engine.LayoutHasSidebar(rd.Layout)
	case "has_toc":
		return engine.LayoutHasTOC(rd.Layout) && len(page.Headings) > 0
	case "has_headings":
		return len(page.Headings) > 0
	case "has_code_blocks":
		return page.HasCodeBlocks
	case "has_images":
		return page.HasImages
	case "has_prev_next":
		return page.PrevPage != nil || page.NextPage != nil
	case "is_content_page":
		return page.Kind == engine.KindPage || page.Kind == engine.KindBundle
	case "has_updated":
		return !page.Updated.IsZero() && page.ShowUpdated()
	}
	return false
}
