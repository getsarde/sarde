package linkbuttongroup

import (
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*link-button-group\s*$`)

type linkButtonGroupParser struct{}

func NewParser() parser.BlockParser              { return &linkButtonGroupParser{} }
func (p *linkButtonGroupParser) Trigger() []byte { return []byte{':'} }

func (p *linkButtonGroupParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	if !openingRegex.MatchString(lineStr) {
		return nil, parser.NoChildren
	}
	reader.AdvanceToEOL()
	return &LinkButtonGroupBlock{}, parser.HasChildren
}

func (p *linkButtonGroupParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return blockutil.ContinueContainer(pc, node, reader, "link-button-group")
}

func (p *linkButtonGroupParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.Release(pc, node)
}
func (p *linkButtonGroupParser) CanInterruptParagraph() bool { return false }
func (p *linkButtonGroupParser) CanAcceptIndentedLine() bool { return false }
