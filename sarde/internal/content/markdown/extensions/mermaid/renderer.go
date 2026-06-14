package mermaid

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
	"github.com/frostybee/sarde/internal/content/markdown/htmlutil"
)

// Renderer intercepts fenced code blocks with language "sarde-mermaid" and renders
// them as <div class="sarde-mermaid"> for client-side rendering.
type mermaidRenderer struct{}

func NewRenderer() renderer.NodeRenderer { return &mermaidRenderer{} }

func (r *mermaidRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.render)
}

func (r *mermaidRenderer) render(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	fcb := node.(*ast.FencedCodeBlock)
	if fcb.Info == nil {
		// Not our concern — let the next renderer handle it
		return ast.WalkContinue, nil
	}

	lang := strings.TrimSpace(string(fcb.Info.Text(source)))
	if lang != "sarde-mermaid" {
		return ast.WalkContinue, nil
	}

	// Extract code
	var code strings.Builder
	lines := fcb.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		code.Write(line.Value(source))
	}

	_, _ = w.WriteString("<div class=\"sarde-mermaid\" role=\"img\" aria-label=\"Mermaid diagram\">\n")
	_, _ = w.WriteString(htmlutil.EscapeHTML(code.String()))
	_, _ = w.WriteString("</div>\n")

	return ast.WalkSkipChildren, nil
}

