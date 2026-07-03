package math

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// --- Inline math parser ($...$) ---

type inlineMathParser struct{}

// NewInlineParser returns a new inline math parser.
func NewInlineParser() parser.InlineParser {
	return &inlineMathParser{}
}

func (p *inlineMathParser) Trigger() []byte {
	return []byte{'$'}
}

func (p *inlineMathParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, segment := block.PeekLine()
	if len(line) == 0 {
		return nil
	}

	// Skip $$ (handled by block parser)
	if len(line) > 1 && line[1] == '$' {
		return nil
	}

	// Find closing $
	content := string(line[1:])
	closeIdx := strings.Index(content, "$")
	if closeIdx <= 0 {
		return nil
	}

	expr := content[:closeIdx]
	if len(strings.TrimSpace(expr)) == 0 {
		return nil
	}

	// Don't match if the closing $ is immediately followed by a digit: this is
	// almost certainly a second currency amount, not a math span. Without this,
	// prose like "costs $9 and $29" gets "9 and " parsed as an expression.
	if closeIdx+1 < len(content) && content[closeIdx+1] >= '0' && content[closeIdx+1] <= '9' {
		return nil
	}

	node := &InlineMath{
		Expression: expr,
	}

	// Advance past the opening $, content, and closing $
	block.Advance(segment.Start + 1 + closeIdx + 1 - segment.Start)

	return node
}

// --- Display math parser ($$...$$) ---

type displayMathParser struct{}

// NewBlockParser returns a new display math block parser.
func NewBlockParser() parser.BlockParser {
	return &displayMathParser{}
}

func (p *displayMathParser) Trigger() []byte {
	return []byte{'$'}
}

func (p *displayMathParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))

	if !strings.HasPrefix(lineStr, "$$") {
		return nil, parser.NoChildren
	}

	// Check if it's a single-line display math: $$ expr $$
	rest := strings.TrimSpace(lineStr[2:])
	if strings.HasSuffix(rest, "$$") && len(rest) > 2 {
		expr := strings.TrimSuffix(rest, "$$")
		reader.Advance(len(line))
		node := &DisplayMath{
			Expression: strings.TrimSpace(expr),
		}
		return node, parser.NoChildren
	}

	reader.Advance(len(line))
	node := &DisplayMath{}
	return node, parser.NoChildren | parser.RequireParagraph
}

func (p *displayMathParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))

	dm := node.(*DisplayMath)

	if strings.HasPrefix(lineStr, "$$") {
		reader.Advance(len(line))
		return parser.Close
	}

	// Accumulate expression lines
	lineContent := string(line)
	if len(lineContent) > 0 && lineContent[len(lineContent)-1] == '\n' {
		lineContent = lineContent[:len(lineContent)-1]
	}
	if dm.Expression != "" {
		dm.Expression += "\n"
	}
	dm.Expression += lineContent

	reader.Advance(len(line))
	return parser.Continue | parser.NoChildren
}

func (p *displayMathParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
}

func (p *displayMathParser) CanInterruptParagraph() bool {
	return true
}

func (p *displayMathParser) CanAcceptIndentedLine() bool {
	return false
}
