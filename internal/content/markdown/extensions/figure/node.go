package figure

import gast "github.com/yuin/goldmark/ast"

var KindFigureBlock = gast.NewNodeKind("FigureBlock")

type FigureBlock struct {
	gast.BaseBlock
	Caption string
}

func (n *FigureBlock) Kind() gast.NodeKind { return KindFigureBlock }

func (n *FigureBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{"Caption": n.Caption}, nil)
}
