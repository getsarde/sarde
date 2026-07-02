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
var closingRegex = regexp.MustCompile(`^:{3,}(?:/([\w-]+))?\s*$`)
var nestedOpenRegex = regexp.MustCompile(`^:{3,}\s*\w+`)

type badgeGroupParser struct{}

func NewParser() parser.BlockParser         { return &badgeGroupParser{} }
func (p *badgeGroupParser) Trigger() []byte { return []byte{':'} }

func (p *badgeGroupParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	if !openingRegex.MatchString(lineStr) {
		return nil, parser.NoChildren
	}
	reader.Advance(len(line))
	return &BadgeGroupBlock{}, parser.HasChildren
}

func (p *badgeGroupParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
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
			if m[1] == "badge-group" {
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

func (p *badgeGroupParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.DeleteDepth(pc, node)
}
func (p *badgeGroupParser) CanInterruptParagraph() bool { return false }
func (p *badgeGroupParser) CanAcceptIndentedLine() bool { return false }
