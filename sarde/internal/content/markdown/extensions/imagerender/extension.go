package imagerender

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Extension registers the custom image renderer with Goldmark.
type Extension struct {
	Lookup   ImageLookupFunc // legacy: creates a new Renderer with this lookup
	Renderer *Renderer       // preferred: reuse an existing Renderer (lookup swappable at runtime)
}

// Extend implements goldmark.Extender.
func (e *Extension) Extend(m goldmark.Markdown) {
	r := e.Renderer
	if r == nil {
		r = NewRenderer(e.Lookup)
	}
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(r, 50),
		),
	)
}
