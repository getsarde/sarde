package linkbuttongroup

import gast "github.com/yuin/goldmark/ast"

var KindLinkButtonGroupBlock = gast.NewNodeKind("LinkButtonGroupBlock")

type LinkButtonGroupBlock struct {
	gast.BaseBlock
}

func (n *LinkButtonGroupBlock) Kind() gast.NodeKind { return KindLinkButtonGroupBlock }

func (n *LinkButtonGroupBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
