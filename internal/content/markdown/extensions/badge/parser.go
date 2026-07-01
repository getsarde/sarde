package badge

import (
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/attrutil"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var openRegex = regexp.MustCompile(`^:{3,}\s*badge(?:\(([^)]+)\))?\s*$`)
var closingRegex = regexp.MustCompile(`^:{3,}(?:/badge)?\s*$`)

type badgeParser struct{}

func NewParser() parser.BlockParser { return &badgeParser{} }
func (p *badgeParser) Trigger() []byte { return []byte{':'} }

func (p *badgeParser) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.PeekLine()
	lineStr := strings.TrimSpace(string(line))
	matches := openRegex.FindStringSubmatch(lineStr)
	if matches == nil {
		return nil, parser.NoChildren
	}

	badgeType := "default"
	icon := ""
	style := ""
	size := ""
	noIcon := false
	if matches[1] != "" {
		raw := matches[1]
		attrs := attrutil.Parse(raw)
		if v, ok := attrs["type"]; ok {
			badgeType = v
		} else if !strings.Contains(raw, "=") {
			badgeType = strings.TrimSpace(raw)
		}
		icon = attrs["icon"]
		if attrs["style"] == "outline" {
			style = "outline"
		}
		if s := attrs["size"]; s == "sm" || s == "lg" {
			size = s
		}
		noIcon = attrutil.Bool(attrs, "no-icon")
	}

	return &Badge{BadgeType: badgeType, Icon: icon, Style: style, Size: size, NoIcon: noIcon}, parser.NoChildren
}

func (p *badgeParser) Continue(node ast.Node, reader text.Reader, pc parser.Context) parser.State {
	line, _ := reader.PeekLine()
	trimmed := strings.TrimSpace(string(line))

	if closingRegex.MatchString(trimmed) {
		reader.AdvanceToEOL()
		return parser.Close
	}

	// Capture body text as badge content.
	if trimmed != "" {
		b := node.(*Badge)
		if b.Content == "" {
			b.Content = trimmed
		} else {
			b.Content += " " + trimmed
		}
	}

	reader.AdvanceToEOL()
	return parser.Continue | parser.NoChildren
}

func (p *badgeParser) Close(node ast.Node, reader text.Reader, pc parser.Context) {}
func (p *badgeParser) CanInterruptParagraph() bool                                 { return false }
func (p *badgeParser) CanAcceptIndentedLine() bool                                 { return false }

