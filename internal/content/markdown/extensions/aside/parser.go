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
var nestedOpenRegex = regexp.MustCompile(`^:{3,}\s*\w+`)
var closingFenceRegex = regexp.MustCompile(`^:{3,}(?:/([\w-]+))?\s*$`)

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

	reader.Advance(len(line))

	node := &AsideBlock{
		AsideType: asideType,
		Title:     title,
		Icon:      icon,
	}

	return node, parser.HasChildren
}

// Continue is called to check if the block continues on the current line.
func (p *asideParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	aside := node.(*AsideBlock)

	line, _ := reader.PeekLine()
	lineStr := string(line)
	trimmed := strings.TrimSpace(lineStr)

	// Get or initialize nested depth from context
	depth := blockutil.GetDepth(pc, aside)

	// Check for ::: fences
	if strings.HasPrefix(trimmed, ":::") {
		// Check if it's an opening fence (:::word)
		if nestedOpenRegex.MatchString(trimmed) && !closingFenceRegex.MatchString(trimmed) {
			blockutil.SetDepth(pc, aside, depth+1)
			return parser.Continue | parser.HasChildren
		}

		// Check if it's a closing fence
		if closingFenceRegex.MatchString(trimmed) {
			if depth > 0 {
				blockutil.SetDepth(pc, aside, depth-1)
				return parser.Continue | parser.HasChildren
			}

			// At depth 0 — check for named closing
			matches := closingFenceRegex.FindStringSubmatch(trimmed)
			closingName := ""
			if len(matches) > 1 {
				closingName = matches[1]
			}

			if closingName != "" && closingName != aside.AsideType {
				// Mismatched named closing — don't close
				return parser.Continue | parser.HasChildren
			}

			if closingName == "" && blockutil.HasInnerOpenBlocks(pc, node) {
				return parser.Continue | parser.HasChildren
			}

			// Valid closing
			reader.AdvanceToEOL()
			return parser.Close
		}
	}

	return parser.Continue | parser.HasChildren
}

// Close is called when the block is closed.
func (p *asideParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	// Clean up context
	aside := node.(*AsideBlock)
	blockutil.DeleteDepth(pc, aside)
}

// CanInterruptParagraph returns false.
func (p *asideParser) CanInterruptParagraph() bool {
	return false
}

// CanAcceptIndentedLine returns false.
func (p *asideParser) CanAcceptIndentedLine() bool {
	return false
}

// Context key for nested depth tracking
