package syntax

import (
	"bytes"
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/fence"
)

type stackEntry struct {
	tag  string
	line int
}

// Check scans markdown content for unclosed, mismatched or malformed ":::"
// fences. It follows the renderer's rules (the fence package, shared with the
// block parsers): a bare ":::" closes the innermost open block, ":::/name"
// closes the nearest open block that name closes and takes any inner block
// left open with it, and a ":::/name" that matches no open block is content.
// Lines inside fenced code blocks (``` or ~~~) are skipped.
// lineOffset is added to all reported line numbers (use page.FrontmatterLines
// when checking RawContent that has frontmatter stripped).
func Check(filename string, content []byte, lineOffset int) []Diagnostic {
	var diags []Diagnostic
	var stack []stackEntry
	var code fence.CodeFence

	diag := func(line int, tag, msg, level string) {
		diags = append(diags, Diagnostic{File: filename, Line: line, Tag: tag, Message: msg, Level: level})
	}

	lines := bytes.Split(content, []byte("\n"))
	for i, line := range lines {
		lineNum := i + 1 + lineOffset
		trimmed := strings.TrimSpace(string(line))

		if code.Feed(trimmed) {
			continue
		}

		f := fence.Classify(trimmed)
		switch f.Kind {
		case fence.Open:
			stack = append(stack, stackEntry{tag: f.Name, line: lineNum})

		case fence.Close:
			if f.Name == "" {
				if len(stack) == 0 {
					diag(lineNum, "", "orphaned closing tag with no matching opener", "error")
				} else {
					stack = stack[:len(stack)-1]
				}
				continue
			}
			if len(stack) == 0 {
				diag(lineNum, f.Name, "closing tag ':::/"+f.Name+"' with no matching opener", "error")
				continue
			}
			names := make([]string, len(stack))
			for k, e := range stack {
				names[k] = e.tag
			}
			k := fence.FindClosable(names, f.Name)
			if k < 0 {
				// The renderer leaves the block open and prints the line as
				// text, so the block stays on the stack.
				top := stack[len(stack)-1]
				diag(lineNum, f.Name, "mismatched closing tag ':::/"+f.Name+"', expected ':::/"+top.tag+"' (opened at line "+itoa(top.line)+")", "error")
				continue
			}
			for _, e := range stack[k+1:] {
				diag(e.line, e.tag, "unclosed block ':::"+e.tag+"' (opened at line "+itoa(e.line)+"), closed implicitly by ':::/"+f.Name+"' at line "+itoa(lineNum), "warning")
			}
			stack = stack[:k]

		case fence.Malformed:
			// ":::/" alone is skipped, as before; a closer with a name but
			// trailing text is a typo the renderer prints as content.
			if f.Name != "" {
				diag(lineNum, f.Name, "malformed closing fence '"+trimmed+"': write ':::/"+f.Name+"' with nothing after the name", "error")
			}
		}
	}

	for _, entry := range stack {
		diag(entry.line, entry.tag, "unclosed block ':::"+entry.tag+"' (opened at line "+itoa(entry.line)+")", "warning")
	}

	return diags
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
