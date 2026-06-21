package icon

import gast "github.com/yuin/goldmark/ast"

// KindIcon is the node kind for inline :icon[...] tokens.
var KindIcon = gast.NewNodeKind("Icon")

// Icon is an inline node for :icon[name attr="val"] tokens.
type Icon struct {
	gast.BaseInline
	Name  string
	Attrs map[string]string
}

func (n *Icon) Kind() gast.NodeKind { return KindIcon }

func (n *Icon) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{"Name": n.Name}, nil)
}
