package highlight

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type highlightRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &highlightRenderer{} }

func (r *highlightRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindHighlight, r.render)
}

func (r *highlightRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<mark class=\"sarde-highlight\">")
	} else {
		_, _ = w.WriteString("</mark>")
	}
	return ast.WalkContinue, nil
}
