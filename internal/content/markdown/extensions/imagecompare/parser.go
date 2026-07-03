package imagecompare

import (
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*image-compare(?:\[([^\]]+)\])?\s*$`)
var closingRegex = regexp.MustCompile(`^:{3,}(?:/([\w-]+))?\s*$`)
var nestedOpenRegex = regexp.MustCompile(`^:{3,}\s*\w+`)

type imageCompareParser struct{}

func NewParser() parser.BlockParser           { return &imageCompareParser{} }
func (p *imageCompareParser) Trigger() []byte { return []byte{':'} }

func (p *imageCompareParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	matches := openingRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}
	reader.Advance(len(line))
	label := ""
	if len(matches) > 1 {
		label = matches[1]
	}
	return &ImageCompareBlock{Label: label}, parser.HasChildren
}

func (p *imageCompareParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
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
			if m[1] == "image-compare" {
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

func (p *imageCompareParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	// Image extraction happens in the renderer, not here: Close runs during
	// block parsing, before the inline pass creates the ast.Image nodes, so a
	// walk for images at this point would always find nothing.
	blockutil.DeleteDepth(pc, node)
}

func (p *imageCompareParser) CanInterruptParagraph() bool { return false }
func (p *imageCompareParser) CanAcceptIndentedLine() bool { return false }
