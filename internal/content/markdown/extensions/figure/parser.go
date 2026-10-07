package figure

import (
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*figure\[([^\]]*)\]\s*$`)

type figureParser struct{}

func NewParser() parser.BlockParser     { return &figureParser{} }
func (p *figureParser) Trigger() []byte { return []byte{':'} }

func (p *figureParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	matches := openingRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}
	reader.AdvanceToEOL()
	return &FigureBlock{Caption: matches[1]}, parser.HasChildren
}

func (p *figureParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return blockutil.ContinueContainer(pc, node, reader, "figure")
}

func (p *figureParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.Release(pc, node)
}
func (p *figureParser) CanInterruptParagraph() bool { return false }
func (p *figureParser) CanAcceptIndentedLine() bool { return false }
