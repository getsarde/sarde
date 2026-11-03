package annotation

import (
	"fmt"

	"github.com/getsarde/sarde/internal/content/markdown/htmlutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type annotationRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &annotationRenderer{} }

func (r *annotationRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindAnnotation, r.render)
}

func (r *annotationRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		a := node.(*Annotation)
		id := htmlutil.PageScopedID("ann", source, a)
		cls := "sarde-annotation"
		switch a.Style {
		case "highlight":
			cls += " sarde-annotation-highlight"
		case "plain":
			cls += " sarde-annotation-plain"
		}
		_, _ = fmt.Fprintf(w, `<abbr class="%s" aria-describedby="%s" tabindex="0">%s<span class="sarde-annotation-tooltip" role="tooltip" id="%s">%s</span></abbr>`,
			cls, id, htmlutil.EscapeHTML(a.Label), id, htmlutil.EscapeHTML(a.Explanation))
	}
	return ast.WalkSkipChildren, nil
}
