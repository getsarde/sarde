package details

import gast "github.com/yuin/goldmark/ast"

// KindDetailsBlock is the NodeKind for DetailsBlock.
var KindDetailsBlock = gast.NewNodeKind("DetailsBlock")

// DetailsBlock is an AST node representing a collapsible details block.
type DetailsBlock struct {
	gast.BaseBlock
	Summary string // The summary/title text
	Open    bool   // Whether the details are open by default
}

// Kind implements ast.Node.Kind.
func (n *DetailsBlock) Kind() gast.NodeKind {
	return KindDetailsBlock
}

// Dump implements ast.Node.Dump.
func (n *DetailsBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Summary": n.Summary,
	}, nil)
}
