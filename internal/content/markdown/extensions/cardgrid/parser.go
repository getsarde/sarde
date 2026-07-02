package cardgrid

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/attrutil"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*card-grid(?:\((.+)\))?\s*$`)

var closingRegex = regexp.MustCompile(`^:{3,}(?:/([\w-]+))?\s*$`)
var nestedOpenRegex = regexp.MustCompile(`^:{3,}\s*\w+`)

type cardGridParser struct{}

func NewParser() parser.BlockParser       { return &cardGridParser{} }
func (p *cardGridParser) Trigger() []byte { return []byte{':'} }

func (p *cardGridParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	matches := openingRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}
	reader.Advance(len(line))

	var cols int
	var stagger bool
	if matches[1] != "" {
		attrs := attrutil.Parse(matches[1])
		if v, err := strconv.Atoi(attrs["cols"]); err == nil && v >= 2 && v <= 4 {
			cols = v
		}
		stagger = attrutil.Has(attrs, "stagger")
	}

	return &CardGridBlock{Cols: cols, Stagger: stagger}, parser.HasChildren
}

func (p *cardGridParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
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
			if m[1] == "card-grid" {
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

func (p *cardGridParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.DeleteDepth(pc, node)
}
func (p *cardGridParser) CanInterruptParagraph() bool { return false }
func (p *cardGridParser) CanAcceptIndentedLine() bool { return false }
