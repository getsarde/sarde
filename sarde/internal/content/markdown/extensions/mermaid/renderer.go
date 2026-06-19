package mermaid

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
)

type mermaidRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &mermaidRenderer{} }

func (r *mermaidRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindMermaidBlock, r.render)
}

func (r *mermaidRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	mb := node.(*MermaidBlock)

	_, _ = w.WriteString("<div class=\"sarde-mermaid\" role=\"img\" aria-label=\"Mermaid diagram\">\n")
	_, _ = w.WriteString(htmlutil.EscapeHTML(mb.Source))
	_, _ = w.WriteString("</div>\n")

	return ast.WalkSkipChildren, nil
}
