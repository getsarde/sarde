package badgegroup

import (
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*badge-group\s*$`)

type badgeGroupParser struct{}

func NewParser() parser.BlockParser         { return &badgeGroupParser{} }
func (p *badgeGroupParser) Trigger() []byte { return []byte{':'} }

func (p *badgeGroupParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	if !openingRegex.MatchString(lineStr) {
		return nil, parser.NoChildren
	}
	reader.AdvanceToEOL()
	return &BadgeGroupBlock{}, parser.HasChildren
}

func (p *badgeGroupParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return blockutil.ContinueContainer(pc, node, reader, "badge-group")
}

func (p *badgeGroupParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.Release(pc, node)
}
func (p *badgeGroupParser) CanInterruptParagraph() bool { return false }
func (p *badgeGroupParser) CanAcceptIndentedLine() bool { return false }
