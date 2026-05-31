package linkrender

import (
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/yuin/goldmark/ast"
	gmrenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// BrokenLink records an unresolvable internal link.
type BrokenLink struct {
	SourceFile string
	RawHref    string
	Dest       ParsedDest
}

// PendingAnchorCheck records a deferred anchor validation.
type PendingAnchorCheck struct {
	SourceFile      string
	TargetPermalink string
	Fragment        string
	RawHref         string
}

// Renderer is a Goldmark node renderer for ast.KindLink that resolves internal
// links through the page index and URL resolver. External links pass through.
//
// The CurrentPage, PageIndex, and URLResolver fields are swapped before each
// page render (same pattern as imagerender.Renderer.Lookup).
type Renderer struct {
	CurrentPage *engine.Page
	PageIndex   *content.PageIndex
	URLResolver *engine.URLResolver
	Policy      string // "error" | "warn" | "ignore"; empty = "error"

	PendingAnchors []PendingAnchorCheck
	BrokenLinks    []BrokenLink
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
	r.BrokenLinks = r.BrokenLinks[:0]
}

// DrainPendingAnchors returns and clears collected anchor checks.
func (r *Renderer) DrainPendingAnchors() []PendingAnchorCheck {
	out := make([]PendingAnchorCheck, len(r.PendingAnchors))
	copy(out, r.PendingAnchors)
	r.PendingAnchors = r.PendingAnchors[:0]
	return out
}

// DrainBrokenLinks returns and clears collected broken links.
func (r *Renderer) DrainBrokenLinks() []BrokenLink {
	out := make([]BrokenLink, len(r.BrokenLinks))
	copy(out, r.BrokenLinks)
	r.BrokenLinks = r.BrokenLinks[:0]
	return out
}

// RegisterFuncs registers the renderer for ast.KindLink.
func (r *Renderer) RegisterFuncs(reg gmrenderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindLink, r.renderLink)
}

func (r *Renderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		w.WriteString("</a>")
		return ast.WalkContinue, nil
	}

	n := node.(*ast.Link)
	href := string(n.Destination)

	// Graceful degradation: no page context → pass through.
	if r.CurrentPage == nil || r.PageIndex == nil || r.URLResolver == nil {
		r.writeOpenTag(w, href, n)
		return ast.WalkContinue, nil
	}

	dest := ClassifyDest(href)

	// External and empty links pass through.
	if dest.Kind == LinkExternal {
		r.writeOpenTag(w, href, n)
		return ast.WalkContinue, nil
	}

	// Resolve using the page index and URL resolver.
	result := ResolveInternalLink(dest, r.CurrentPage, r.PageIndex, r.URLResolver.URL)

	if !result.Found {
		r.handleBroken(dest)
		r.writeOpenTag(w, r.brokenHref(href), n)
		return ast.WalkContinue, nil
	}

	// Record pending anchor validation if there's a fragment.
	if dest.Fragment != "" && result.TargetPermalink != "" {
		r.PendingAnchors = append(r.PendingAnchors, PendingAnchorCheck{
			SourceFile:      r.CurrentPage.FilePath,
			TargetPermalink: result.TargetPermalink,
			Fragment:        dest.Fragment,
			RawHref:         href,
		})
	}

	r.writeOpenTag(w, result.URL, n)
	return ast.WalkContinue, nil
}

func (r *Renderer) handleBroken(dest ParsedDest) {
	policy := r.effectivePolicy()
	if policy == "ignore" {
		return
	}
	r.BrokenLinks = append(r.BrokenLinks, BrokenLink{
		SourceFile: r.CurrentPage.FilePath,
		RawHref:    dest.Raw,
		Dest:       dest,
	})
}

func (r *Renderer) brokenHref(original string) string {
	policy := r.effectivePolicy()
	if policy == "ignore" {
		return original
	}
	return "#"
}

func (r *Renderer) effectivePolicy() string {
	if r.Policy == "" {
		return "error"
	}
	return r.Policy
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
