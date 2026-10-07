package card

import (
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/attrutil"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*card(?:\[([^\]]*)\])?(?:\((.+)\))?\s*$`)

type cardParser struct{}

func NewParser() parser.BlockParser   { return &cardParser{} }
func (p *cardParser) Trigger() []byte { return []byte{':'} }

func (p *cardParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	matches := openingRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}
	reader.AdvanceToEOL()

	title := matches[1]
	attrs := attrutil.Parse(matches[2])
	if title == "" {
		title = attrs["title"]
	}

	return &CardBlock{Title: title, Icon: attrs["icon"], Variant: attrs["variant"]}, parser.HasChildren
}

func (p *cardParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	return blockutil.ContinueContainer(pc, node, reader, "card")
}

func (p *cardParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.Release(pc, node)
}
func (p *cardParser) CanInterruptParagraph() bool { return false }
func (p *cardParser) CanAcceptIndentedLine() bool { return false }
