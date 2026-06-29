package copytext

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

const prefix = "::copy["

type copyTextParser struct{}

func NewParser() parser.InlineParser { return &copyTextParser{} }

func (p *copyTextParser) Trigger() []byte { return []byte{':'} }

func (p *copyTextParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	s := string(line)

	if !strings.HasPrefix(s, prefix) {
		return nil
	}

	closeIdx := strings.Index(s[len(prefix):], "]")
	if closeIdx < 1 {
		return nil
	}

	content := s[len(prefix) : len(prefix)+closeIdx]
	consumed := len(prefix) + closeIdx + 1

	node := &CopyText{Content: content}
	block.Advance(consumed)
	return node
}
