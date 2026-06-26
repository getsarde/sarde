package badge

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"

	"github.com/getsarde/sarde/internal/content/markdown/htmlutil"
	"github.com/getsarde/sarde/internal/content/markdown/icons"
)

var defaultIcons = map[string]string{
	"default":   "info",
	"primary":   "circle-check",
	"secondary": "circle-minus",
	"success":   "circle-check",
	"warning":   "triangle-alert",
	"danger":    "circle-x",
	"info":      "info",
	"note":      "pencil",
	"tip":       "sparkles",
}

type badgeRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &badgeRenderer{} }

func (r *badgeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindBadge, r.render)
}

func (r *badgeRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		b := node.(*Badge)

		iconName := b.Icon
		if iconName == "" {
			iconName = defaultIcons[b.BadgeType]
		}
		if iconName == "" {
			iconName = "info"
		}
		iconSVG := icons.GetWithClass(iconName, "sarde-badge-icon")

		_, _ = fmt.Fprintf(w, `<span class="sarde-badge sarde-badge-%s" role="status" aria-label="%s">%s%s</span>`,
			htmlutil.EscapeHTML(b.BadgeType), htmlutil.EscapeHTML(b.Content), iconSVG, htmlutil.EscapeHTML(b.Content))
	}
	return ast.WalkSkipChildren, nil
}
