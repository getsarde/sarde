package terminal

import (
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`(?i)^:{3,}\s*terminal\s*$`)

type terminalParser struct{}

func NewParser() parser.BlockParser       { return &terminalParser{} }
func (p *terminalParser) Trigger() []byte { return []byte{':'} }

func (p *terminalParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	if !openingRegex.MatchString(lineStr) {
		return nil, parser.NoChildren
	}
	reader.AdvanceToEOL()
	return &TerminalBlock{}, parser.HasChildren
}

func (p *terminalParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return blockutil.ContinueContainer(pc, node, reader, "terminal")
}

func (p *terminalParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.Release(pc, node)
}
func (p *terminalParser) CanInterruptParagraph() bool { return false }
func (p *terminalParser) CanAcceptIndentedLine() bool { return false }
