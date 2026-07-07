package columns

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type columnsRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &columnsRenderer{} }

func (r *columnsRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindColumnsBlock, r.renderColumnsBlock)
	reg.Register(KindColumnBlock, r.renderColumnBlock)
}

func (r *columnsRenderer) renderColumnsBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		c := node.(*ColumnsBlock)
		cols := c.Cols
		if cols < minCols || cols > maxCols {
			cols = defaultCols
		}
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-columns not-content sarde-columns-%d\">\n", cols)
	} else {
		_, _ = w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}

func (r *columnsRenderer) renderColumnBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<div class=\"sarde-column\">\n")
	} else {
		_, _ = w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}
