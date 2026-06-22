package markdown

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"

	"github.com/frostybee/sarde/internal/engine"
	nethtml "golang.org/x/net/html"
)

var slugifyRegex = regexp.MustCompile(`[^a-z0-9]+`)

// extractHeadings parses rendered HTML, finds h2-h4 headings, injects slugified
// IDs and anchor links, and returns the headings as engine.Heading slices.
// The htmlContent is modified in place with IDs and anchors injected.
func extractHeadings(htmlContent *string) []engine.Heading {
	var headings []engine.Heading

	doc, err := nethtml.Parse(strings.NewReader(*htmlContent))
	if err != nil {
		return headings
	}

	usedIDs := make(map[string]bool)

	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode && (n.Data == "h2" || n.Data == "h3" || n.Data == "h4") {
			text := extractText(n)
			level := int(n.Data[1] - '0')
			id := slugifyHeading(text)

			// Handle ID collisions
			if usedIDs[id] {
				counter := 1
				for usedIDs[id+"-"+strconv.Itoa(counter)] {
					counter++
				}
				id = id + "-" + strconv.Itoa(counter)
			}
			usedIDs[id] = true

			setAttr(n, "id", id)

			anchor := &nethtml.Node{
				Type: nethtml.ElementNode,
				Data: "a",
				Attr: []nethtml.Attribute{
					{Key: "class", Val: "sarde-heading-anchor"},
					{Key: "href", Val: "#" + id},
					{Key: "aria-label", Val: "Link to section: " + text},
				},
			}
			n.AppendChild(anchor)

			headings = append(headings, engine.Heading{
				Level: level,
				ID:    id,
				Text:  text,
			})
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	var buf bytes.Buffer
	if err := nethtml.Render(&buf, doc); err == nil {
		rendered := buf.String()
		rendered = extractBodyContent(rendered)
		*htmlContent = rendered
	}

	return headings
}

func extractText(n *nethtml.Node) string {
	var buf bytes.Buffer
	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if n.Type == nethtml.TextNode {
			buf.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(buf.String())
}

func setAttr(n *nethtml.Node, key, val string) {
	for i, attr := range n.Attr {
		if attr.Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, nethtml.Attribute{Key: key, Val: val})
}

func slugifyHeading(text string) string {
	text = strings.ToLower(text)
	text = slugifyRegex.ReplaceAllString(text, "-")
	text = strings.Trim(text, "-")
	if text == "" {
		return "section"
	}
	return text
}

func extractBodyContent(htmlStr string) string {
	bodyStart := strings.Index(htmlStr, "<body>")
	bodyEnd := strings.LastIndex(htmlStr, "</body>")
	if bodyStart >= 0 && bodyEnd > bodyStart {
		return htmlStr[bodyStart+6 : bodyEnd]
	}
	return htmlStr
}
