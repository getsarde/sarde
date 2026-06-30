package steps

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*steps\s*$`)
var closingRegex = regexp.MustCompile(`^:{3,}(?:/([\w-]+))?\s*$`)
var nestedOpenRegex = regexp.MustCompile(`^:{3,}\s*\w+`)

type stepsParser struct{}

// NewParser returns a new steps block parser.
func NewParser() parser.BlockParser {
	return &stepsParser{}
}

func (p *stepsParser) Trigger() []byte {
	return []byte{':'}
}

func (p *stepsParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))

	if !openingRegex.MatchString(lineStr) {
		return nil, parser.NoChildren
	}

	reader.Advance(len(line))

	node := &StepsBlock{}
	return node, parser.HasChildren
}

func (p *stepsParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
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
			if m[1] == "steps" {
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

func (p *stepsParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {
	deleteDepth(pc, node)

	stepsBlock := node.(*StepsBlock)

	// First pass: find the first H2 or H3 heading to determine the step delimiter level.
	var stepLevel int
	for c := stepsBlock.FirstChild(); c != nil; c = c.NextSibling() {
		if heading, ok := c.(*ast.Heading); ok && (heading.Level == 2 || heading.Level == 3) {
			stepLevel = heading.Level
			break
		}
	}

	if stepLevel == 0 {
		return // ordered-list mode: CSS handles numbering via .steps > ol > li
	}

	// Heading mode: split at the detected heading level into StepItem nodes.
	var items []*StepItem
	var currentItem *StepItem
	stepIndex := 0

	var children []ast.Node
	for c := stepsBlock.FirstChild(); c != nil; c = c.NextSibling() {
		children = append(children, c)
	}

	source := reader.Source()

	for _, child := range children {
		if heading, ok := child.(*ast.Heading); ok && heading.Level == stepLevel {
			stepIndex++
			currentItem = &StepItem{
				Title:        headingTitle(heading, source),
				Index:        stepIndex,
				HeadingLevel: stepLevel,
			}
			items = append(items, currentItem)
			continue
		}

		if currentItem == nil {
			// Content before first heading — create an implicit step
			stepIndex++
			currentItem = &StepItem{
				Title: "",
				Index: stepIndex,
			}
			items = append(items, currentItem)
		}

		// Move child under the current step item
		child.Parent().RemoveChild(child.Parent(), child)
		currentItem.AppendChild(currentItem, child)
	}

	// Remove all remaining direct children from stepsBlock
	for c := stepsBlock.FirstChild(); c != nil; {
		next := c.NextSibling()
		stepsBlock.RemoveChild(stepsBlock, c)
		c = next
	}

	// Append step items to stepsBlock
	for _, item := range items {
		stepsBlock.AppendChild(stepsBlock, item)
	}
}

func (p *stepsParser) CanInterruptParagraph() bool {
	return false
}

func (p *stepsParser) CanAcceptIndentedLine() bool {
	return false
}

// Context key for nested depth tracking
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

func headingTitle(heading *ast.Heading, source []byte) string {
	lines := heading.Lines()
	if lines.Len() == 0 {
		return ""
	}
	var buf []byte
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		buf = append(buf, seg.Value(source)...)
	}
	return strings.TrimSpace(string(buf))
}

func deleteDepth(pc parser.Context, node ast.Node) {
	v := pc.Get(contextKeyDepth)
	if v != nil {
		if m, ok := v.(map[ast.Node]int); ok {
			delete(m, node)
		}
	}
}
