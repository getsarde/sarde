package mermaid

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type Extension struct{}

func (e *Extension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		parser.WithASTTransformers(util.Prioritized(newTransformer(), 100)),
	)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(util.Prioritized(NewRenderer(), 500)),
	)
}
