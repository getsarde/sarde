package badge

import gast "github.com/yuin/goldmark/ast"

var KindBadge = gast.NewNodeKind("Badge")

type Badge struct {
	gast.BaseBlock
	BadgeType string
	Content   string
	Icon      string
}

func (n *Badge) Kind() gast.NodeKind { return KindBadge }

func (n *Badge) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Type":    n.BadgeType,
		"Content": n.Content,
	}, nil)
}
