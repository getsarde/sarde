package linkrender

import (
	"path"
	"strings"

	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/engine"
)

// ResolveResult holds the outcome of resolving an internal link.
type ResolveResult struct {
	URL             string // final resolved URL (empty if not found)
	TargetPermalink string // target page's Permalink (for anchor validation)
	Found           bool
}

// ResolveContext carries the per-build dependencies the resolver needs.
// Collections is nil-safe: a nil map falls back to same-lane resolution (the
// current page's version), preserving behavior for callers that don't supply it.
type ResolveContext struct {
	PageIndex   PageLookup
	URLResolver URLResolverFunc
	Collections map[string]*engine.Collection
}

// ResolveInternalLink resolves a parsed destination in the current page's lane,
// returning the final URL via the URL resolver. External, anchor-only, and
// site-root links short-circuit without an index lookup.
func ResolveInternalLink(
	dest ParsedDest,
	currentPage *engine.Page,
	ctx ResolveContext,
) ResolveResult {
	switch dest.Kind {
	case LinkExternal:
		return ResolveResult{URL: dest.Raw, Found: true}

	case LinkAnchorOnly:
		version := engine.ResolvePageVersion(currentPage)
		url := ctx.URLResolver(currentPage.RelPermalink, currentPage.Lang, version)
		return ResolveResult{
			URL:             withSuffix(url, dest.Fragment, dest.Query),
			TargetPermalink: currentPage.Permalink,
			Found:           true,
		}

	case LinkSiteRoot:
		// Escape hatch: resolve at the site root with no lane segments. Never
		// validated against the index — the author opts out of resolution.
		url := ctx.URLResolver(dest.Path, "", "")
		return ResolveResult{URL: withSuffix(url, dest.Fragment, dest.Query), Found: true}

	case LinkAmbiguous:
		return ResolveResult{Found: false}

	case LinkRelative:
		return resolveRelative(dest, currentPage, ctx)

	case LinkContentRoot:
		return resolveContentRoot(dest, currentPage, ctx)
	}

	return ResolveResult{Found: false}
}

func resolveRelative(
	dest ParsedDest,
	currentPage *engine.Page,
	ctx ResolveContext,
) ResolveResult {
	baseDir := path.Dir(effectiveRelPath(currentPage))
	logical := path.Clean(path.Join(baseDir, dest.Path))

	return lookupAndResolve(logical, dest, currentPage, ctx)
}

// effectiveRelPath returns the version-free relative path for resolution.
// For versioned pages, we need the collection name + VersionRelPath (since
// VersionRelPath excludes both version AND collection name segments).
// For unversioned pages, LangRelPath is used directly (it includes collection name).
func effectiveRelPath(page *engine.Page) string {
	if page.VersionRelPath != "" && page.Collection != nil {
		return page.Collection.Name + "/" + page.VersionRelPath
	}
	return page.LangRelPath
}

func resolveContentRoot(
	dest ParsedDest,
	currentPage *engine.Page,
	ctx ResolveContext,
) ResolveResult {
	collName := ""
	if currentPage.Collection != nil {
		collName = currentPage.Collection.Name
	}

	// Strip leading / from dest.Path, prepend collection name.
	destPath := dest.Path
	if len(destPath) > 0 && destPath[0] == '/' {
		destPath = destPath[1:]
	}

	var logical string
	if collName != "" {
		logical = collName + "/" + destPath
	} else {
		logical = destPath
	}

	return lookupAndResolve(logical, dest, currentPage, ctx)
}

func lookupAndResolve(
	logical string,
	dest ParsedDest,
	currentPage *engine.Page,
	ctx ResolveContext,
) ResolveResult {
	// Cross-dimension: when the target collection (first path segment of the
	// logical path) is unversioned, drop the version coordinate so a versioned
	// page (e.g. docs/v1) can link into an unversioned collection (e.g. blog).
	// Within the same versioned collection the source version is preserved.
	lookupVersion := currentPage.Version
	if ctx.Collections != nil {
		if isUnversioned(ctx.Collections[firstPathSegment(logical)]) {
			lookupVersion = ""
		}
	}

	// Compute the expected RelPermalink from the logical filesystem path.
	relPermalink := content.ComputePermalinkFromRelPath(logical + ".md")

	target := ctx.PageIndex.LookupInLane(relPermalink, currentPage.Lang, lookupVersion)

	// Fallback: try as a directory index (e.g. ./guides/ → guides/_index.md).
	if target == nil {
		indexLogical := logical + "/_index"
		indexRelPermalink := content.ComputePermalinkFromRelPath(indexLogical + ".md")
		target = ctx.PageIndex.LookupInLane(indexRelPermalink, currentPage.Lang, lookupVersion)
	}

	// Fallback: try as index.md (leaf bundle).
	if target == nil {
		indexLogical := logical + "/index"
		indexRelPermalink := content.ComputePermalinkFromRelPath(indexLogical + ".md")
		target = ctx.PageIndex.LookupInLane(indexRelPermalink, currentPage.Lang, lookupVersion)
	}

	if target == nil {
		return ResolveResult{Found: false}
	}

	// The URL version segment comes from the TARGET, not the source: "" for an
	// unversioned or latest-version target, the version ID otherwise.
	url := ctx.URLResolver(target.RelPermalink, currentPage.Lang, engine.ResolvePageVersion(target))

	return ResolveResult{
		URL:             withSuffix(url, dest.Fragment, dest.Query),
		TargetPermalink: target.Permalink,
		Found:           true,
	}
}

// firstPathSegment returns the first slash-separated segment of a logical path
// (the target collection name), ignoring a leading slash.
func firstPathSegment(p string) string {
	p = strings.TrimPrefix(p, "/")
	if i := strings.IndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return p
}

// isUnversioned reports whether a collection has no active version axis. A nil
// collection (target not found / not a collection) is treated as unversioned.
func isUnversioned(c *engine.Collection) bool {
	return c == nil || c.Config == nil || c.Config.Versioning == nil || !c.Config.Versioning.Enabled
}

// PageLookup is the interface the resolver needs from PageIndex.
type PageLookup interface {
	LookupInLane(relPermalink, lang, version string) *engine.Page
}

// URLResolverFunc is the URL resolver signature needed by the link resolver.
type URLResolverFunc func(relPath, lang, version string) string
