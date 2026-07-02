// Package blockutil provides shared helpers for container-style block
// directive parsers. Directives that open with a ":::name" fence and close
// with ":::" track how many nested container fences are currently open so
// that a closing fence is matched to the correct block.
package blockutil

import (
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
)

// depthKey stores a map[ast.Node]int in the parser context. Entries are
// keyed by the container node itself, so one key is safe to share across
// all container extensions.
var depthKey = parser.NewContextKey()

// GetDepth returns the nesting depth recorded for node, or 0 if none.
func GetDepth(pc parser.Context, node ast.Node) int {
	v := pc.Get(depthKey)
	if v == nil {
		return 0
	}
	if m, ok := v.(map[ast.Node]int); ok {
		return m[node]
	}
	return 0
}

// SetDepth records the nesting depth for node.
func SetDepth(pc parser.Context, node ast.Node, depth int) {
	var m map[ast.Node]int
	if v := pc.Get(depthKey); v != nil {
		m = v.(map[ast.Node]int)
	} else {
		m = make(map[ast.Node]int)
	}
	m[node] = depth
	pc.Set(depthKey, m)
}

// DeleteDepth removes the depth entry for node.
func DeleteDepth(pc parser.Context, node ast.Node) {
	if v := pc.Get(depthKey); v != nil {
		if m, ok := v.(map[ast.Node]int); ok {
			delete(m, node)
		}
	}
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
