package genericdirective

import (
	"github.com/getsarde/sarde/internal/directive"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	gmrenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Extension registers the generic directive parser and renderer. It must be
// added AFTER every built-in extension: same-trigger block parsers run in
// registration order, which is what lets built-ins win name conflicts.
type Extension struct {
	Registry *directive.Registry
}

func (e *Extension) Extend(m goldmark.Markdown) {
	if e.Registry == nil || e.Registry.Empty() {
		return
	}
	m.Parser().AddOptions(parser.WithBlockParsers(util.Prioritized(NewParser(e.Registry), 500)))
	m.Renderer().AddOptions(gmrenderer.WithNodeRenderers(util.Prioritized(NewRenderer(e.Registry, m.Renderer()), 500)))
}
