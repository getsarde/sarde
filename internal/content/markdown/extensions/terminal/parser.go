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
var closingRegex = regexp.MustCompile(`^:{3,}(?:/([\w-]+))?\s*$`)
var nestedOpenRegex = regexp.MustCompile(`^:{3,}\s*\w+`)

type terminalParser struct{}

func NewParser() parser.BlockParser       { return &terminalParser{} }
func (p *terminalParser) Trigger() []byte { return []byte{':'} }

func (p *terminalParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	if !openingRegex.MatchString(lineStr) {
		return nil, parser.NoChildren
	}
	reader.Advance(len(line))
	return &TerminalBlock{}, parser.HasChildren
}

func (p *terminalParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	trimmed := strings.TrimSpace(string(line))

	depth := blockutil.GetDepth(pc, node)

	if strings.HasPrefix(trimmed, ":::") {
		if nestedOpenRegex.MatchString(trimmed) && !closingRegex.MatchString(trimmed) {
			blockutil.SetDepth(pc, node, depth+1)
			return parser.Continue | parser.HasChildren
		}
		if m := closingRegex.FindStringSubmatch(trimmed); m != nil {
			if depth > 0 {
				blockutil.SetDepth(pc, node, depth-1)
				return parser.Continue | parser.HasChildren
			}
			if m[1] == "terminal" {
				reader.AdvanceToEOL()
				return parser.Close
			}
			if m[1] == "" && !blockutil.HasInnerOpenBlocks(pc, node) {
				reader.AdvanceToEOL()
				return parser.Close
			}
		}
	}

	return parser.Continue | parser.HasChildren
}

func (p *terminalParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.DeleteDepth(pc, node)
}
func (p *terminalParser) CanInterruptParagraph() bool { return false }
func (p *terminalParser) CanAcceptIndentedLine() bool { return false }
