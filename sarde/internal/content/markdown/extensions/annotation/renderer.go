package annotation

import (
	"fmt"
	"sync/atomic"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/getsarde/sarde/internal/content/markdown/htmlutil"
)

var annCounter atomic.Int64

type annotationRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &annotationRenderer{} }

func (r *annotationRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindAnnotation, r.render)
}

func (r *annotationRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		a := node.(*Annotation)
		id := annCounter.Add(1)
		_, _ = fmt.Fprintf(w, `<abbr class="sarde-annotation" title="%s" aria-describedby="ann-%d" tabindex="0">%s<span class="sarde-annotation-tooltip" role="tooltip" id="ann-%d">%s</span></abbr>`,
			htmlutil.EscapeHTML(a.Explanation), id, htmlutil.EscapeHTML(a.Label), id, htmlutil.EscapeHTML(a.Explanation))
	}
	return ast.WalkSkipChildren, nil
}

