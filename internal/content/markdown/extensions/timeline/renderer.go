package timeline

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/getsarde/sarde/internal/content/markdown/htmlutil"
)

type timelineRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &timelineRenderer{} }

func (r *timelineRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindTimelineBlock, r.renderBlock)
	reg.Register(KindTimelineItem, r.renderItem)
}

func (r *timelineRenderer) renderBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<div class=\"sarde-timeline not-content\">\n")
	} else {
		_, _ = w.WriteString("</div>\n")
	}
	return ast.WalkContinue, nil
}

func (r *timelineRenderer) renderItem(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	item := node.(*TimelineItem)

	if entering {
		_, _ = w.WriteString("<div class=\"sarde-timeline-entry\">\n")
		_, _ = w.WriteString("<div class=\"sarde-timeline-marker\">")
		if item.Title != "" {
			_, _ = fmt.Fprintf(w, "<span class=\"sarde-timeline-date\">%s</span>", htmlutil.EscapeHTML(item.Title))
		}
		_, _ = w.WriteString("</div>\n")
		_, _ = w.WriteString("<div class=\"sarde-timeline-content\">\n")
	} else {
		_, _ = w.WriteString("</div>\n</div>\n")
	}

	return ast.WalkContinue, nil
}

