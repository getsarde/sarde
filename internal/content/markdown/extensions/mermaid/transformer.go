package mermaid

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type transformer struct{}

func newTransformer() parser.ASTTransformer { return &transformer{} }

func (t *transformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	source := reader.Source()
	var toReplace []*ast.FencedCodeBlock

	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		fcb, ok := n.(*ast.FencedCodeBlock)
		if !ok || fcb.Info == nil {
			return ast.WalkContinue, nil
		}
		lang := strings.TrimSpace(string(fcb.Info.Text(source)))
		// Accept the standard "mermaid" fence language alongside the legacy
		// "sarde-mermaid" spelling. Claiming the node here (before Kazari's
		// code-block renderer runs) is what routes it through Sarde's own
		// MermaidBlock renderer and script-injection plugin.
		if lang == "mermaid" || lang == "sarde-mermaid" {
			toReplace = append(toReplace, fcb)
		}
		return ast.WalkContinue, nil
	})
	for _, fcb := range toReplace {
		var code strings.Builder
		lines := fcb.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			code.Write(line.Value(source))
		}

		mb := &MermaidBlock{Source: code.String()}
		parent := fcb.Parent()
		parent.ReplaceChild(parent, fcb, mb)
	}
}
