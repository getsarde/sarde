package columns

import (
	"fmt"

	gast "github.com/yuin/goldmark/ast"
)

var KindColumnsBlock = gast.NewNodeKind("ColumnsBlock")

var KindColumnBlock = gast.NewNodeKind("ColumnBlock")

type ColumnsBlock struct {
	gast.BaseBlock
	Cols int
}

func (n *ColumnsBlock) Kind() gast.NodeKind { return KindColumnsBlock }

func (n *ColumnsBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Cols": fmt.Sprintf("%d", n.Cols),
	}, nil)
}

type ColumnBlock struct {
	gast.BaseBlock
}

func (n *ColumnBlock) Kind() gast.NodeKind { return KindColumnBlock }

func (n *ColumnBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
