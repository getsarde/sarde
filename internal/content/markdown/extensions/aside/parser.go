package aside

import (
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingFenceRegex = regexp.MustCompile(`^:{3,}\s*([\w-]+)(?:\[([^\]]+)\])?(?:\s+icon=([\w-]+))?`)

// asideParser is a goldmark block parser for aside blocks.
type asideParser struct{}

// NewParser returns a new aside block parser.
func NewParser() parser.BlockParser {
	return &asideParser{}
}

// Trigger returns the trigger characters for this parser.
func (p *asideParser) Trigger() []byte {
	return []byte{':'}
}

// Open is called when the parser encounters a potential block start.
func (p *asideParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))

	matches := openingFenceRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}

	asideType := strings.ToLower(matches[1])
	if !ValidTypes[asideType] {
		return nil, parser.NoChildren
	}

	title := ""
	if len(matches) > 2 {
		title = matches[2]
	}
	icon := ""
	if len(matches) > 3 {
		icon = matches[3]
	}

	reader.AdvanceToEOL()

	node := &AsideBlock{
		AsideType: asideType,
		Title:     title,
		Icon:      icon,
	}

	return node, parser.HasChildren
}

// Continue is called to check if the block continues on the current line.
func (p *asideParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	// The aside type is the block name, so ":::/note" closes a note;
	// fence.Closes also accepts ":::/aside" for any type.
	return blockutil.ContinueContainer(pc, node, reader, node.(*AsideBlock).AsideType)
}

// Close is called when the block is closed.
func (p *asideParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.Release(pc, node)
}

// CanInterruptParagraph returns false.
func (p *asideParser) CanInterruptParagraph() bool {
	return false
}

// CanAcceptIndentedLine returns false.
func (p *asideParser) CanAcceptIndentedLine() bool {
	return false
}
