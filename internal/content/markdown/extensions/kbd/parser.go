package kbd

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var (
	sizeAttrRe = regexp.MustCompile(`size\s*=\s*"([^"]*)"`)
	wideFlagRe = regexp.MustCompile(`\bwide\b`)
)

type kbdParser struct{}

func NewParser() parser.InlineParser { return &kbdParser{} }

func (p *kbdParser) Trigger() []byte { return []byte{':'} }

func (p *kbdParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	line, _ := block.PeekLine()
	s := string(line)

	if !strings.HasPrefix(s, "::kbd[") {
		return nil
	}

	closeIdx := strings.Index(s[6:], "]")
	if closeIdx < 1 {
		return nil
	}

	key := s[6 : 6+closeIdx]
	consumed := 6 + closeIdx + 1

	var size string
	var wide bool

	rest := s[consumed:]
	if strings.HasPrefix(rest, "(") {
		endParen := strings.Index(rest, ")")
		if endParen >= 1 {
			attrBlock := rest[1:endParen]
			if m := sizeAttrRe.FindStringSubmatch(attrBlock); m != nil {
				if m[1] == "sm" || m[1] == "lg" {
					size = m[1]
				}
			}
			wide = wideFlagRe.MatchString(attrBlock)
			consumed += endParen + 1
		}
	}

	node := &Kbd{Keys: key, Size: size, Wide: wide}
	block.Advance(consumed)
	return node
}
