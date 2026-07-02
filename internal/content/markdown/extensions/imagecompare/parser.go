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
	ic := node.(*ImageCompareBlock)
	blockutil.DeleteDepth(pc, node)

	// Walk children to find first two image nodes
	var walk func(n ast.Node)
	walk = func(n ast.Node) {
		if n.Kind() == ast.KindImage {
			img := n.(*ast.Image)
			alt := extractText(n, reader.Source())
			if ic.BeforeSrc == "" {
				ic.BeforeSrc = string(img.Destination)
				ic.BeforeAlt = alt
			} else if ic.AfterSrc == "" {
				ic.AfterSrc = string(img.Destination)
				ic.AfterAlt = alt
			}
		}
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			walk(c)
		}
	}
	walk(node)
}

func extractText(n ast.Node, source []byte) string {
	var sb strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if c.Kind() == ast.KindText {
			t := c.(*ast.Text)
			sb.Write(t.Segment.Value(source))
		}
	}
	return sb.String()
}

func (p *imageCompareParser) CanInterruptParagraph() bool { return false }
func (p *imageCompareParser) CanAcceptIndentedLine() bool { return false }
