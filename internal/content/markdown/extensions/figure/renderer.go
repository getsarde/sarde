package figure

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/getsarde/sarde/internal/content/markdown/htmlutil"
)

type figureRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &figureRenderer{} }

func (r *figureRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindFigureBlock, r.render)
}

func (r *figureRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<figure class=\"sarde-figure\">\n")
	} else {
		f := node.(*FigureBlock)
		if f.Caption != "" {
			_, _ = fmt.Fprintf(w, "<figcaption>%s</figcaption>\n", htmlutil.EscapeHTML(f.Caption))
		}
		_, _ = w.WriteString("</figure>\n")
	}
	return ast.WalkContinue, nil
}

