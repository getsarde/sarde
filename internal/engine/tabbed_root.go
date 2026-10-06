package engine

import "strings"

// IsTabbedRoot reports whether p is the root index page of a tabbed
// collection (e.g. /courses/ when each course is a tab). The match is by
// lane-free RelPermalink rather than by pointer, so it holds for every
// language of the root and for a page rebuilt in place by the dev server.
// Versioned tabbed collections keep Tabs empty and never match.
func IsTabbedRoot(p *Page) bool {
	if p == nil || p.Kind != KindSection {
		return false
	}
	col := p.Collection
	if col == nil || !col.IsTabbed || len(col.Tabs) == 0 || col.IndexPage == nil {
		return false
	}
	return p.RelPermalink != "" && p.RelPermalink == col.IndexPage.RelPermalink
}

// TabbedRootRedirect returns the URL a tabbed collection's root forwards to,
// or "" when p renders as a page. A root whose _index.md has no body only
// redirects to the first tab; a root with a body is a catalog landing page.
func TabbedRootRedirect(p *Page) string {
	if !IsTabbedRoot(p) || strings.TrimSpace(p.RawContent) != "" {
		return ""
	}
	return p.Collection.Tabs[0].Permalink
}

// IsTabbedLanding reports whether p is a tabbed collection's root rendered
// as a catalog of its tabs, which happens when its _index.md has a body.
func IsTabbedLanding(p *Page) bool {
	return IsTabbedRoot(p) && strings.TrimSpace(p.RawContent) != ""
}
