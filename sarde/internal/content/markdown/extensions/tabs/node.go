package tabs

import gast "github.com/yuin/goldmark/ast"

// KindTabsBlock is the NodeKind for TabsBlock.
var KindTabsBlock = gast.NewNodeKind("TabsBlock")

// KindTabItem is the NodeKind for TabItem.
var KindTabItem = gast.NewNodeKind("TabItem")

// TabsBlock is an AST node representing a tabs container.
type TabsBlock struct {
	gast.BaseBlock
}

// Kind implements ast.Node.Kind.
func (n *TabsBlock) Kind() gast.NodeKind {
	return KindTabsBlock
}

// Dump implements ast.Node.Dump.
func (n *TabsBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

// TabItem is an AST node representing a single tab within a TabsBlock.
type TabItem struct {
	gast.BaseBlock
	Label string
	Index int // 0-based tab index
}

// Kind implements ast.Node.Kind.
func (n *TabItem) Kind() gast.NodeKind {
	return KindTabItem
}

// Dump implements ast.Node.Dump.
func (n *TabItem) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Label": n.Label,
	}, nil)
}
