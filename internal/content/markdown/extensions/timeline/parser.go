package timeline

import (
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/blockutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*timeline\s*$`)
var closingRegex = regexp.MustCompile(`^:{3,}(?:/([\w-]+))?\s*$`)
var nestedOpenRegex = regexp.MustCompile(`^:{3,}\s*\w+`)
var entryBoundaryRegex = regexp.MustCompile(`^==\s+(.+)`)

type timelineParser struct{}

func NewParser() parser.BlockParser       { return &timelineParser{} }
func (p *timelineParser) Trigger() []byte { return []byte{':'} }

func (p *timelineParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))

	if !openingRegex.MatchString(lineStr) {
		return nil, parser.NoChildren
	}

	reader.Advance(len(line))
	return &TimelineBlock{}, parser.HasChildren
}

func (p *timelineParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
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
			if m[1] == "timeline" {
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

func (p *timelineParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	blockutil.DeleteDepth(pc, node)

	// Post-process: split children at == markers into TimelineItem nodes.
	block := node.(*TimelineBlock)
	source := reader.Source()
	var items []*TimelineItem
	var currentItem *TimelineItem

	// Collect all children first to avoid mutation during iteration.
	var children []ast.Node
	for c := block.FirstChild(); c != nil; c = c.NextSibling() {
		children = append(children, c)
	}

	for _, child := range children {
		// Check for "== Title" paragraph boundary
		if para, ok := child.(*ast.Paragraph); ok {
			text := strings.TrimSpace(string(para.Text(source)))
			if matches := entryBoundaryRegex.FindStringSubmatch(text); matches != nil {
				currentItem = &TimelineItem{
					Title: strings.TrimSpace(matches[1]),
				}
				items = append(items, currentItem)
				// If paragraph has content after the == line, keep it as body text
				idx := strings.Index(text, "\n")
				if idx >= 0 {
					remaining := strings.TrimSpace(text[idx+1:])
					if remaining != "" {
						currentItem.BodyText = remaining
					}
				}
				continue
			}
		}

		// Also support ### heading syntax
		if heading, ok := child.(*ast.Heading); ok && heading.Level == 3 {
			currentItem = &TimelineItem{
				Title: string(heading.Text(source)),
			}
			items = append(items, currentItem)
			continue
		}

		if currentItem == nil {
			currentItem = &TimelineItem{Title: ""}
			items = append(items, currentItem)
		}

		child.Parent().RemoveChild(child.Parent(), child)
		currentItem.AppendChild(currentItem, child)
	}

	// Remove all remaining direct children from the block.
	for c := block.FirstChild(); c != nil; {
		next := c.NextSibling()
		block.RemoveChild(block, c)
		c = next
	}

	// Append timeline items to the block.
	for _, item := range items {
		block.AppendChild(block, item)
	}
}

func (p *timelineParser) CanInterruptParagraph() bool { return false }
func (p *timelineParser) CanAcceptIndentedLine() bool { return false }

// Context key for nested depth tracking — each package gets its own key.
