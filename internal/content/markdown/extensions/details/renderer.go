package details

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/getsarde/sarde/internal/content/markdown/htmlutil"
)

type detailsRenderer struct{}

// NewRenderer returns a new details renderer.
func NewRenderer() renderer.NodeRenderer {
	return &detailsRenderer{}
}

func (r *detailsRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindDetailsBlock, r.renderDetails)
}

func (r *detailsRenderer) renderDetails(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	details := node.(*DetailsBlock)

	if entering {
		openAttr := ""
		if details.Open {
			openAttr = " open"
		}
		_, _ = fmt.Fprintf(w, "<details class=\"sarde-details not-content\"%s>\n", openAttr)
		_, _ = fmt.Fprintf(w, "<summary class=\"sarde-details-summary\">%s</summary>\n", htmlutil.EscapeHTML(details.Summary))
		_, _ = w.WriteString("<div class=\"sarde-details-content\">\n")
	} else {
		_, _ = w.WriteString("</div>\n</details>\n")
	}

	return ast.WalkContinue, nil
}

