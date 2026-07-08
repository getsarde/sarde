package linkrender

import (
	"path"
	"strings"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/content/markdown/htmlutil"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/links"
	"github.com/yuin/goldmark/ast"
	gmrenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Renderer is a Goldmark node renderer for ast.KindLink that resolves internal
// links through the page index and URL resolver. External links pass through.
//
// The CurrentPage, PageIndex, and URLResolver fields are swapped before each
// page render (same pattern as imagerender.Renderer.Lookup).
type Renderer struct {
	CurrentPage *engine.Page
	PageIndex   *content.PageIndex
	URLResolver *engine.URLResolver
	LinkGraph   *links.LinkGraph

	// Collections is the per-build collection registry, used to detect when a
	// link target lives in an unversioned collection (cross-dimension resolution).
	Collections map[string]*engine.Collection
	// SiteRootEscapePrefix is the configured prefix (e.g. "site:") that routes a
	// link to the site root, bypassing collection/lane logic. Empty disables it.
	SiteRootEscapePrefix string

	PendingAnchors []links.PendingAnchorCheck
	// RecordedRefs buffers a per-render copy of every ref written to LinkGraph,
	// so the build layer can persist the page's resolution snapshot in the page
	// cache and replay it on cache hits.
	RecordedRefs []links.LinkRef
}

// NewRenderer creates a link renderer with no page context.
// Context must be set via SetPage before rendering.
func NewRenderer() *Renderer {
	return &Renderer{}
}

// SetPage sets the current page context for link resolution.
func (r *Renderer) SetPage(page *engine.Page) {
	r.CurrentPage = page
}

// Reset clears accumulated state between page renders.
func (r *Renderer) Reset() {
	r.PendingAnchors = r.PendingAnchors[:0]
	r.RecordedRefs = r.RecordedRefs[:0]
}

// DrainPendingAnchors returns and clears collected anchor checks.
func (r *Renderer) DrainPendingAnchors() []links.PendingAnchorCheck {
	out := make([]links.PendingAnchorCheck, len(r.PendingAnchors))
	copy(out, r.PendingAnchors)
	r.PendingAnchors = r.PendingAnchors[:0]
	return out
}

// DrainRecordedRefs returns and clears the per-render link ref snapshot.
func (r *Renderer) DrainRecordedRefs() []links.LinkRef {
	out := make([]links.LinkRef, len(r.RecordedRefs))
	copy(out, r.RecordedRefs)
	r.RecordedRefs = r.RecordedRefs[:0]
	return out
}

// RegisterFuncs registers the renderer for ast.KindLink.
func (r *Renderer) RegisterFuncs(reg gmrenderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, r.renderLink)
}

// ResolveHref classifies, resolves, records in the link graph, and handles
// pending anchors for a given href. Returns the resolved URL. Card/button
// extension renderers call this to share the same resolution infrastructure.
func (r *Renderer) ResolveHref(href string) (resolvedURL string) {
	if r.CurrentPage == nil || r.PageIndex == nil || r.URLResolver == nil {
		return href
	}

	// Site-root escape (configurable prefix, e.g. "site:/pricing"): resolve at
	// the site root, bypassing all collection/lane logic. Checked before
	// classification so the prefix never collides with normal link parsing.
	if r.SiteRootEscapePrefix != "" && strings.HasPrefix(href, r.SiteRootEscapePrefix) {
		return r.resolveSiteRoot(href, strings.TrimPrefix(href, r.SiteRootEscapePrefix))
	}

	dest := ClassifyDest(href)

	if dest.Kind == LinkExternal {
		raw := dest.Raw
		// Site-absolute internal URL ("/docs/plugins/auth"): not a .md ref, but
		// still internal — it needs base-path application and, when it matches a
		// page, validation. Exclude protocol-relative "//host".
		if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
			return r.resolveSiteAbsolute(raw)
		}
		// Truly external: http(s)://, mailto:, tel:, data:, etc.
		r.recordLinkRef(dest, nil, "", links.StatusExternal)
		return href
	}

	result := ResolveInternalLink(dest, r.CurrentPage, ResolveContext{
		PageIndex:   r.PageIndex,
		URLResolver: r.URLResolver.URL,
		Collections: r.Collections,
	})

	if !result.Found {
		if dest.Kind == LinkContentRoot && !hasMarkdownExtension(dest.Raw) {
			return r.resolveSiteAbsolute(dest.Raw)
		}
		r.recordLinkRef(dest, nil, "", links.StatusBrokenTarget)
		return "#"
	}

	targetPage := r.PageIndex.LookupByPermalink(result.TargetPermalink)

	if dest.Fragment != "" && result.TargetPermalink != "" {
		r.appendPendingAnchor(dest, targetPage, result)
	} else {
		r.recordLinkRef(dest, targetPage, result.URL, links.StatusOK)
	}

	return result.URL
}

