package build

import (
	"cmp"
	"slices"
	"strings"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/linkrender"
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
		targetRel := ""
		if p.TargetPage != nil {
			targetRel = p.TargetPage.RelPermalink
		}
		out[i] = CachedAnchorCheck{
			TargetPermalink:    p.TargetPermalink,
			TargetRelPermalink: targetRel,
			Fragment:           p.Fragment,
			RawHref:            p.RawHref,
			Kind:               p.Kind,
			Resolved:           p.Resolved,
		}
	}
	return out
}

// toCachedRefs strips a page's recorded link refs down to the content-derived
// fields that are safe to persist in the page cache. Refs and pending anchors
// are mutually exclusive per href (recordLinkRef vs appendPendingAnchor), so
// the two cached lists never overlap.
func toCachedRefs(recorded []links.LinkRef) []CachedLinkRef {
	if len(recorded) == 0 {
		return nil
	}
	out := make([]CachedLinkRef, len(recorded))
	for i, r := range recorded {
		var targetPermalink, targetRel string
		if r.TargetPage != nil {
			targetPermalink = r.TargetPage.Permalink
			targetRel = r.TargetPage.RelPermalink
		}
		out[i] = CachedLinkRef{
			RawDest:            r.RawDest,
			Kind:               r.Kind,
			Resolved:           r.Resolved,
			TargetPermalink:    targetPermalink,
			TargetRelPermalink: targetRel,
			Fragment:           r.Fragment,
			Status:             r.Status,
			Line:               r.Line,
			Col:                r.Col,
		}
	}
	return out
}

// replayCachedRefs reconstructs recorded link refs from a cache hit,
// re-deriving the page-specific fields (source file, from/target page,
// dimension) from the live page and index. Mirrors replayPendingAnchors.
func replayCachedRefs(cached []CachedLinkRef, page *engine.Page, idx *content.PageIndex) []links.LinkRef {
	if len(cached) == 0 {
		return nil
	}
	collName := ""
	if page.Collection != nil {
		collName = page.Collection.Name
	}
	dim := links.DimKey{Collection: collName, Lang: page.Lang, Version: page.Version}
	out := make([]links.LinkRef, len(cached))
	for i, c := range cached {
		var targetPage *engine.Page
		if idx != nil && c.TargetPermalink != "" {
			targetPage = idx.LookupByPermalink(c.TargetPermalink)
		}
		out[i] = links.LinkRef{
			FromPage:   page,
			FromFile:   page.FilePath,
			RawDest:    c.RawDest,
			Dim:        dim,
			Kind:       c.Kind,
			Resolved:   c.Resolved,
			TargetPage: targetPage,
			Fragment:   c.Fragment,
			Status:     c.Status,
			Line:       c.Line,
			Col:        c.Col,
		}
	}
	return out
}

// entryStale verifies a page-cache entry's refs and pending anchors against
// live state before replay. Any single divergence marks the whole entry stale
// with no partial patching: the caller falls through to a full render, which
// re-Puts a fresh entry under the same hash (self-healing).
func entryStale(entry *CacheEntry, page *engine.Page, idx *content.PageIndex, resolver *engine.URLResolver, collections map[string]*engine.Collection) bool {
	if idx == nil {
		return false
	}
	for _, c := range entry.PendingAnchors {
		if targetLinkStale(idx, c.TargetPermalink, c.TargetRelPermalink) {
			return true
		}
	}
	for _, ref := range entry.Refs {
		if refStale(ref, page, idx, resolver, collections) {
			return true
		}
	}
	return false
}

// targetLinkStale reports whether a previously-resolved target was deleted
// (permalink gone from the index) or replaced by a different page occupying
// the same permalink with a different URL. The RelPermalink comparison has no
// observable effect on report output today (TargetPage is never read by
// consumers); it exists to force a self-healing re-render of the source
// page's baked URL in the delete-and-recreate edge case.
func targetLinkStale(idx *content.PageIndex, targetPermalink, targetRelPermalink string) bool {
	target := idx.LookupByPermalink(targetPermalink)
	return target == nil || target.RelPermalink != targetRelPermalink
}

func refStale(ref CachedLinkRef, page *engine.Page, idx *content.PageIndex, resolver *engine.URLResolver, collections map[string]*engine.Collection) bool {
	switch ref.Status {
	case links.StatusOK:
		return targetLinkStale(idx, ref.TargetPermalink, ref.TargetRelPermalink)

	case links.StatusBrokenTarget:
		if resolver == nil {
			return false
		}
		dest := linkrender.ClassifyDest(ref.RawDest)
		result := linkrender.ResolveInternalLink(dest, page, linkrender.ResolveContext{
			PageIndex:   idx,
			URLResolver: resolver.URL,
			Collections: collections,
		})
		// Now resolves, but the cached HTML still bakes href="#".
		return result.Found

	case links.StatusAmbiguous:
		// Never resolves by design; the cached href="#" stays right.
		return false

	case links.StatusUnverified:
		// Unverified refs arise from resolveSiteAbsolute, reached either
		// directly or as ResolveHref's fallthrough after a failed content-root
		// resolution. Mirror that full chain: a target appearing in EITHER
		// branch means a fresh render would resolve the link, so the cached
		// HTML (un-prefixed passthrough href) is stale.
		if resolver != nil {
			dest := linkrender.ClassifyDest(ref.RawDest)
			if dest.Kind == linkrender.LinkContentRoot {
				result := linkrender.ResolveInternalLink(dest, page, linkrender.ResolveContext{
					PageIndex:   idx,
					URLResolver: resolver.URL,
					Collections: collections,
				})
				if result.Found {
					return true
				}
			}
		}
		// Mirrors resolveSiteAbsolute's fragment/query stripping
		// (linkrender/renderer.go); keep in sync if that parsing changes.
		pathPart := ref.RawDest
		if i := strings.IndexByte(pathPart, '#'); i >= 0 {
			pathPart = pathPart[:i]
		}
		if i := strings.IndexByte(pathPart, '?'); i >= 0 {
			pathPart = pathPart[:i]
		}
		relP := content.NormalizePermalink(pathPart)
		return linkrender.LookupInLaneWithDefaultFallback(idx, relP, page, resolver) != nil

	default:
		// StatusExternal (and any future unchecked status): never stale.
		return false
	}
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
