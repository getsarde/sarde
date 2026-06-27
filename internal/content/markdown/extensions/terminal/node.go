package terminal

import gast "github.com/yuin/goldmark/ast"

var KindTerminalBlock = gast.NewNodeKind("TerminalBlock")

type TerminalBlock struct {
	gast.BaseBlock
	Content string
}

func (n *TerminalBlock) Kind() gast.NodeKind { return KindTerminalBlock }

func (n *TerminalBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
