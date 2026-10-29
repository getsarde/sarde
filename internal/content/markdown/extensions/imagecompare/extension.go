package imagecompare

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type Extension struct {
	// AssetURL resolves a site-absolute image path, adding the base path;
	// nil leaves paths as written.
	AssetURL func(string) string
}

func (e *Extension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithBlockParsers(util.Prioritized(NewParser(), 500)))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(NewRenderer(e.AssetURL), 500)))
}
