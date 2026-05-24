package highlight

import gast "github.com/yuin/goldmark/ast"

var KindHighlight = gast.NewNodeKind("Highlight")

// Highlight is an inline node for ==highlighted text==.
type Highlight struct {
	gast.BaseInline
}

func (n *Highlight) Kind() gast.NodeKind { return KindHighlight }

func (n *Highlight) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
