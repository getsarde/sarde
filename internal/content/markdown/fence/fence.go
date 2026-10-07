// Package fence is the one grammar for ":::" directive fence lines. The
// Goldmark container parsers (through blockutil), sarde check-syntax and the
// content_lint plugin all classify lines with it, so a fence means the same
// thing to the renderer and to the tools that check content before a build.
package fence

import "strings"

// Kind classifies a trimmed line.
type Kind int

const (
	// None is a line that is not a fence.
	None Kind = iota
	// Open is ":::name ..." with three or more colons.
	Open
	// Close is a bare ":::" or a named ":::/name" with nothing else on the line.
	Close
	// Malformed is ":::/" followed by something the parsers reject, such as
	// ":::/ note" or ":::/note extra". Parsers treat it as content; the
	// checker reports it.
	Malformed
)

// Fence describes one classified line.
type Fence struct {
	Kind   Kind
	Name   string // lowercased; empty for a bare closer or a nameless Malformed
	Colons int
}

// Classify reads a whitespace-trimmed line. The opener name is the run of
// word characters and hyphens after the colons and optional spaces; a named
// closer must end right after its name.
func Classify(trimmed string) Fence {
	n := leadingRun(trimmed, ':')
	if n < 3 {
		return Fence{}
	}
	rest := trimmed[n:]
	if rest == "" {
		return Fence{Kind: Close, Colons: n}
	}
	if rest[0] == '/' {
		name := nameRun(rest[1:])
		if name == "" || strings.TrimSpace(rest[1+len(name):]) != "" {
			// Malformed; keep the intended name (":::/ note" means note) so
			// the checker can say what to write. ":::/" alone has none.
			return Fence{Kind: Malformed, Name: strings.ToLower(nameRun(strings.TrimLeft(rest[1:], " \t"))), Colons: n}
		}
		return Fence{Kind: Close, Name: strings.ToLower(name), Colons: n}
	}
	name := nameRun(strings.TrimLeft(rest, " \t"))
	if name == "" || !isWord(name[0]) {
		return Fence{Kind: None, Colons: n}
	}
	return Fence{Kind: Open, Name: strings.ToLower(name), Colons: n}
}

// AsideTypes lists the aside names the aside extension opens. The aside
// parser's ValidTypes set is built from it, and ":::/aside" closes any of
// them.
var AsideTypes = []string{
	"note", "tip", "info", "danger", "warning", "important", "caution",
	"gh-note", "gh-tip", "gh-important", "gh-warning", "gh-caution",
}

// closerAliases maps a closer name to the opener names it also closes.
var closerAliases = map[string][]string{
	"filetree": {"file-tree"},
	"aside":    AsideTypes,
}

// Closes reports whether ":::/closer" closes a block opened as ":::opener".
// Names compare case-insensitively.
func Closes(closer, opener string) bool {
	closer, opener = strings.ToLower(closer), strings.ToLower(opener)
	if closer == opener {
		return true
	}
	for _, o := range closerAliases[closer] {
		if o == opener {
			return true
		}
	}
	return false
}

// FindClosable returns the index of the innermost (last) name in names that
// closer closes, or -1.
func FindClosable(names []string, closer string) int {
	for i := len(names) - 1; i >= 0; i-- {
		if Closes(closer, names[i]) {
			return i
		}
	}
	return -1
}

// CodeFence tracks fenced code blocks line by line. An opener is a run of
// three or more backticks or tildes; the closer is a run of the same
// character at least as long, alone on its line.
type CodeFence struct {
	char   byte
	length int
}

// Feed reports whether trimmed is inside a fenced code block, counting the
// opening and closing fence lines themselves.
func (c *CodeFence) Feed(trimmed string) bool {
	if c.char == 0 {
		if n := leadingRun(trimmed, '`'); n >= 3 {
			c.char, c.length = '`', n
			return true
		}
		if n := leadingRun(trimmed, '~'); n >= 3 {
			c.char, c.length = '~', n
			return true
		}
		return false
	}
	if n := leadingRun(trimmed, c.char); n >= c.length && n == len(trimmed) {
		c.char, c.length = 0, 0
	}
	return true
}

// Inside reports whether the tracker is currently inside a code block.
func (c *CodeFence) Inside() bool { return c.char != 0 }

func leadingRun(s string, ch byte) int {
	n := 0
	for n < len(s) && s[n] == ch {
		n++
	}
	return n
}

// nameRun returns the leading run of [A-Za-z0-9_-] characters.
func nameRun(s string) string {
	n := 0
	for n < len(s) && (isWord(s[n]) || s[n] == '-') {
		n++
	}
	return s[:n]
}

func isWord(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}
