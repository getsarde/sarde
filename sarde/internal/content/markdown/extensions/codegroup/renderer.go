package codegroup

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
)

type codeGroupRenderer struct{}

// NewRenderer returns a new code group renderer.
func NewRenderer() renderer.NodeRenderer {
	return &codeGroupRenderer{}
}

func (r *codeGroupRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindCodeGroupBlock, r.renderCodeGroup)
}

func (r *codeGroupRenderer) renderCodeGroup(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		// Collect code block labels from fenced code block info strings
		var labels []string
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			if fc, ok := c.(*ast.FencedCodeBlock); ok {
				info := string(fc.Info.Text(source))
				label := extractLabel(info)
				labels = append(labels, label)
			}
		}

		_, _ = w.WriteString("<div class=\"sarde-code-group\">\n<div class=\"sarde-code-group-header\" role=\"tablist\">\n")

		for i, label := range labels {
			activeClass := ""
			ariaSelected := "false"
			if i == 0 {
				activeClass = " is-active"
				ariaSelected = "true"
			}
			_, _ = fmt.Fprintf(w, "<button class=\"sarde-code-group-tab%s\" role=\"tab\" aria-selected=\"%s\" data-tab=\"%d\" data-tab-label=\"%s\">%s</button>\n",
				activeClass, ariaSelected, i, htmlutil.EscapeHTML(label), htmlutil.EscapeHTML(label))
		}

		_, _ = w.WriteString("</div>\n<div class=\"sarde-code-group-panels\">\n")

		// Render each code block as a panel
		idx := 0
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			if fc, ok := c.(*ast.FencedCodeBlock); ok {
				hiddenAttr := ""
				activeClass := ""
				if idx == 0 {
					activeClass = " is-active"
				} else {
					hiddenAttr = " hidden"
				}
				_, _ = fmt.Fprintf(w, "<div class=\"sarde-code-group-panel%s\" role=\"tabpanel\" data-tab=\"%d\" data-tab-label=\"%s\"%s>\n",
					activeClass, idx, htmlutil.EscapeHTML(labels[idx]), hiddenAttr)

				// Render the code block content
				info := string(fc.Info.Text(source))
				lang := extractLang(info)

				_, _ = fmt.Fprintf(w, "<pre><code class=\"language-%s\">", htmlutil.EscapeHTML(lang))
				lines := fc.Lines()
				for i := 0; i < lines.Len(); i++ {
					line := lines.At(i)
					_, _ = w.Write(escapeHTMLBytes(line.Value(source)))
				}
				_, _ = w.WriteString("</code></pre>\n")
				_, _ = w.WriteString("</div>\n")
				idx++
			}
		}

		_, _ = w.WriteString("</div>\n</div>\n")
		return ast.WalkSkipChildren, nil
	}

	return ast.WalkContinue, nil
}

// extractLabel gets the display label from a fenced code info string.
// e.g., "js [JavaScript]" -> "JavaScript", "python" -> "python"
func extractLabel(info string) string {
	info = strings.TrimSpace(info)
	if idx := strings.Index(info, "["); idx >= 0 {
		if end := strings.Index(info[idx:], "]"); end >= 0 {
			return info[idx+1 : idx+end]
		}
	}
	// Fall back to language name
	parts := strings.Fields(info)
	if len(parts) > 0 {
		return parts[0]
	}
	return "code"
}

// extractLang gets the language identifier from info string.
func extractLang(info string) string {
	parts := strings.Fields(info)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}


func escapeHTMLBytes(b []byte) []byte {
	var result []byte
	for _, c := range b {
		switch c {
		case '&':
			result = append(result, "&amp;"...)
		case '<':
			result = append(result, "&lt;"...)
		case '>':
			result = append(result, "&gt;"...)
		case '"':
			result = append(result, "&quot;"...)
		default:
			result = append(result, c)
		}
	}
	return result
}
