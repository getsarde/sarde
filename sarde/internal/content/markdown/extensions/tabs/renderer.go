package tabs

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
)

type tabsRenderer struct{}

// NewRenderer returns a new tabs renderer.
func NewRenderer() renderer.NodeRenderer {
	return &tabsRenderer{}
}

func (r *tabsRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindTabsBlock, r.renderTabsBlock)
	reg.Register(KindTabItem, r.renderTabItem)
}

func (r *tabsRenderer) renderTabsBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		_, _ = w.WriteString("<div class=\"sarde-tabs\">\n<div class=\"sarde-tabs-header\" role=\"tablist\">\n")

		// Render tab buttons
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			if tab, ok := c.(*TabItem); ok {
				activeClass := ""
				ariaSelected := "false"
				if tab.Index == 0 {
					activeClass = " is-active"
					ariaSelected = "true"
				}
				_, _ = fmt.Fprintf(w, "<button class=\"sarde-tab-button%s\" role=\"tab\" aria-selected=\"%s\" data-tab=\"%d\" data-tab-label=\"%s\">%s</button>\n",
					activeClass, ariaSelected, tab.Index, htmlutil.EscapeHTML(tab.Label), htmlutil.EscapeHTML(tab.Label))
			}
		}

		_, _ = w.WriteString("</div>\n<div class=\"sarde-tabs-panels\">\n")
	} else {
		_, _ = w.WriteString("</div>\n</div>\n")
	}
	return ast.WalkContinue, nil
}

func (r *tabsRenderer) renderTabItem(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	tab := node.(*TabItem)

	if entering {
		hiddenAttr := ""
		activeClass := ""
		if tab.Index == 0 {
			activeClass = " is-active"
		} else {
			hiddenAttr = " hidden"
		}
		_, _ = fmt.Fprintf(w, "<div class=\"sarde-tab-panel%s\" role=\"tabpanel\" data-tab=\"%d\" data-tab-label=\"%s\"%s>\n",
			activeClass, tab.Index, htmlutil.EscapeHTML(tab.Label), hiddenAttr)
	} else {
		_, _ = w.WriteString("</div>\n")
	}

	return ast.WalkContinue, nil
}

