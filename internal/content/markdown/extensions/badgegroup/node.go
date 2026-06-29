package badgegroup

import gast "github.com/yuin/goldmark/ast"

var KindBadgeGroupBlock = gast.NewNodeKind("BadgeGroupBlock")

type BadgeGroupBlock struct {
	gast.BaseBlock
}

func (n *BadgeGroupBlock) Kind() gast.NodeKind { return KindBadgeGroupBlock }

func (n *BadgeGroupBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
