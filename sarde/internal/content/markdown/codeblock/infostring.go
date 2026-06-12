package codeblock

import (
	"regexp"
	"strconv"
	"strings"
)

// CodeBlockInfo holds parsed info string metadata for a fenced code block.
type CodeBlockInfo struct {
	Language        string
	Title           string
	HighlightLines  map[int]bool // lines to highlight
	InsertedLines   map[int]bool // lines marked as inserted (green)
	DeletedLines    map[int]bool // lines marked as deleted (red)
	ShowLineNumbers bool
	Collapse        bool
	NoCollapse      bool
	IsTerminal      bool // auto-detected from language
}

var (
	titleRegex     = regexp.MustCompile(`title=(?:"([^"]+)"|'([^']+)')`)
	highlightRegex = regexp.MustCompile(`\{([^}]+)\}`)
	insRegex       = regexp.MustCompile(`ins=\{([^}]+)\}`)
	delRegex       = regexp.MustCompile(`del=\{([^}]+)\}`)
)

var terminalLanguages = map[string]bool{
	"bash": true, "sh": true, "shell": true, "sarde-terminal": true,
	"zsh": true, "fish": true, "powershell": true, "ps1": true,
	"cmd": true, "bat": true, "console": true,
}

// ParseInfoString parses a fenced code block info string into structured metadata.
func ParseInfoString(info string) CodeBlockInfo {
	info = strings.TrimSpace(info)
	result := CodeBlockInfo{
		HighlightLines: make(map[int]bool),
		InsertedLines:  make(map[int]bool),
		DeletedLines:   make(map[int]bool),
	}

	if info == "" {
		return result
	}

	// Extract title (must be done before general parsing to avoid conflicts)
	if m := titleRegex.FindStringSubmatch(info); m != nil {
		if m[1] != "" {
			result.Title = m[1]
		} else {
			result.Title = m[2]
		}
		info = titleRegex.ReplaceAllString(info, "")
	}

	// Extract ins={...} before general highlight to avoid confusion
	if m := insRegex.FindStringSubmatch(info); m != nil {
		result.InsertedLines = parseLineRanges(m[1])
		info = insRegex.ReplaceAllString(info, "")
	}

	// Extract del={...}
	if m := delRegex.FindStringSubmatch(info); m != nil {
		result.DeletedLines = parseLineRanges(m[1])
		info = delRegex.ReplaceAllString(info, "")
	}

	// Extract highlight {2-5,8}
	if m := highlightRegex.FindStringSubmatch(info); m != nil {
		result.HighlightLines = parseLineRanges(m[1])
		info = highlightRegex.ReplaceAllString(info, "")
	}

	// Check for flags (nocollapse before collapse — collapse is a substring)
	if strings.Contains(info, "showLineNumbers") {
		result.ShowLineNumbers = true
		info = strings.ReplaceAll(info, "showLineNumbers", "")
	}
	if strings.Contains(info, "nocollapse") {
		result.NoCollapse = true
		info = strings.ReplaceAll(info, "nocollapse", "")
	} else if strings.Contains(info, "collapse") {
		result.Collapse = true
		info = strings.ReplaceAll(info, "collapse", "")
	}

	// Language is the first word remaining
	fields := strings.Fields(info)
	if len(fields) > 0 {
		result.Language = fields[0]
	}

	// Auto-detect terminal
	result.IsTerminal = terminalLanguages[result.Language]

	return result
}

// maxLineRangeSpan caps how many lines a single range like {1-1000000} may
// expand to, so hostile or typo'd info strings cannot allocate unbounded maps.
const maxLineRangeSpan = 10000

// parseLineRanges parses "2-5,8,10-12" into a set of line numbers.
func parseLineRanges(s string) map[int]bool {
	lines := make(map[int]bool)
	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if idx := strings.Index(part, "-"); idx > 0 {
			start, err1 := strconv.Atoi(strings.TrimSpace(part[:idx]))
			end, err2 := strconv.Atoi(strings.TrimSpace(part[idx+1:]))
			if err1 == nil && err2 == nil {
				if start < 1 {
					start = 1
				}
				if end-start+1 > maxLineRangeSpan {
					end = start + maxLineRangeSpan - 1
				}
				for i := start; i <= end; i++ {
					lines[i] = true
				}
			}
		} else {
			if n, err := strconv.Atoi(part); err == nil && n >= 1 {
				lines[n] = true
			}
		}
	}
	return lines
}
