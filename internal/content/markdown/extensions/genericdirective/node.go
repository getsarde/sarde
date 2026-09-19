// Package genericdirective is the goldmark side of site- and theme-authored
// generic directives (internal/directive): one block parser and renderer
// handle every registered ::: directive, dispatching on the fence name.
package genericdirective

import gast "github.com/yuin/goldmark/ast"

var KindGenericDirective = gast.NewNodeKind("GenericDirective")

// Node is a parsed generic directive block. Container directives carry their
// body as parsed children and record where its raw Markdown sits; leaf directives
// capture the body verbatim in RawBody.
type Node struct {
	gast.BaseBlock
	Name    string
	Label   string
	Attrs   map[string]string
	RawBody string
	// SourceStart and SourceStop bound the raw body of a container directive in
	// the document source. SourceStop stays 0 when the fence is never closed.
	SourceStart int
	SourceStop  int
}

func (n *Node) Kind() gast.NodeKind { return KindGenericDirective }

func (n *Node) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{"Name": n.Name, "Label": n.Label}, nil)
}
