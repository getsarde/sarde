package annotation

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var styleRegex = regexp.MustCompile(`^style="([^"]*)"(\s*)`)

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

	// Find (explanation)
	explanation := ""
	consumed := 13 + closeIdx + 1
	if strings.HasPrefix(rest, "(") {
		endParen := strings.Index(rest, ")")
		if endParen > 1 {
			explanation = rest[1:endParen]
			consumed += endParen + 1
		}
	}

	var style string
	if m := styleRegex.FindStringSubmatch(explanation); m != nil {
		if m[1] == "highlight" || m[1] == "plain" || m[1] == "underline" {
			style = m[1]
		}
		explanation = explanation[len(m[0]):]
	}

	node := &Annotation{Label: text_, Explanation: explanation, Style: style}
	block.Advance(seg.Start + consumed - seg.Start)
	return node
}
