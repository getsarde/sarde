package badgegroup

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type badgeGroupRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &badgeGroupRenderer{} }

func (r *badgeGroupRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindBadgeGroupBlock, r.render)
}

func (r *badgeGroupRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<div class=\"sarde-badge-group\">\n")
	} else {
		_, _ = w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}
