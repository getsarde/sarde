package card

import gast "github.com/yuin/goldmark/ast"

var KindCardBlock = gast.NewNodeKind("CardBlock")

type CardBlock struct {
	gast.BaseBlock
	Title   string
	Icon    string
	Variant string
}

func (n *CardBlock) Kind() gast.NodeKind { return KindCardBlock }

func (n *CardBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{"Title": n.Title, "Icon": n.Icon, "Variant": n.Variant}, nil)
}
