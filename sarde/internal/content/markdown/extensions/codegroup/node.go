package codegroup

import gast "github.com/yuin/goldmark/ast"

// KindCodeGroupBlock is the NodeKind for CodeGroupBlock.
var KindCodeGroupBlock = gast.NewNodeKind("CodeGroupBlock")

// CodeGroupBlock is an AST node representing a tabbed code group.
// Its children are standard fenced code blocks, each rendered as a tab.
type CodeGroupBlock struct {
	gast.BaseBlock
}

// Kind implements ast.Node.Kind.
func (n *CodeGroupBlock) Kind() gast.NodeKind {
	return KindCodeGroupBlock
}

// Dump implements ast.Node.Dump.
func (n *CodeGroupBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
