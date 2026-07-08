package build

import (
	"cmp"
	"slices"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/links"
)

func populatePageIndexHeadings(idx *content.PageIndex, pages []*engine.Page) {
	for _, page := range pages {
		setPageIndexHeadings(idx, page)
	}
}

func setPageIndexHeadings(idx *content.PageIndex, page *engine.Page) {
	if idx == nil || page == nil || len(page.Headings) == 0 {
		return
	}
	if existing := idx.HeadingsFor(page.Permalink); existing != nil {
		return
	}
	ids := make([]string, len(page.Headings))
	for i, h := range page.Headings {
		ids[i] = h.ID
	}
	idx.SetHeadings(page.Permalink, ids)
}

func updateValidationEntry(data map[string]engine.ValidationEntry, page *engine.Page, links []engine.CollectedLink) {
	if len(links) == 0 {
		delete(data, page.Permalink)
		return
	}
	data[page.Permalink] = engine.ValidationEntry{Links: links, FilePath: page.FilePath, Lang: page.Lang}
}

// toCachedAnchors strips a page's pending anchor checks down to the
// content-derived fields that are safe to persist in the page cache.
func toCachedAnchors(pending []links.PendingAnchorCheck) []CachedAnchorCheck {
	if len(pending) == 0 {
		return nil
	}
	out := make([]CachedAnchorCheck, len(pending))
	for i, p := range pending {
		out[i] = CachedAnchorCheck{
			TargetPermalink: p.TargetPermalink,
			Fragment:        p.Fragment,
			RawHref:         p.RawHref,
			Kind:            p.Kind,
			Resolved:        p.Resolved,
		}
	}
	return out
}

// replayPendingAnchors reconstructs pending anchor checks from a cache hit,
// re-deriving the page-specific fields (source file, from/target page,
// dimension) from the live page and index instead of trusting stale pointers.
func replayPendingAnchors(cached []CachedAnchorCheck, page *engine.Page, idx *content.PageIndex) []links.PendingAnchorCheck {
	if len(cached) == 0 || idx == nil {
		return nil
	}
	collName := ""
	if page.Collection != nil {
		collName = page.Collection.Name
	}
	dim := links.DimKey{Collection: collName, Lang: page.Lang, Version: page.Version}
	out := make([]links.PendingAnchorCheck, len(cached))
	for i, c := range cached {
		out[i] = links.PendingAnchorCheck{
			SourceFile:      page.FilePath,
			TargetPermalink: c.TargetPermalink,
			Fragment:        c.Fragment,
			RawHref:         c.RawHref,
			FromPage:        page,
			TargetPage:      idx.LookupByPermalink(c.TargetPermalink),
			Dim:             dim,
			Kind:            c.Kind,
			Resolved:        c.Resolved,
		}
	}
	return out
}

func countMarkdownPages(pages []*engine.Page) int {
	count := 0
	for _, page := range pages {
		if page.RawContent != "" {
			count++
		}
	}
	return count
}

func sortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
