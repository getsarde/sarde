package accordion

import (
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/attrutil"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*accordion(?:\((.+)\))?\s*$`)

type accordionParser struct{}

func NewParser() parser.BlockParser        { return &accordionParser{} }
func (p *accordionParser) Trigger() []byte { return []byte{':'} }

func (p *accordionParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	matches := openingRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}
	reader.AdvanceToEOL()

	independent := false
	if matches[1] != "" {
		independent = attrutil.Has(attrutil.Parse(matches[1]), "independent")
	}

	return &AccordionBlock{Independent: independent}, parser.HasChildren
}

func (p *accordionParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return blockutil.ContinueContainer(pc, node, reader, "accordion")
}

func (p *accordionParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.Release(pc, node)
}
func (p *accordionParser) CanInterruptParagraph() bool { return false }
func (p *accordionParser) CanAcceptIndentedLine() bool { return false }
