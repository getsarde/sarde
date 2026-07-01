package syntax

import (
	"bytes"
	"strings"
)

type stackEntry struct {
	tag  string
	line int
}

// Check scans markdown content for unclosed or mismatched fenced block tags.
// Lines inside fenced code blocks (``` or ~~~) are skipped.
// lineOffset is added to all reported line numbers (use page.FrontmatterLines
// when checking RawContent that has frontmatter stripped).
func Check(filename string, content []byte, lineOffset int) []Diagnostic {
	var diags []Diagnostic
	var stack []stackEntry
	inCodeFence := false

	lines := bytes.Split(content, []byte("\n"))
	for i, line := range lines {
		lineNum := i + 1 + lineOffset
		trimmed := strings.TrimSpace(string(line))

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inCodeFence = !inCodeFence
			continue
		}
		if inCodeFence {
			continue
		}

		if !strings.HasPrefix(trimmed, ":::") {
			continue
		}

		rest := trimmed[3:]
		for strings.HasPrefix(rest, ":") {
			rest = rest[1:]
		}
		rest = strings.TrimSpace(rest)

		if rest == "" {
			// Bare ::: — implicit close
			if len(stack) == 0 {
				diags = append(diags, Diagnostic{
					File:    filename,
					Line:    lineNum,
					Tag:     "",
					Message: "orphaned closing tag with no matching opener",
					Level:   "error",
				})
			} else {
				stack = stack[:len(stack)-1]
			}
			continue
		}

		if rest[0] == '/' {
			// Explicit close: :::/tagname
			fields := strings.Fields(rest[1:])
			if len(fields) == 0 {
				continue
			}
			closeTag := fields[0]
			if len(stack) == 0 {
				diags = append(diags, Diagnostic{
					File:    filename,
					Line:    lineNum,
					Tag:     closeTag,
					Message: "closing tag :::/'" + closeTag + "' with no matching opener",
					Level:   "error",
				})
			} else {
				top := stack[len(stack)-1]
				if top.tag != closeTag {
					diags = append(diags, Diagnostic{
						File:    filename,
						Line:    lineNum,
						Tag:     closeTag,
						Message: "mismatched closing tag ':::/'" + closeTag + "', expected ':::/'" + top.tag + "' (opened at line " + itoa(top.line) + ")",
						Level:   "error",
					})
				}
				stack = stack[:len(stack)-1]
			}
			continue
		}

		// Opener: :::tagname[...](...)
		tag := extractTag(rest)
		if tag != "" {
			stack = append(stack, stackEntry{tag: tag, line: lineNum})
		}
	}

	for _, entry := range stack {
		diags = append(diags, Diagnostic{
			File:    filename,
			Line:    entry.line,
			Tag:     entry.tag,
			Message: "unclosed block ':::'" + entry.tag + "' (opened at line " + itoa(entry.line) + ")",
			Level:   "warning",
		})
	}

	return diags
}

func extractTag(s string) string {
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			b.WriteRune(c)
		} else {
			break
		}
	}
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
