package htmlutil

import (
	"fmt"
	"hash/fnv"

	"github.com/yuin/goldmark/ast"
)

// PageScopedID returns an HTML id for node that is stable across builds:
// prefix, a hash of the page's Markdown source, and the node's 1-based
// position among the nodes of its kind in document order, as in
// "ann-3f9a1c2e-2". A counter shared by every page of a build gives ids that
// depend on the order pages render in, which changes between parallel
// builds; the source hash keeps ids distinct when one template shows several
// pages' content.
func PageScopedID(prefix string, source []byte, node ast.Node) string {
	h := fnv.New32a()
	_, _ = h.Write(source)

	var root ast.Node = node
	for root.Parent() != nil {
		root = root.Parent()
	}
	kind := node.Kind()
	n := 0
	_ = ast.Walk(root, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || c.Kind() != kind {
			return ast.WalkContinue, nil
		}
		n++
		if c == node {
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})
	return fmt.Sprintf("%s-%08x-%d", prefix, h.Sum32(), n)
}
