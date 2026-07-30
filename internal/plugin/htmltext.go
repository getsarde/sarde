package plugin

import (
	"html/template"
	"strings"
	"unicode/utf8"
)

// RenderedFallbackMaxChars is the description budget used by consumers that
// fall back to RenderedTextFallback, matching the 160-character convention of
// the transformer's auto-description.
const RenderedFallbackMaxChars = 160

// renderedFallbackScanBytes bounds how much rendered HTML is scanned before
// stripping. A multiple of the character budget is not enough here: the head
// of a rendered page can be all markup, so the bound is a generous fixed size
// instead.
const renderedFallbackScanBytes = 4096

// StripHTML removes HTML tags and collapses whitespace, inserting a space at
// tag boundaries so adjacent elements do not run together.
func StripHTML(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inTag := false
	lastSpace := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '<' {
			inTag = true
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
			continue
		}
		if c == '>' {
			inTag = false
			continue
		}
		if inTag {
			continue
		}
		if c <= ' ' {
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
			continue
		}
		b.WriteByte(c)
		lastSpace = false
	}
	return strings.TrimSpace(b.String())
}

// TruncateRuneSafe cuts s to at most max bytes without splitting a UTF-8 rune.
func TruncateRuneSafe(s string, max int) string {
	if max < 0 {
		max = 0
	}
	if len(s) <= max {
		return s
	}
	for max > 0 && !utf8.RuneStart(s[max]) {
		max--
	}
	return s[:max]
}

// TrimTitlePrefix drops a leading title prefix from a rendered-text excerpt.
// The rendered body usually opens with the page's own H1, which reads as
// duplication anywhere the title is already displayed alongside the text.
func TrimTitlePrefix(text, title string) string {
	if title == "" {
		return text
	}
	if trimmed := strings.TrimPrefix(text, title); trimmed != text {
		return strings.TrimSpace(trimmed)
	}
	return text
}

// RenderedTextFallback returns a plain-text excerpt of rendered page HTML,
// truncated to at most maxChars at a word boundary, for use as a last-resort
// description when both the frontmatter description and the auto-generated
// summary are empty (e.g. a body that is entirely directive blocks, which
// the raw-markdown summary extractor rightly skips). Entities are left
// encoded: callers apply a single terminal html.UnescapeString so the value
// is decoded exactly once.
func RenderedTextFallback(content template.HTML, maxChars int) string {
	text := StripHTML(TruncateRuneSafe(string(content), renderedFallbackScanBytes))
	runes := []rune(text)
	if len(runes) <= maxChars {
		return text
	}
	truncated := string(runes[:maxChars])
	if idx := strings.LastIndex(truncated, " "); idx > 0 {
		truncated = truncated[:idx]
	}
	return truncated + "..."
}
