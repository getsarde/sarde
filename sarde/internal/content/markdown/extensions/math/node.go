package math

import gast "github.com/yuin/goldmark/ast"

// KindInlineMath is the NodeKind for InlineMath.
var KindInlineMath = gast.NewNodeKind("InlineMath")

// KindDisplayMath is the NodeKind for DisplayMath.
var KindDisplayMath = gast.NewNodeKind("DisplayMath")

// InlineMath is an inline AST node for $...$ math expressions.
type InlineMath struct {
	gast.BaseInline
	Expression string
}

// Kind implements ast.Node.Kind.
func (n *InlineMath) Kind() gast.NodeKind {
	return KindInlineMath
}

// Dump implements ast.Node.Dump.
func (n *InlineMath) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Expression": n.Expression,
	}, nil)
}

// DisplayMath is a block AST node for $$...$$ math expressions.
type DisplayMath struct {
	gast.BaseBlock
	Expression string
}

// Kind implements ast.Node.Kind.
func (n *DisplayMath) Kind() gast.NodeKind {
	return KindDisplayMath
}

// Dump implements ast.Node.Dump.
func (n *DisplayMath) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Expression": n.Expression,
	}, nil)
}
