package linkrender

import (
	"path"

	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/engine"
)

// ResolveResult holds the outcome of resolving an internal link.
type ResolveResult struct {
	URL             string // final resolved URL (empty if not found)
	TargetPermalink string // target page's Permalink (for anchor validation)
	Found           bool
}

// ResolveInternalLink resolves a parsed destination in the current page's lane,
// returning the final URL via the URL resolver. External and anchor-only links
// short-circuit without index lookup.
func ResolveInternalLink(
	dest ParsedDest,
	currentPage *engine.Page,
	pageIndex PageLookup,
	urlResolver URLResolverFunc,
) ResolveResult {
	version := engine.ResolvePageVersion(currentPage)

	switch dest.Kind {
	case LinkExternal:
		return ResolveResult{URL: dest.Raw, Found: true}

	case LinkAnchorOnly:
		url := urlResolver(currentPage.RelPermalink, currentPage.Lang, version)
		return ResolveResult{
			URL:             url + "#" + dest.Fragment,
			TargetPermalink: currentPage.Permalink,
			Found:           true,
		}

	case LinkAmbiguous:
		return ResolveResult{Found: false}

	case LinkRelative:
		return resolveRelative(dest, currentPage, pageIndex, urlResolver, version)

	case LinkContentRoot:
		return resolveContentRoot(dest, currentPage, pageIndex, urlResolver, version)
	}

	return ResolveResult{Found: false}
}

func resolveRelative(
	dest ParsedDest,
	currentPage *engine.Page,
	pageIndex PageLookup,
	urlResolver URLResolverFunc,
	version string,
) ResolveResult {
	baseDir := path.Dir(effectiveRelPath(currentPage))
	logical := path.Clean(path.Join(baseDir, dest.Path))

	return lookupAndResolve(logical, dest, currentPage, pageIndex, urlResolver, version)
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
	pageIndex PageLookup,
	urlResolver URLResolverFunc,
	version string,
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

	return lookupAndResolve(logical, dest, currentPage, pageIndex, urlResolver, version)
}


func lookupAndResolve(
	logical string,
	dest ParsedDest,
	currentPage *engine.Page,
	pageIndex PageLookup,
	urlResolver URLResolverFunc,
	version string,
) ResolveResult {
	// Compute the expected RelPermalink from the logical filesystem path.
	relPermalink := content.ComputePermalinkFromRelPath(logical + ".md")

	target := pageIndex.LookupInLane(relPermalink, currentPage.Lang, currentPage.Version)

	// Fallback: try as a directory index (e.g. ./guides/ → guides/_index.md).
	if target == nil {
		indexLogical := logical + "/_index"
		indexRelPermalink := content.ComputePermalinkFromRelPath(indexLogical + ".md")
		target = pageIndex.LookupInLane(indexRelPermalink, currentPage.Lang, currentPage.Version)
	}

	// Fallback: try as index.md (leaf bundle).
	if target == nil {
		indexLogical := logical + "/index"
		indexRelPermalink := content.ComputePermalinkFromRelPath(indexLogical + ".md")
		target = pageIndex.LookupInLane(indexRelPermalink, currentPage.Lang, currentPage.Version)
	}

	if target == nil {
		return ResolveResult{Found: false}
	}

	url := urlResolver(target.RelPermalink, currentPage.Lang, version)
	if dest.Fragment != "" {
		url += "#" + dest.Fragment
	}
	if dest.Query != "" {
		url += "?" + dest.Query
	}

	return ResolveResult{
		URL:             url,
		TargetPermalink: target.Permalink,
		Found:           true,
	}
}

// PageLookup is the interface the resolver needs from PageIndex.
type PageLookup interface {
	LookupInLane(relPermalink, lang, version string) *engine.Page
}

// URLResolverFunc is the URL resolver signature needed by the link resolver.
type URLResolverFunc func(relPath, lang, version string) string
