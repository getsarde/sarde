package accordion

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type accordionRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &accordionRenderer{} }

func (r *accordionRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindAccordionBlock, r.render)
}

func (r *accordionRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		a := node.(*AccordionBlock)
		if a.Independent {
			_, _ = w.WriteString("<div class=\"sarde-details-group not-content\" data-independent>\n")
		} else {
			_, _ = w.WriteString("<div class=\"sarde-details-group not-content\">\n")
		}
	} else {
		_, _ = w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}
