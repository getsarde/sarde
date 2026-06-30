package linkbuttongroup

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type linkButtonGroupRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &linkButtonGroupRenderer{} }

func (r *linkButtonGroupRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindLinkButtonGroupBlock, r.render)
}

func (r *linkButtonGroupRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<div class=\"sarde-link-button-group not-content\">\n")
	} else {
		_, _ = w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}
