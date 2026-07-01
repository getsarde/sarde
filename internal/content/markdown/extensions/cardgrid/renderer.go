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
		cls := "sarde-card-grid not-content"
		cols := g.Cols
		if g.Stagger {
			cols = 2
		}
		if cols >= 2 && cols <= 4 {
			cls += fmt.Sprintf(" sarde-card-grid-%d", cols)
		}
		if g.Stagger {
			cls += " sarde-card-grid-stagger"
		}
		_, _ = fmt.Fprintf(w, "<div class=\"%s\">\n", cls)
	} else {
		_, _ = w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}
