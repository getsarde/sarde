package linkrender

import (
	"path"
	"strings"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/links"
)

// CheckResult holds the outcome of checking whether an href resolves.
type CheckResult struct {
	Status          links.LinkStatus
	TargetPermalink string
	HasFragment     bool
}

// CheckHref determines whether href resolves in the context of page, using the
// same dispatch as Renderer.ResolveHref but without side effects (no LinkGraph
// recording, no pending-anchor accumulation). The caller is responsible for
// anchor validation via PageIndex.HasHeading when HasFragment is true.
//
// idx is needed for site-absolute lookups via LookupInLaneWithDefaultFallback;
// ctx.PageIndex is used for lane-aware resolution of relative/content-root links.
func CheckHref(href string, page *engine.Page, ctx ResolveContext,
	idx *content.PageIndex, resolver *engine.URLResolver, siteRootEscapePrefix string) CheckResult {

	if siteRootEscapePrefix != "" && strings.HasPrefix(href, siteRootEscapePrefix) {
		return CheckResult{Status: links.StatusExternal}
	}

	dest := ClassifyDest(href)

	if dest.Kind == LinkExternal {
		raw := dest.Raw
		if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
			return checkSiteAbsolute(raw, page, resolver, idx)
		}
		return CheckResult{Status: links.StatusExternal}
	}

	result := ResolveInternalLink(dest, page, ctx)

	if !result.Found {
		if dest.Kind == LinkContentRoot && !hasMarkdownExtension(dest.Raw) {
			return checkSiteAbsolute(dest.Raw, page, resolver, idx)
		}
		if dest.Kind == LinkAmbiguous {
			return CheckResult{Status: links.StatusAmbiguous}
		}
		return CheckResult{Status: links.StatusBrokenTarget}
	}

	return CheckResult{
		Status:          links.StatusOK,
		TargetPermalink: result.TargetPermalink,
		HasFragment:     dest.Fragment != "",
	}
}

// checkSiteAbsolute mirrors Renderer.resolveSiteAbsolute's classification logic
// without producing a resolved URL or recording to the link graph.
func checkSiteAbsolute(raw string, page *engine.Page, resolver *engine.URLResolver, idx *content.PageIndex) CheckResult {
	pathPart := raw
	var fragment string
	if i := strings.IndexByte(pathPart, '#'); i >= 0 {
		fragment = pathPart[i+1:]
		pathPart = pathPart[:i]
	}
	if i := strings.IndexByte(pathPart, '?'); i >= 0 {
		pathPart = pathPart[:i]
	}

	relP := content.NormalizePermalink(pathPart)

	target := LookupInLaneWithDefaultFallback(idx, relP, page, resolver)
	if target != nil {
		return CheckResult{
			Status:          links.StatusOK,
			TargetPermalink: target.Permalink,
			HasFragment:     fragment != "",
		}
	}

	if pathPart == "/" || path.Ext(path.Base(pathPart)) != "" {
		return CheckResult{Status: links.StatusExternal}
	}
	return CheckResult{Status: links.StatusUnverified}
}
