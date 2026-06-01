package linkrender

import (
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
		FromPage:   page,
		FromFile:   page.FilePath,
		RawDest:    dest.Raw,
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
