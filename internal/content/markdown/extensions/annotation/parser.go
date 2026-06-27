package annotation

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type annotationParser struct{}

func NewParser() parser.InlineParser { return &annotationParser{} }

func (p *annotationParser) Trigger() []byte { return []byte{':'} }

func (p *annotationParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, seg := block.PeekLine()
	s := string(line)

	if !strings.HasPrefix(s, "::annotation[") {
		return nil
	}

	// Find closing ]
	closeIdx := strings.Index(s[13:], "]")
	if closeIdx < 0 {
		return nil
	}

	text_ := s[13 : 13+closeIdx]
	rest := s[13+closeIdx+1:]

	// Find {explanation}
	explanation := ""
	consumed := 13 + closeIdx + 1
	if strings.HasPrefix(rest, "{") {
		endBrace := strings.Index(rest, "}")
		if endBrace > 1 {
			explanation = rest[1:endBrace]
			consumed += endBrace + 1
		}
	}

	node := &Annotation{Label: text_, Explanation: explanation}
	block.Advance(seg.Start + consumed - seg.Start)
	return node
}