// LookupInLaneWithDefaultFallback resolves relPermalink in the page's own
// (lang, version) lane, falling back to the resolver's default language.
// Exported so build-side page-cache staleness verification reproduces
// resolveSiteAbsolute's exact lookup order without risking drift.
func LookupInLaneWithDefaultFallback(idx *content.PageIndex, relPermalink string, page *engine.Page, resolver *engine.URLResolver) *engine.Page {
	target := idx.LookupInLane(relPermalink, page.Lang, page.Version)
	if target == nil && resolver != nil && page.Lang != resolver.DefaultLang {
		target = idx.LookupInLane(relPermalink, resolver.DefaultLang, page.Version)
	}
	return target
}

// resolveSiteAbsolute handles site-absolute internal hrefs without a .md
// extension (e.g. "/docs/plugins/auth"). It applies the base path / lang /
// version prefixing the plain pass-through used to skip, and validates the
// target against the page index in the current lane:
//
//   - Found:        resolve to the canonical URL, record StatusOK (or defer an
//                   anchor check when a #fragment is present).
//   - Not found, page-like (no file extension on the basename, not the site
//     root): apply base path and record StatusUnverified — surfaced as an
//     `unverified_internal` finding (cross-lane target or a typo).
//   - Not found, static asset (basename has an extension) or site root ("/"):
//     apply base path and record StatusExternal — never flagged.
func (r *Renderer) resolveSiteAbsolute(raw string) string {
	// Split fragment and query (mirrors ClassifyDest).
	pathPart := raw
	var fragment, query string
	if idx := strings.IndexByte(pathPart, '#'); idx >= 0 {
		fragment = pathPart[idx+1:]
		pathPart = pathPart[:idx]
	}
	if idx := strings.IndexByte(pathPart, '?'); idx >= 0 {
		query = pathPart[idx+1:]
		pathPart = pathPart[:idx]
	}

	// Normalize to canonical permalink form: trailing slash for page-like paths,
	// unchanged for assets (dotted basename) and the site root.
	relP := content.NormalizePermalink(pathPart)

	// Synthetic dest so graph recording / anchor deferral can reuse the existing
	// helpers. Treated as a content-root internal link.
	dest := ParsedDest{Kind: LinkContentRoot, Raw: raw, Fragment: fragment, Query: query}

	target := LookupInLaneWithDefaultFallback(r.PageIndex, relP, r.CurrentPage, r.URLResolver)
	if target != nil {
		version := engine.ResolvePageVersion(r.CurrentPage)
		url := withSuffix(r.URLResolver.URL(target.RelPermalink, r.CurrentPage.Lang, version), fragment, query)
		result := ResolveResult{URL: url, TargetPermalink: target.Permalink, Found: true}
		if fragment != "" {
			r.appendPendingAnchor(dest, target, result)
		} else {
			r.recordLinkRef(dest, target, url, links.StatusOK)
		}
		return url
	}

	// Not found: apply base path only (no lang/version inference — the target may
	// be a static asset or a legitimate cross-lane page we can't resolve yet).
	url := withSuffix(r.URLResolver.URL(relP, "", ""), fragment, query)
	if pathPart == "/" || path.Ext(path.Base(pathPart)) != "" {
		// Static asset or the site root — never flagged.
		r.recordLinkRef(dest, nil, url, links.StatusExternal)
	} else {
		// Page-like but unresolved — surface as unverified so the checker reports it.
		r.recordLinkRef(dest, nil, url, links.StatusUnverified)
	}
	return url
}

