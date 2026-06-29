package linkcard

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openingRegex = regexp.MustCompile(`^:{3,}\s*link-card(?:\[([^\]]*)\])?(?:\{([^}]*)\})?\s*$`)
var closingRegex = regexp.MustCompile(`^:{3,}(?:/link-card)?\s*$`)
var attrRegex = regexp.MustCompile(`([\w-]+)="([^"]*)"`)

type linkCardParser struct{}

func NewParser() parser.BlockParser { return &linkCardParser{} }
func (p *linkCardParser) Trigger() []byte { return []byte{':'} }

func (p *linkCardParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	matches := openingRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}
	// Do not advance — the framework calls AdvanceLine() after Open.

	title := matches[1]
	attrs := parseAttrs(matches[2])
	if t, ok := attrs["title"]; ok && title == "" {
		title = t
	}

	var newTab *bool
	if v, ok := attrs["new-tab"]; ok {
		b := v == "true"
		newTab = &b
	}

	return &LinkCardBlock{
		Title:       title,
		Href:        attrs["href"],
		Description: attrs["description"],
		Icon:        attrs["icon"],
		Image:       attrs["image"],
		NewTab:      newTab,
	}, parser.NoChildren
}

func (p *linkCardParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	trimmed := strings.TrimSpace(string(line))

	if closingRegex.MatchString(trimmed) {
		reader.AdvanceToEOL()
		return parser.Close
	}

	// Capture body text as description (overrides attribute if present).
	if trimmed != "" {
		lc := node.(*LinkCardBlock)
		if lc.Description == "" {
			lc.Description = trimmed
		} else {
			lc.Description += " " + trimmed
		}
	}

	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

func (p *linkCardParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}
func (p *linkCardParser) CanInterruptParagraph() bool                                 { return false }
func (p *linkCardParser) CanAcceptIndentedLine() bool                                 { return false }

func parseAttrs(s string) map[string]string {
	result := make(map[string]string)
	for _, m := range attrRegex.FindAllStringSubmatch(s, -1) {
		result[m[1]] = m[2]
	}
	return result
}
