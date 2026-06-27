package steps

import gast "github.com/yuin/goldmark/ast"

// KindStepsBlock is the NodeKind for StepsBlock.
var KindStepsBlock = gast.NewNodeKind("StepsBlock")

// KindStepItem is the NodeKind for StepItem.
var KindStepItem = gast.NewNodeKind("StepItem")

// StepsBlock is an AST node representing a steps container.
type StepsBlock struct {
	gast.BaseBlock
}

// Kind implements ast.Node.Kind.
func (n *StepsBlock) Kind() gast.NodeKind {
	return KindStepsBlock
}

// Dump implements ast.Node.Dump.
func (n *StepsBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// StepItem is an AST node representing a single step within a StepsBlock.
type StepItem struct {
	gast.BaseBlock
	Title string
	Index int // 1-based step number
}

// Kind implements ast.Node.Kind.
func (n *StepItem) Kind() gast.NodeKind {
	return KindStepItem
}

// Dump implements ast.Node.Dump.
func (n *StepItem) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Title": n.Title,
	}, nil)
}
