package tabs

import (
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/attrutil"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*tabs\s*$`)
var tabBoundaryRegex = regexp.MustCompile(`^==\s+(.+)`)
var tabAttrBlockRegex = regexp.MustCompile(`\(([^)]*)\)\s*$`)

func parseTabLabel(raw string) (label, icon string) {
	if m := tabAttrBlockRegex.FindStringSubmatchIndex(raw); m != nil {
		attrs := attrutil.Parse(raw[m[2]:m[3]])
		label = strings.TrimSpace(raw[:m[0]])
		icon = attrs["icon"]
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

	reader.AdvanceToEOL()
	return &TabsBlock{}, parser.HasChildren
}

func (p *tabsParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	// Tab boundary lines (== Tab Name) stay as paragraph text inside the
	// block and are split into tabs in Close.
	return blockutil.ContinueContainer(pc, node, reader, "tabs")
}

func (p *tabsParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.Release(pc, node)

	tabsBlock := node.(*TabsBlock)
	source := reader.Source()

	var items []*TabItem
	var currentItem *TabItem
	tabIndex := 0

	var children []ast.Node
	for c := tabsBlock.FirstChild(); c != nil; c = c.NextSibling() {
		children = append(children, c)
	}

	newTab := func(label, icon string) {
		currentItem = &TabItem{
			Label: label,
			Icon:  icon,
			Index: tabIndex,
		}
		items = append(items, currentItem)
		tabIndex++
	}

	for _, child := range children {
		// A paragraph may hold several "== Label" markers, because Markdown
		// only starts a new paragraph at a blank line. Split it so compact
		// markup keeps every tab and every line of body text.
		if para, ok := child.(*ast.Paragraph); ok {
			if segments := blockutil.SplitParagraphAtMarkers(para, source, tabBoundaryRegex); segments != nil {
				for _, seg := range segments {
					if seg.IsMarker {
						newTab(parseTabLabel(seg.Marker))
					} else if currentItem == nil {
						newTab("Tab 1", "")
					}
					if seg.Body != nil {
						currentItem.AppendChild(currentItem, seg.Body)
					}
				}
				continue
			}
		}

		if currentItem == nil {
			newTab("Tab 1", "")
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
