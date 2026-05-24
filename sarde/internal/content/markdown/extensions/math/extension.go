package math

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Extension is a goldmark extension that adds math expression support.
// Inline: $expression$ -> <span class="sarde-math math-inline">
// Display: $$expression$$ -> <div class="sarde-math math-display">
type Extension struct{}

// Extend implements goldmark.Extender.
func (e *Extension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithBlockParsers(
			util.Prioritized(NewBlockParser(), 500),
		),
		parser.WithInlineParsers(
			util.Prioritized(NewInlineParser(), 500),
		),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(
			util.Prioritized(NewRenderer(), 500),
		),
	)
}
