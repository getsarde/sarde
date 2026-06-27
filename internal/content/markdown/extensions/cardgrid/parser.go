package cardgrid

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*card-grid\s*$`)
var closingRegex = regexp.MustCompile(`^:{3,}(?:/([\w-]+))?\s*$`)
var nestedOpenRegex = regexp.MustCompile(`^:{3,}\s*\w+`)

type cardGridParser struct{}

func NewParser() parser.BlockParser { return &cardGridParser{} }
func (p *cardGridParser) Trigger() []byte { return []byte{':'} }

func (p *cardGridParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	if !openingRegex.MatchString(lineStr) {
		return nil, parser.NoChildren
	}
	reader.Advance(len(line))
	return &CardGridBlock{}, parser.HasChildren
}

func (p *cardGridParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	trimmed := strings.TrimSpace(string(line))
	depth := getDepth(pc, node)
	if strings.HasPrefix(trimmed, ":::") {
		if nestedOpenRegex.MatchString(trimmed) && !closingRegex.MatchString(trimmed) {
			setDepth(pc, node, depth+1)
			return parser.Continue | parser.HasChildren
		}
		if m := closingRegex.FindStringSubmatch(trimmed); m != nil {
			if depth > 0 {
				setDepth(pc, node, depth-1)
				return parser.Continue | parser.HasChildren
			}
			if m[1] == "card-grid" {
				reader.AdvanceToEOL()
				return parser.Close
			}
			if m[1] == "" && !hasInnerOpenBlocks(pc, node) {
				reader.AdvanceToEOL()
				return parser.Close
			}
		}
	}
	return parser.Continue | parser.HasChildren
}

func (p *cardGridParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	deleteDepth(pc, node)
}
func (p *cardGridParser) CanInterruptParagraph() bool { return false }
func (p *cardGridParser) CanAcceptIndentedLine() bool { return false }

var contextKeyDepth = parser.NewContextKey()

func getDepth(pc parser.Context, node ast.Node) int {
	if v := pc.Get(contextKeyDepth); v != nil {
		if m, ok := v.(map[ast.Node]int); ok {
			return m[node]
		}
	}
	return 0
}

func setDepth(pc parser.Context, node ast.Node, depth int) {
	v := pc.Get(contextKeyDepth)
	var m map[ast.Node]int
	if v == nil {
		m = make(map[ast.Node]int)
	} else {
		m = v.(map[ast.Node]int)
	}
	m[node] = depth
	pc.Set(contextKeyDepth, m)
}

func hasInnerOpenBlocks(pc parser.Context, node ast.Node) bool {
	blocks := pc.OpenedBlocks()
	for i, b := range blocks {
		if b.Node == node {
			for j := i + 1; j < len(blocks); j++ {
				for _, t := range blocks[j].Parser.Trigger() {
					if t == ':' {
						return true
					}
				}
			}
			return false
		}
	}
	return false
}

func deleteDepth(pc parser.Context, node ast.Node) {
	if v := pc.Get(contextKeyDepth); v != nil {
		if m, ok := v.(map[ast.Node]int); ok {
			delete(m, node)
		}
	}
}
