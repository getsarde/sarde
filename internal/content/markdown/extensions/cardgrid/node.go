package cardgrid

import (
	"fmt"

	gast "github.com/yuin/goldmark/ast"
)

var KindCardGridBlock = gast.NewNodeKind("CardGridBlock")

type CardGridBlock struct {
	gast.BaseBlock
	Cols int
}

func (n *CardGridBlock) Kind() gast.NodeKind { return KindCardGridBlock }

func (n *CardGridBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{"Cols": fmt.Sprintf("%d", n.Cols)}, nil)
}
