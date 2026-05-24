package socialcards

import (
	"strings"
	"unicode"

	"golang.org/x/image/font"
)

// wrapText splits text into lines that fit within maxWidth pixels.
// If maxLines > 0, the result is capped at that many lines with "..." truncation.
func wrapText(text string, face font.Face, maxWidth int, maxLines int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	words := splitWords(text)
	if len(words) == 0 {
		return nil
	}

	maxWidthFixed := fixed266ToInt(maxWidth)
	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		candidate := currentLine.String()
		if candidate == "" {
			candidate = word
		} else {
			candidate += " " + word
		}

		width := font.MeasureString(face, candidate)
		if width.Ceil() > maxWidthFixed && currentLine.Len() > 0 {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		} else {
			currentLine.Reset()
			currentLine.WriteString(candidate)
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
		last := lines[maxLines-1]
		if len(last) > 3 {
			lines[maxLines-1] = truncateWithEllipsis(last, face, maxWidthFixed)
		}
	}

	return lines
}

// truncateWithEllipsis shortens a line so that "..." fits within maxWidth.
func truncateWithEllipsis(line string, face font.Face, maxWidth int) string {
	ellipsis := "..."
	ellipsisWidth := font.MeasureString(face, ellipsis).Ceil()
	targetWidth := maxWidth - ellipsisWidth

	runes := []rune(line)
	for i := len(runes); i > 0; i-- {
		candidate := string(runes[:i])
		if font.MeasureString(face, candidate).Ceil() <= targetWidth {
			return strings.TrimRightFunc(candidate, unicode.IsSpace) + ellipsis
		}
	}
	return ellipsis
}

func splitWords(s string) []string {
	return strings.Fields(s)
}

func fixed266ToInt(px int) int {
	return px
}
