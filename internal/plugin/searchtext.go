package plugin

import (
	"strings"

	nethtml "golang.org/x/net/html"
)

// searchSkipTags are elements whose entire subtree is never searchable text.
var searchSkipTags = map[string]bool{
	"script":   true,
	"style":    true,
	"noscript": true,
	"svg":      true,
	"iframe":   true,
	"template": true,
}

// searchSkipClasses mark subtrees to exclude from search text by exact class
// token: "not-content" (Mermaid diagrams and other non-prose regions),
// "sarde-math" (raw LaTeX source rendered client-side), and Kazari's
// line-number gutter markup ("kz-gutter", "kz-ln"; note "kz-line" is the code
// text itself and must stay searchable, which is why matching is per token,
// not substring).
var searchSkipClasses = map[string]bool{
	"not-content": true,
	"sarde-math":  true,
	"kz-gutter":   true,
	"kz-ln":       true,
}

// ExtractSearchText returns the searchable plain text of rendered page HTML.
// Unlike StripHTML it parses the markup, so it can drop subtrees that render
// as UI rather than prose (diagram DSLs, raw math source, code line-number
// gutters) and it decodes entities exactly once. Whitespace is collapsed and
// a space is inserted at element boundaries, matching StripHTML's shape. On
// a parse failure it falls back to StripHTML.
func ExtractSearchText(s string) string {
	doc, err := nethtml.Parse(strings.NewReader(s))
	if err != nil {
		return StripHTML(s)
	}

	var b strings.Builder
	b.Grow(len(s) / 2)
	lastSpace := true

	writeText := func(text string) {
		for _, r := range text {
			if r <= ' ' {
				if !lastSpace {
					b.WriteByte(' ')
					lastSpace = true
				}
				continue
			}
			b.WriteRune(r)
			lastSpace = false
		}
	}

	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		switch n.Type {
		case nethtml.TextNode:
			writeText(n.Data)
			return
		case nethtml.ElementNode:
			if searchSkipTags[n.Data] || hasSearchSkipClass(n) {
				return
			}
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == nethtml.ElementNode && !lastSpace {
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	walk(doc)
	return strings.TrimSpace(b.String())
}

func hasSearchSkipClass(n *nethtml.Node) bool {
	for _, a := range n.Attr {
		if a.Key != "class" {
			continue
		}
		for _, cls := range strings.Fields(a.Val) {
			if searchSkipClasses[cls] {
				return true
			}
		}
	}
	return false
}
