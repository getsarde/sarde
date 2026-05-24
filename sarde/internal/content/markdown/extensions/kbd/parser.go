package kbd

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type kbdParser struct{}

func NewParser() parser.InlineParser { return &kbdParser{} }

func (p *kbdParser) Trigger() []byte { return []byte{':'} }

func (p *kbdParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, seg := block.PeekLine()
	s := string(line)

	if !strings.HasPrefix(s, "::kbd[") {
		return nil
	}

	closeIdx := strings.Index(s[6:], "]")
	if closeIdx < 1 {
		return nil
	}

	key := s[6 : 6+closeIdx]
	node := &Kbd{Keys: key}
	block.Advance(seg.Start + 6 + closeIdx + 1 - seg.Start)
	return node
}
