package filetree

import gast "github.com/yuin/goldmark/ast"

var KindFileTreeBlock = gast.NewNodeKind("FileTreeBlock")

type FileTreeBlock struct {
	gast.BaseBlock
	Content string
}

func (n *FileTreeBlock) Kind() gast.NodeKind { return KindFileTreeBlock }

func (n *FileTreeBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}
