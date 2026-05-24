package math

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
)

type mathRenderer struct{}

// NewRenderer returns a new math renderer.
func NewRenderer() renderer.NodeRenderer {
	return &mathRenderer{}
}

func (r *mathRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindInlineMath, r.renderInlineMath)
	reg.Register(KindDisplayMath, r.renderDisplayMath)
}

func (r *mathRenderer) renderInlineMath(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		math := node.(*InlineMath)
		_, _ = fmt.Fprintf(w, `<span class="sarde-math sarde-math-inline">$%s$</span>`, htmlutil.EscapeHTML(math.Expression))
	}
	return ast.WalkSkipChildren, nil
}

func (r *mathRenderer) renderDisplayMath(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		math := node.(*DisplayMath)
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-math sarde-math-block\">$$\n%s\n$$</div>", htmlutil.EscapeHTML(math.Expression))
	}
	return ast.WalkSkipChildren, nil
}

