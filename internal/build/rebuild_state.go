package build

import (
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
)

func collectionName(p *engine.Page) string {
	if p != nil && p.Collection != nil {
		return p.Collection.Name
	}
	return ""
}

func preserveStablePageState(next, old *engine.Page) {
	next.Collection = old.Collection
	next.Section = old.Section
	next.PrevPage = old.PrevPage
	next.NextPage = old.NextPage
	next.Siblings = old.Siblings
	next.NavNode = old.NavNode
	next.Version = old.Version
	next.VersionRelPath = old.VersionRelPath
	next.VersionPeers = old.VersionPeers
	next.Translations = old.Translations
}

// swapPage returns a copy of pages with next in place of old, matched the
// way replacePagePointer matches it; next is appended when old is absent.
func swapPage(pages []*engine.Page, old, next *engine.Page) []*engine.Page {
	out := make([]*engine.Page, len(pages), len(pages)+1)
	copy(out, pages)
	found := false
	for i, p := range out {
		if p == old || (p != nil && p.FilePath == old.FilePath && !p.IsFallback) {
			out[i] = next
			found = true
		}
	}
	if !found {
		out = append(out, next)
	}
	return out
}

func replacePagePointer(pages *[]*engine.Page, old, next *engine.Page) {
	if pages == nil {
		return
	}
	for i, p := range *pages {
		if p == old || (p != nil && p.FilePath == old.FilePath && !p.IsFallback) {
			(*pages)[i] = next
		}
	}
}

// patchTaxonomyPagePointers swaps old for next inside every term's Pages
// slice across all taxonomies in the given set. Body-only edits cannot change
// term membership (frontmatter is proven unchanged by the digest and title
// gate), but every re-parsed file is a new *engine.Page object, so reused
// taxonomy structures must be re-pointed or term pages would render the old
// object's stale summary and content.
func patchTaxonomyPagePointers(taxonomies map[string]*engine.Taxonomy, old, next *engine.Page) {
	for _, tax := range taxonomies {
		for _, term := range tax.Terms {
			replacePagePointer(&term.Pages, old, next)
		}
	}
}

func copyRenderedContentToFallbacks(pages []*engine.Page, changed map[string]*engine.Page, idx *content.PageIndex) {
	for _, p := range pages {
		if !p.IsFallback || p.FilePath == "" {
			continue
		}
		src := changed[p.FilePath]
		if src == nil {
			continue
		}
		p.Content = src.Content
		p.Headings = src.Headings
		p.HasCodeBlocks = src.HasCodeBlocks
		p.HasImages = src.HasImages
		setPageIndexHeadings(idx, p)
	}
}
