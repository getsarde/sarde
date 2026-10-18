package build

import (
	"slices"

	"github.com/getsarde/sarde/internal/engine"
)

// addLegacyDateAliases adds each date-prefixed page's Sarde 1.5 URL
// (Page.LegacyRelPermalink) to its Aliases, so the old URL redirects to the
// current one. A real page, an author alias or a global redirect at that
// path wins, and two pages that shared a 1.5 URL give it to the first. Runs
// after resolvePermalinks; a fallback copy is skipped, as its source owns the
// alias.
func addLegacyDateAliases(r *engine.URLResolver, pages []*engine.Page, redirects map[string]string) {
	if r == nil {
		return
	}
	taken := make(map[string]struct{}, len(pages)+len(redirects))
	for _, p := range pages {
		if p.RelPermalink != "" {
			taken[PageOutputPath(r.OutputRelPath(p.RelPermalink, p.Lang, resolvePageVersion(p)))] = struct{}{}
		}
		for _, alias := range p.Aliases {
			taken[PageOutputPath(alias)] = struct{}{}
		}
	}
	for from := range redirects {
		taken[PageOutputPath(from)] = struct{}{}
	}
	for _, p := range pages {
		if p.IsFallback || p.LegacyRelPermalink == "" {
			continue
		}
		alias := legacyDateAlias(r, p)
		key := PageOutputPath(alias)
		if _, ok := taken[key]; ok {
			continue
		}
		taken[key] = struct{}{}
		// Clip: a fallback copy shares its source's backing array.
		p.Aliases = append(slices.Clip(p.Aliases), alias)
	}
}

// carryLegacyDateAlias keeps a page's 1.5 URL redirect across a dev
// incremental rebuild: next gets it again when old had it, so the full
// build's collision decision stands.
func carryLegacyDateAlias(r *engine.URLResolver, next, old *engine.Page) {
	if r == nil || next.LegacyRelPermalink == "" {
		return
	}
	alias := legacyDateAlias(r, next)
	if slices.Contains(old.Aliases, alias) && !slices.Contains(next.Aliases, alias) {
		next.Aliases = append(slices.Clip(next.Aliases), alias)
	}
}

// legacyDateAlias is the output path of p's 1.5 URL, language- and
// version-prefixed like an author alias.
func legacyDateAlias(r *engine.URLResolver, p *engine.Page) string {
	return r.OutputRelPath(p.LegacyRelPermalink, p.Lang, resolvePageVersion(p))
}