// resolveSiteRoot handles the configured site-root escape (e.g. "site:/pricing").
// It applies the base path with no lang/version segments and records the link as
// a deliberate, unchecked escape: never validated against the index, never flagged.
// This is for links to other parts of the deployment that Sarde does not own.
func (r *Renderer) resolveSiteRoot(rawHref, rest string) string {
	pathPart := rest
	var fragment, query string
	if idx := strings.IndexByte(pathPart, '#'); idx >= 0 {
		fragment = pathPart[idx+1:]
		pathPart = pathPart[:idx]
	}
	if idx := strings.IndexByte(pathPart, '?'); idx >= 0 {
		query = pathPart[idx+1:]
		pathPart = pathPart[:idx]
	}

	relP := content.NormalizePermalink(pathPart)
	url := withSuffix(r.URLResolver.URL(relP, "", ""), fragment, query)

	dest := ParsedDest{Kind: LinkSiteRoot, Raw: rawHref, Fragment: fragment, Query: query}
	r.recordLinkRef(dest, nil, url, links.StatusExternal)
	return url
}

// withSuffix re-attaches a ?query and/or #fragment to a resolved URL in the
// canonical order (path?query#fragment).
func withSuffix(url, fragment, query string) string {
	if query != "" {
		url += "?" + query
	}
	if fragment != "" {
		url += "#" + fragment
	}
	return url
}

func (r *Renderer) appendPendingAnchor(dest ParsedDest, targetPage *engine.Page, result ResolveResult) {
	page := r.CurrentPage
	collName := ""
	if page.Collection != nil {
		collName = page.Collection.Name
	}
	r.PendingAnchors = append(r.PendingAnchors, links.PendingAnchorCheck{
		SourceFile:      page.FilePath,
		TargetPermalink: result.TargetPermalink,
		Fragment:        dest.Fragment,
		RawHref:         dest.Raw,
		FromPage:        page,
		TargetPage:      targetPage,
		Dim: links.DimKey{
			Collection: collName,
			Lang:       page.Lang,
			Version:    page.Version,
		},
		Kind:     mapLinkKind(dest.Kind),
		Resolved: result.URL,
	})
}

func (r *Renderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		w.WriteString("</a>")
		return ast.WalkContinue, nil
	}

	n := node.(*ast.Link)
	href := string(n.Destination)

	resolved := r.ResolveHref(href)

	r.writeOpenTag(w, resolved, n)
	return ast.WalkContinue, nil
}

func (r *Renderer) recordLinkRef(dest ParsedDest, targetPage *engine.Page, resolved string, status links.LinkStatus) {
	if r.LinkGraph == nil {
		return
	}
	page := r.CurrentPage
	collName := ""
	if page.Collection != nil {
		collName = page.Collection.Name
	}
	ref := links.LinkRef{
		FromPage: page,
		FromFile: page.FilePath,
		RawDest:  dest.Raw,
		Dim: links.DimKey{
			Collection: collName,
			Lang:       page.Lang,
			Version:    page.Version,
		},
		Kind:       mapLinkKind(dest.Kind),
		Resolved:   resolved,
		TargetPage: targetPage,
		Fragment:   dest.Fragment,
		Status:     status,
	}
	r.LinkGraph.Record(ref)
	r.RecordedRefs = append(r.RecordedRefs, ref)
}

func mapLinkKind(k LinkKind) links.LinkKind {
	switch k {
	case LinkRelative:
		return links.KindRelative
	case LinkContentRoot:
		return links.KindContentRoot
	case LinkAnchorOnly:
		return links.KindAnchorOnly
	case LinkExternal:
		return links.KindExternal
	case LinkAmbiguous:
		return links.KindAmbiguous
	case LinkSiteRoot:
		return links.KindExternal
	default:
		return links.KindExternal
	}
}

func (r *Renderer) writeOpenTag(w util.BufWriter, href string, n *ast.Link) {
	w.WriteString("<a href=\"")
	w.WriteString(htmlutil.EscapeHTML(href))
	w.WriteByte('"')

	if n.Title != nil {
		w.WriteString(` title="`)
		w.WriteString(htmlutil.EscapeHTML(string(n.Title)))
		w.WriteByte('"')
	}

	w.WriteByte('>')
}
