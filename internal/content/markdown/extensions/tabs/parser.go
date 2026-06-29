package tabs

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*tabs\s*$`)
var closingRegex = regexp.MustCompile(`^:{3,}(?:/([\w-]+))?\s*$`)
var nestedOpenRegex = regexp.MustCompile(`^:{3,}\s*\w+`)
var tabBoundaryRegex = regexp.MustCompile(`^==\s+(.+)`)
var tabAttrBlockRegex = regexp.MustCompile(`\{([^}]*)\}\s*$`)
var attrRegex = regexp.MustCompile(`(\w+)="([^"]*)"`)

func parseTabLabel(raw string) (label, icon string) {
	if m := tabAttrBlockRegex.FindStringSubmatchIndex(raw); m != nil {
		attrStr := raw[m[2]:m[3]]
		label = strings.TrimSpace(raw[:m[0]])
		for _, attr := range attrRegex.FindAllStringSubmatch(attrStr, -1) {
			if attr[1] == "icon" {
				icon = attr[2]
			}
		}
		return
	}
	return raw, ""
}

type tabsParser struct{}

// NewParser returns a new tabs block parser.
func NewParser() parser.BlockParser {
	return &tabsParser{}
}

func (p *tabsParser) Trigger() []byte {
	return []byte{':'}
}

func (p *tabsParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))

	if !openingRegex.MatchString(lineStr) {
		return nil, parser.NoChildren
	}

	reader.Advance(len(line))
	return &TabsBlock{}, parser.HasChildren
}

func (p *tabsParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
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
			if m[1] == "tabs" {
				reader.AdvanceToEOL()
				return parser.Close
			}
			if m[1] == "" && !hasInnerOpenBlocks(pc, node) {
				reader.AdvanceToEOL()
				return parser.Close
			}
		}
	}

	// Tab boundary lines (== Tab Name) are consumed here at depth 0
	// They become paragraph text inside the block, handled in Close
	return parser.Continue | parser.HasChildren
}

func (p *tabsParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	deleteDepth(pc, node)

	tabsBlock := node.(*TabsBlock)
	source := reader.Source()

	var items []*TabItem
	var currentItem *TabItem
	tabIndex := 0

	var children []ast.Node
	for c := tabsBlock.FirstChild(); c != nil; c = c.NextSibling() {
		children = append(children, c)
	}

	for _, child := range children {
		// Check if this is a paragraph whose text starts with "== "
		if para, ok := child.(*ast.Paragraph); ok {
			text := strings.TrimSpace(string(para.Text(source)))
			if matches := tabBoundaryRegex.FindStringSubmatch(text); matches != nil {
				label, icon := parseTabLabel(strings.TrimSpace(matches[1]))
				currentItem = &TabItem{
					Label: label,
					Icon:  icon,
					Index: tabIndex,
				}
				items = append(items, currentItem)
				tabIndex++
				continue
			}
		}

		if currentItem == nil {
			currentItem = &TabItem{
				Label: "Tab 1",
				Index: 0,
			}
			items = append(items, currentItem)
			tabIndex++
		}

		child.Parent().RemoveChild(child.Parent(), child)
		currentItem.AppendChild(currentItem, child)
	}

	// Clear and rebuild
	for c := tabsBlock.FirstChild(); c != nil; {
		next := c.NextSibling()
		tabsBlock.RemoveChild(tabsBlock, c)
		c = next
	}

	for _, item := range items {
		tabsBlock.AppendChild(tabsBlock, item)
	}
}

func (p *tabsParser) CanInterruptParagraph() bool {
	return false
}

func (p *tabsParser) CanAcceptIndentedLine() bool {
	return false
}

var contextKeyDepth = parser.NewContextKey()

func getDepth(pc parser.Context, node ast.Node) int {
	v := pc.Get(contextKeyDepth)
	if v == nil {
		return 0
	}
	if m, ok := v.(map[ast.Node]int); ok {
		return m[node]
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
	v := pc.Get(contextKeyDepth)
	if v != nil {
		if m, ok := v.(map[ast.Node]int); ok {
			delete(m, node)
		}
	}
}
