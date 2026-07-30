package content

import (
	"html/template"
	"math"
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/engine"
)

const defaultWordsPerMinute = 200

// Transformer enriches a Page with computed fields: word count, reading time, and summary.
type Transformer struct {
	SummaryLength int // max words in auto-generated summary (from config)
}

// Transform computes WordCount, ReadingTime, and Summary for a page.
func (t *Transformer) Transform(page *engine.Page) error {
	// Word count from raw markdown
	page.WordCount = countWords(page.RawContent)

	// Reading time: ceil(words / 200), minimum 1 minute
	if page.WordCount > 0 {
		page.ReadingTime = int(math.Ceil(float64(page.WordCount) / float64(defaultWordsPerMinute)))
	} else {
		page.ReadingTime = 0
	}
	if page.ReadingTime < 1 && page.WordCount > 0 {
		page.ReadingTime = 1
	}

	// Auto-description: if frontmatter omitted description, derive from first paragraph.
	if page.Description == "" && page.RawContent != "" {
		page.Description = extractDescription(page.RawContent, 160)
	}

	// Summary: description → first paragraph → truncated content
	if page.Summary == "" {
		if page.Description != "" {
			page.Summary = template.HTML(page.Description)
		} else {
			summaryLen := t.SummaryLength
			if summaryLen <= 0 {
				summaryLen = 70
			}
			page.Summary = template.HTML(extractSummary(page.RawContent, summaryLen))
		}
	}

	return nil
}

// countWords counts whitespace-separated tokens in text.
func countWords(text string) int {
	return len(strings.Fields(text))
}

// directiveOpenRe matches an opening directive fence: three or more colons
// followed by a name, e.g. ":::card-grid" or ":::card(title=\"Fast\")". A
// bare ":::" or a ":::/name" closer has no name character after the colons
// and deliberately does not match.
var directiveOpenRe = regexp.MustCompile(`^:{3,}\s*[\w-]`)

// firstProseParagraph extracts the first prose paragraph from raw markdown.
// It skips headings, frontmatter-like "---" lines, HTML block lines, fenced
// code blocks, and directive container blocks. Directives nest with the same
// fence length (":::card-grid" containing ":::card", both closed by a bare
// ":::"), so nesting is tracked by depth rather than a toggle; a stray
// closer clamps at depth zero. Returns "" when the body has no prose at all,
// e.g. a homepage that is entirely directive blocks.
func firstProseParagraph(markdown string) string {
	lines := strings.Split(markdown, "\n")
	var para strings.Builder
	inCodeFence := false
	directiveDepth := 0
	inPara := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeFence = !inCodeFence
			continue
		}
		if inCodeFence {
			continue
		}
		if strings.HasPrefix(trimmed, ":::") {
			if directiveOpenRe.MatchString(trimmed) {
				directiveDepth++
			} else if directiveDepth > 0 {
				directiveDepth--
			}
			continue
		}
		if directiveDepth > 0 {
			continue
		}
		if trimmed == "" {
			if inPara {
				break // end of paragraph
			}
			continue
		}
		// Skip headings, frontmatter-like lines, and HTML blocks.
		if strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "---") ||
			strings.HasPrefix(trimmed, "<") {
			if inPara {
				break
			}
			continue
		}
		if inPara {
			para.WriteString(" ")
		}
		para.WriteString(trimmed)
		inPara = true
	}

	return para.String()
}

// extractSummary extracts the first prose paragraph from markdown, truncated
// to maxWords.
func extractSummary(markdown string, maxWords int) string {
	text := firstProseParagraph(markdown)
	words := strings.Fields(text)
	if len(words) > maxWords {
		words = words[:maxWords]
		return strings.Join(words, " ") + "..."
	}
	return strings.Join(words, " ")
}

// extractDescription extracts a plain-text description from markdown content,
// suitable for SEO meta tags. Truncates to maxChars at a word boundary.
func extractDescription(markdown string, maxChars int) string {
	text := firstProseParagraph(markdown)
	if text == "" {
		return ""
	}
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
