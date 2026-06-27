package spoiler

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type spoilerParser struct{}

func NewInlineParser() parser.InlineParser { return &spoilerParser{} }

func (p *spoilerParser) Trigger() []byte { return []byte{'|'} }

func (p *spoilerParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, segment := block.PeekLine()
	if len(line) < 4 || line[0] != '|' || line[1] != '|' {
		return nil
	}

	// Find the closing ||
	for i := 2; i < len(line)-1; i++ {
		if line[i] == '|' && line[i+1] == '|' {
			if i == 2 {
				return nil // empty |||| not allowed
			}
			node := &SpoilerInline{}
			seg := text.NewSegment(segment.Start+2, segment.Start+i)
			node.AppendChild(node, ast.NewTextSegment(seg))
			block.Advance(i + 2)
			return node
		}
	}
	return nil
}
