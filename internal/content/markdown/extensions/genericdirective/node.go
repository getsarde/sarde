// Package genericdirective is the goldmark side of site- and theme-authored
// generic directives (internal/directive): one block parser and renderer
// handle every registered ::: directive, dispatching on the fence name.
package genericdirective

import gast "github.com/yuin/goldmark/ast"

var KindGenericDirective = gast.NewNodeKind("GenericDirective")

// Node is a parsed generic directive block. Container directives carry their
// body as parsed children; leaf directives capture it verbatim in RawBody.
type Node struct {
	gast.BaseBlock
	Name    string
	Label   string
	Attrs   map[string]string
	RawBody string
}

func (n *Node) Kind() gast.NodeKind { return KindGenericDirective }

func (n *Node) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{"Name": n.Name, "Label": n.Label}, nil)
}
