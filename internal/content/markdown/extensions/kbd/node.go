package kbd

import (
	"fmt"

	gast "github.com/yuin/goldmark/ast"
)

var KindKbd = gast.NewNodeKind("Kbd")

type Kbd struct {
	gast.BaseInline
	Keys string
	Size string // "sm" | "lg" | ""
	Wide bool
}

func (n *Kbd) Kind() gast.NodeKind { return KindKbd }

func (n *Kbd) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Keys": n.Keys,
		"Size": n.Size,
		"Wide": fmt.Sprintf("%v", n.Wide),
	}, nil)
}
