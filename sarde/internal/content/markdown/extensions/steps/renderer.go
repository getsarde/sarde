package steps

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
)

type stepsRenderer struct{}

// NewRenderer returns a new steps renderer.
func NewRenderer() renderer.NodeRenderer {
	return &stepsRenderer{}
}

func (r *stepsRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindStepsBlock, r.renderStepsBlock)
	reg.Register(KindStepItem, r.renderStepItem)
}

func (r *stepsRenderer) renderStepsBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<div class=\"sarde-steps\">\n")
	} else {
		_, _ = w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}

func (r *stepsRenderer) renderStepItem(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	step := node.(*StepItem)

	if entering {
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-step\" data-step=\"%d\">\n", step.Index)
		_, _ = w.WriteString("<div class=\"sarde-step-content\">\n")
		if step.Title != "" {
			_, _ = fmt.Fprintf(w, "<h3 class=\"sarde-step-title\">%s</h3>\n", htmlutil.EscapeHTML(step.Title))
		}
	} else {
		_, _ = w.WriteString("</div>\n</div>\n")
	}

	return ast.WalkContinue, nil
}

