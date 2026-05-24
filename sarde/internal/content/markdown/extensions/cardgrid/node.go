package cardgrid

import gast "github.com/yuin/goldmark/ast"

var KindCardGridBlock = gast.NewNodeKind("CardGridBlock")

type CardGridBlock struct {
	gast.BaseBlock
}

func (n *CardGridBlock) Kind() gast.NodeKind { return KindCardGridBlock }

func (n *CardGridBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
