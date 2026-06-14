package collection

import (
	"slices"
	"strings"

	"github.com/frostybee/sarde/internal/engine"
)

// SortPages sorts pages in-place by the given key and order.
// Supported keys: "date", "order", "title", "slug".
// Order: "asc" or "desc" (defaults to "asc").
// Uses stable sort to preserve filesystem order for ties.
func SortPages(pages []*engine.Page, sortBy, sortOrder string) {
	if len(pages) < 2 {
		return
	}

	desc := strings.ToLower(sortOrder) == "desc"

	slices.SortStableFunc(pages, func(a, b *engine.Page) int {
		cmp := comparePage(a, b, sortBy)
		if desc {
			cmp = -cmp
		}
		return cmp
	})
}

func comparePage(a, b *engine.Page, sortBy string) int {
	switch sortBy {
	case "date":
		return a.Date.Compare(b.Date)
	case "order":
		if a.Sidebar.Order != b.Sidebar.Order {
			return a.Sidebar.Order - b.Sidebar.Order
		}
		// Tiebreaker: alphabetical by title
		return strings.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title))
	case "title":
		return strings.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title))
	case "slug":
		return strings.Compare(a.Slug, b.Slug)
	default:
		return strings.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title))
	}
}
