package linkbutton

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// matches[1] = colon sequence, matches[2] = label, matches[3] = attrs
var openingRegex = regexp.MustCompile(`^(:{3,})\s*link-button(?:\[([^\]]*)\])?(?:\{([^}]*)\})?\s*$`)
var attrRegex = regexp.MustCompile(`(\w+)="([^"]*)"`)

// linkButtonParser is stateless between blocks; hasLabel is reset in Open on each new block.
type linkButtonParser struct {
	hasLabel bool
}

func NewParser() parser.BlockParser         { return &linkButtonParser{} }
func (p *linkButtonParser) Trigger() []byte { return []byte{':'} }

func (p *linkButtonParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	matches := openingRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}

	colonCount := len(matches[1])
	label := matches[2]
	attrs := parseAttrs(matches[3])

	if attrs["href"] == "" {
		return nil, parser.NoChildren
	}

	variant := attrs["variant"]
	if variant == "" || !isValidVariant(variant) {
		variant = "primary"
	}

	iconPlacement := attrs["iconPlacement"]
	if iconPlacement == "" || !isValidIconPlacement(iconPlacement) {
		iconPlacement = "end"
	}

	content := label
	if content == "" {
		content = attrs["label"]
	}

	p.hasLabel = content != ""

	return &LinkButtonBlock{
		Href:          attrs["href"],
		Variant:       variant,
		Icon:          attrs["icon"],
		IconPlacement: iconPlacement,
		Content:       content,
		ColonCount:    colonCount,
	}, parser.NoChildren
}

func (p *linkButtonParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	trimmed := strings.TrimSpace(string(line))
	lb := node.(*LinkButtonBlock)

	if isClosingFence(trimmed, lb.ColonCount) {
		reader.AdvanceToEOL()
		return parser.Close
	}

	// Only use body content when no explicit label was given in the opening tag.
	if !p.hasLabel && trimmed != "" {
		if lb.Content != "" {
			lb.Content += " "
		}
		lb.Content += trimmed
	}

	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

func (p *linkButtonParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}

func (p *linkButtonParser) CanInterruptParagraph() bool { return false }
func (p *linkButtonParser) CanAcceptIndentedLine() bool { return false }

// isClosingFence returns true when line is a valid closing fence: at least minColons
// colons followed by nothing or "/link-button".
func isClosingFence(line string, minColons int) bool {
	count := 0
	for count < len(line) && line[count] == ':' {
		count++
	}
	if count < minColons {
		return false
	}
	rest := strings.TrimSpace(line[count:])
	return rest == "" || rest == "/link-button"
}

func parseAttrs(s string) map[string]string {
	result := make(map[string]string)
	if s == "" {
		return result
	}
	for _, m := range attrRegex.FindAllStringSubmatch(s, -1) {
		result[m[1]] = m[2]
	}
	return result
}

func isValidVariant(v string) bool {
	return v == "primary" || v == "secondary" || v == "minimal"
}

func isValidIconPlacement(p string) bool {
	return p == "start" || p == "end"
}
