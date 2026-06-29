package cardgrid

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type cardGridRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &cardGridRenderer{} }

func (r *cardGridRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindCardGridBlock, r.render)
}

func (r *cardGridRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		g := node.(*CardGridBlock)
		cls := "sarde-card-grid"
		if g.Cols >= 2 && g.Cols <= 4 {
			cls += fmt.Sprintf(" sarde-card-grid-%d", g.Cols)
		}
		_, _ = fmt.Fprintf(w, "<div class=\"%s\">\n", cls)
	} else {
		_, _ = w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}
