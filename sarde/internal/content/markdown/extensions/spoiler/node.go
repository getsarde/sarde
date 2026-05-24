package spoiler

import gast "github.com/yuin/goldmark/ast"

var KindSpoilerInline = gast.NewNodeKind("SpoilerInline")

type SpoilerInline struct {
	gast.BaseInline
}

func (n *SpoilerInline) Kind() gast.NodeKind { return KindSpoilerInline }

func (n *SpoilerInline) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
