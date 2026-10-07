package details

import (
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// :::details[Summary text]  or  :::details[Summary text](open)  or  :::details(open)[Summary text]  or  :::details
var openingRegex = regexp.MustCompile(`^:{3,}\s*details(?:\(open\))?(?:\[([^\]]*)\])?(?:\(open\))?\s*(open)?\s*$`)

type detailsParser struct{}

// NewParser returns a new details block parser.
func NewParser() parser.BlockParser {
	return &detailsParser{}
}

func (p *detailsParser) Trigger() []byte {
	return []byte{':'}
}

func (p *detailsParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))

	matches := openingRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}

	reader.AdvanceToEOL()

	summary := matches[1]
	if summary == "" {
		summary = "Details"
	}
	node := &DetailsBlock{
		Summary: summary,
		Open:    strings.Contains(lineStr, "(open)") || matches[2] == "open",
	}

	return node, parser.HasChildren
}

func (p *detailsParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return blockutil.ContinueContainer(pc, node, reader, "details")
}

func (p *detailsParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.Release(pc, node)
}

func (p *detailsParser) CanInterruptParagraph() bool {
	return false
}

func (p *detailsParser) CanAcceptIndentedLine() bool {
	return false
}
