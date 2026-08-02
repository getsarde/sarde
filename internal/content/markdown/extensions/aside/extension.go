package aside

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Extension is a goldmark extension that adds aside block support.
// Style selects the icon set: "galaxy" swaps note/tip/danger icons to match
// the galaxy aside style; any other value uses the classic icons.
type Extension struct {
	Style string
}

// Extend implements goldmark.Extender.
func (e *Extension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(
			util.Prioritized(NewParser(), 500),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(NewRenderer(e.Style), 500),
		),
	)
}
