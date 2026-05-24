package mermaid

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type Extension struct{}

func (e *Extension) Extend(m goldmark.Markdown) {
	// Priority 50 — runs before Chroma renderer (priority 100)
	m.Renderer().AddOptions(
		renderer.WithNodeRenderers(util.Prioritized(NewRenderer(), 50)),
	)
}
