package highlight

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type highlightParser struct{}

func NewParser() parser.InlineParser { return &highlightParser{} }

func (p *highlightParser) Trigger() []byte { return []byte{'='} }

func (p *highlightParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, seg := block.PeekLine()
	s := string(line)

	if !strings.HasPrefix(s, "==") {
		return nil
	}

	// Find closing ==
	closeIdx := strings.Index(s[2:], "==")
	if closeIdx < 1 {
		return nil
	}

	content := s[2 : 2+closeIdx]
	node := &Highlight{}
	node.AppendChild(node, ast.NewString([]byte(content)))
	block.Advance(seg.Start + 2 + closeIdx + 2 - seg.Start)
	return node
}
