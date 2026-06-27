package mermaid

import gast "github.com/yuin/goldmark/ast"

var KindMermaidBlock = gast.NewNodeKind("MermaidBlock")

// MermaidBlock wraps mermaid diagram source for client-side rendering.
type MermaidBlock struct {
	gast.BaseBlock
	Source string
}

func (n *MermaidBlock) Kind() gast.NodeKind { return KindMermaidBlock }

func (n *MermaidBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
