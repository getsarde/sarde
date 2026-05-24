package badge

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
)

type badgeRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &badgeRenderer{} }

func (r *badgeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindBadge, r.render)
}

func (r *badgeRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		b := node.(*Badge)
		_, _ = fmt.Fprintf(w, `<span class="sarde-badge sarde-badge-%s" role="status" aria-label="%s">%s</span>`,
			htmlutil.EscapeHTML(b.BadgeType), htmlutil.EscapeHTML(b.Content), htmlutil.EscapeHTML(b.Content))
	}
	return ast.WalkSkipChildren, nil
}

