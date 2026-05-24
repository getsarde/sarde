package card

import (
	"fmt"

	"github.com/frostybee/sarde/internal/content/markdown/icons"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
)

type cardRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &cardRenderer{} }

func (r *cardRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindCardBlock, r.render)
}

func (r *cardRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		c := node.(*CardBlock)
		_, _ = w.WriteString("<div class=\"sarde-card\">\n")
		if c.Title != "" || c.Icon != "" {
			_, _ = w.WriteString("<div class=\"sarde-card-header\">\n")
			if c.Icon != "" {
				iconSVG := icons.GetWithClass(c.Icon, "sarde-card-icon-svg")
				if iconSVG != "" {
					_, _ = fmt.Fprintf(w, "<span class=\"sarde-card-icon\">%s</span>\n", iconSVG)
				}
			}
			if c.Title != "" {
				_, _ = fmt.Fprintf(w, "<span class=\"sarde-card-title\">%s</span>\n", htmlutil.EscapeHTML(c.Title))
			}
			_, _ = w.WriteString("</div>\n")
		}
		_, _ = w.WriteString("<div class=\"sarde-card-body\">\n")
	} else {
		_, _ = w.WriteString("</div>\n</div>\n")
	}
	return ast.WalkContinue, nil
}

