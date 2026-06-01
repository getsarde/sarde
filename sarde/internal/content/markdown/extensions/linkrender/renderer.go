package linkrender

import (
	"path"
	"strings"

	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/links"
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

	PendingAnchors []links.PendingAnchorCheck
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
}

// DrainPendingAnchors returns and clears collected anchor checks.
func (r *Renderer) DrainPendingAnchors() []links.PendingAnchorCheck {
	out := make([]links.PendingAnchorCheck, len(r.PendingAnchors))
	copy(out, r.PendingAnchors)
	r.PendingAnchors = r.PendingAnchors[:0]
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

	result := ResolveInternalLink(dest, r.CurrentPage, r.PageIndex, r.URLResolver.URL)

	if !result.Found {
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

	target := r.PageIndex.LookupInLane(relP, r.CurrentPage.Lang, r.CurrentPage.Version)
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

// withSuffix re-attaches a #fragment and/or ?query to a resolved URL.
func withSuffix(url, fragment, query string) string {
	if fragment != "" {
		url += "#" + fragment
	}
	if query != "" {
		url += "?" + query
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
	r.LinkGraph.Record(links.LinkRef{
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
	})
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
