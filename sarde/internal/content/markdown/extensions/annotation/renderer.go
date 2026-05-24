package annotation

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
)

type annotationRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &annotationRenderer{} }

func (r *annotationRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindAnnotation, r.render)
}

func (r *annotationRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		a := node.(*Annotation)
		_, _ = fmt.Fprintf(w, `<abbr class="sarde-annotation" title="%s" role="tooltip" tabindex="0">%s<span class="sarde-annotation-tooltip">%s</span></abbr>`,
			htmlutil.EscapeHTML(a.Explanation), htmlutil.EscapeHTML(a.Label), htmlutil.EscapeHTML(a.Explanation))
	}
	return ast.WalkSkipChildren, nil
}

