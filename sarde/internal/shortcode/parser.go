package shortcode

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// reSelfClosing matches: {{< name key="value" />}}
	reSelfClosing = regexp.MustCompile(`\{\{<\s*([\w-]+)((?:\s+[\w-]+=(?:"[^"]*"|'[^']*'|\S+))*)\s*/>\}\}`)

	// reOpening matches: {{< name key="value" >}}
	reOpening = regexp.MustCompile(`\{\{<\s*([\w-]+)((?:\s+[\w-]+=(?:"[^"]*"|'[^']*'|\S+))*)\s*>\}\}`)

	// reClosing matches: {{< /name >}}
	reClosing = regexp.MustCompile(`\{\{<\s*/([\w-]+)\s*>\}\}`)

	// reParam extracts key=value pairs from the attributes string.
	reParam = regexp.MustCompile(`([\w-]+)=(?:"([^"]*)"|'([^']*)'|(\S+))`)
)

// ParseParams extracts key=value pairs from a raw attribute string.
func ParseParams(raw string) map[string]string {
	params := make(map[string]string)
	// Index-based submatches distinguish "group matched the empty string"
	// (key="") from "group did not participate", which string submatches
	// cannot.
	for _, m := range reParam.FindAllStringSubmatchIndex(raw, -1) {
		key := raw[m[2]:m[3]]
		switch {
		case m[4] >= 0: // double-quoted value
			params[key] = raw[m[4]:m[5]]
		case m[6] >= 0: // single-quoted value
			params[key] = raw[m[6]:m[7]]
		default: // bare value
			params[key] = raw[m[8]:m[9]]
		}
	}
	return params
}

const sentinel = "\x00SC_BLOCK_%d\x00"

// stripCodeBlocks replaces fenced code blocks with opaque sentinels so that
// shortcode syntax inside code blocks is not expanded.
func stripCodeBlocks(src string) (string, []string) {
	var blocks []string
	var out strings.Builder
	lines := strings.Split(src, "\n")
	inFence := false
	var fenceChar byte
	var fenceLen int
	var blockBuf strings.Builder

	for i, line := range lines {
		if !inFence {
			trimmed := strings.TrimLeft(line, " \t")
			if len(trimmed) >= 3 && (trimmed[0] == '`' || trimmed[0] == '~') {
				ch := trimmed[0]
				n := countLeading(trimmed, ch)
				if n >= 3 {
					inFence = true
					fenceChar = ch
					fenceLen = n
					blockBuf.Reset()
					blockBuf.WriteString(line)
					continue
				}
			}
			out.WriteString(line)
			if i < len(lines)-1 {
				out.WriteByte('\n')
			}
		} else {
			blockBuf.WriteByte('\n')
			blockBuf.WriteString(line)

			trimmed := strings.TrimLeft(line, " \t")
			if len(trimmed) >= fenceLen && trimmed[0] == fenceChar {
				n := countLeading(trimmed, fenceChar)
				rest := strings.TrimSpace(trimmed[n:])
				if n >= fenceLen && rest == "" {
					inFence = false
					idx := len(blocks)
					blocks = append(blocks, blockBuf.String())
					out.WriteString(fmt.Sprintf(sentinel, idx))
					if i < len(lines)-1 {
						out.WriteByte('\n')
					}
				}
			}
		}
	}

	// Unclosed fence — treat the entire remaining block as code.
	if inFence {
		idx := len(blocks)
		blocks = append(blocks, blockBuf.String())
		out.WriteString(fmt.Sprintf(sentinel, idx))
	}

	return out.String(), blocks
}

// restoreCodeBlocks replaces sentinels with the original code block text.
func restoreCodeBlocks(src string, blocks []string) string {
	if len(blocks) == 0 {
		return src
	}
	result := src
	for i, block := range blocks {
		result = strings.Replace(result, fmt.Sprintf(sentinel, i), block, 1)
	}
	return result
}

func countLeading(s string, ch byte) int {
	n := 0
	for n < len(s) && s[n] == ch {
		n++
	}
	return n
}
