package icon

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

const iconPrefix = ":icon["
const iconPrefixLen = 6

var attrRegex = regexp.MustCompile(`(\w[\w-]*)=(?:"([^"]*)"|'([^']*)'|(\S+))`)

type iconParser struct{}

// NewParser returns the inline parser for :icon[...] tokens.
func NewParser() parser.InlineParser { return &iconParser{} }

func (p *iconParser) Trigger() []byte { return []byte{':'} }

func (p *iconParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	s := string(line)

	if !strings.HasPrefix(s, iconPrefix) {
		return nil
	}

	closeIdx := strings.Index(s[iconPrefixLen:], "]")
	if closeIdx < 1 {
		return nil
	}

	inner := s[iconPrefixLen : iconPrefixLen+closeIdx]
	name, attrs := parseInner(inner)
	if name == "" {
		return nil
	}

	node := &Icon{Name: name, Attrs: attrs}
	block.Advance(iconPrefixLen + closeIdx + 1)
	return node
}

func parseInner(inner string) (string, map[string]string) {
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return "", nil
	}

	name := inner
	rest := ""
	if idx := strings.IndexAny(inner, " \t"); idx >= 0 {
		name = inner[:idx]
		rest = inner[idx+1:]
	}

	if rest == "" {
		return name, nil
	}

	attrs := make(map[string]string)
	for _, m := range attrRegex.FindAllStringSubmatch(rest, -1) {
		val := m[2]
		if val == "" {
			val = m[3]
		}
		if val == "" {
			val = m[4]
		}
		attrs[m[1]] = val
	}
	return name, attrs
}
