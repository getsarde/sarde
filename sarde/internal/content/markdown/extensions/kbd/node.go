package kbd

import gast "github.com/yuin/goldmark/ast"

var KindKbd = gast.NewNodeKind("Kbd")

// Kbd is an inline node for [[Ctrl+S]] keyboard shortcuts.
type Kbd struct {
	gast.BaseInline
	Keys string // raw key string e.g. "Ctrl+S"
}

func (n *Kbd) Kind() gast.NodeKind { return KindKbd }

func (n *Kbd) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{"Keys": n.Keys}, nil)
}
