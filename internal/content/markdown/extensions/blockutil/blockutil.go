// Package blockutil provides shared helpers for container-style block
// directive parsers. Directives that open with a ":::name" fence and close
// with ":::" or ":::/name" share one Continue implementation, so every
// container nests, closes and recovers the same way.
package blockutil

import (
	"strings"

	"github.com/getsarde/sarde/internal/content/markdown/fence"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// containerState is what an open container remembers about the lines it has
// seen. Goldmark hands every line to the outermost open block first, so a
// container sees the fences of the blocks nested inside it and has to track
// them to know which closer is its own.
type containerState struct {
	// nested holds the lowercased names of nested openers seen since this
	// block opened and not yet closed, innermost last.
	nested []string
	// code tracks fenced code blocks inside the container; fence lines
	// inside them are content.
	code fence.CodeFence
}

// stateKey stores a map[ast.Node]*containerState in the parser context.
// Entries are keyed by the container node itself, so one key is safe to
// share across all container extensions.
var stateKey = parser.NewContextKey()

func getState(pc parser.Context, node ast.Node) *containerState {
	var m map[ast.Node]*containerState
	if v := pc.Get(stateKey); v != nil {
		m = v.(map[ast.Node]*containerState)
	} else {
		m = make(map[ast.Node]*containerState)
		pc.Set(stateKey, m)
	}
	st := m[node]
	if st == nil {
		st = &containerState{}
		m[node] = st
	}
	return st
}

// Release drops the state recorded for node. Call it from Close.
func Release(pc parser.Context, node ast.Node) {
	if v := pc.Get(stateKey); v != nil {
		if m, ok := v.(map[ast.Node]*containerState); ok {
			delete(m, node)
		}
	}
}

// ContinueContainer is the Continue body shared by every ":::name" container.
// name is the node's canonical lowercased opener name ("tabs", "file-tree",
// an aside's type, a generic directive's name); closer aliases and case
// folding come from fence.Closes.
//
// Rules, for the trimmed line:
//   - inside a fenced code block: content
//   - ":::word": a nested opener, pushed on the stack
//   - ":::": pops one nested opener; with none left, closes this block when
//     no inner ':'-triggered block is still open
//   - ":::/x" that closes a nested opener: pops through it, taking any
//     opener left open above it
//   - ":::/x" that closes this block: closes it, and Goldmark closes any
//     child left open
//   - any other ":::/x": content
func ContinueContainer(pc parser.Context, node ast.Node, reader text.Reader, name string) parser.State {
	line, _ := reader.PeekLine()
	trimmed := strings.TrimSpace(string(line))
	st := getState(pc, node)

	if st.code.Feed(trimmed) {
		return parser.Continue | parser.HasChildren
	}
	if !strings.HasPrefix(trimmed, ":::") {
		return parser.Continue | parser.HasChildren
	}

	f := fence.Classify(trimmed)
	switch f.Kind {
	case fence.Open:
		st.nested = append(st.nested, f.Name)
	case fence.Close:
		if f.Name == "" {
			if n := len(st.nested); n > 0 {
				st.nested = st.nested[:n-1]
			} else if !HasInnerOpenBlocks(pc, node) {
				reader.AdvanceToEOL()
				return parser.Close
			}
			return parser.Continue | parser.HasChildren
		}
		if i := fence.FindClosable(st.nested, f.Name); i >= 0 {
			st.nested = st.nested[:i]
		} else if fence.Closes(f.Name, name) {
			reader.AdvanceToEOL()
			return parser.Close
		}
	}
	return parser.Continue | parser.HasChildren
}

// HasInnerOpenBlocks reports whether a ':'-triggered container opened after
// node is still open, meaning a bare closing fence belongs to that inner
// container rather than to node.
func HasInnerOpenBlocks(pc parser.Context, node ast.Node) bool {
	blocks := pc.OpenedBlocks()
	for i, b := range blocks {
		if b.Node == node {
			for j := i + 1; j < len(blocks); j++ {
				for _, t := range blocks[j].Parser.Trigger() {
					if t == ':' {
						return true
					}
				}
			}
			return false
		}
	}
	return false
}
