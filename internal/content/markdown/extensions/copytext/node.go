package copytext

import gast "github.com/yuin/goldmark/ast"

var KindCopyText = gast.NewNodeKind("CopyText")

type CopyText struct {
	gast.BaseInline
	Content string
}

func (n *CopyText) Kind() gast.NodeKind { return KindCopyText }

func (n *CopyText) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Content": n.Content,
	}, nil)
}
