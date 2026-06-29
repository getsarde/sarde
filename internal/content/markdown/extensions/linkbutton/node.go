package linkbutton

import gast "github.com/yuin/goldmark/ast"

var KindLinkButtonBlock = gast.NewNodeKind("LinkButtonBlock")

type LinkButtonBlock struct {
	gast.BaseBlock
	Href          string
	Variant       string
	Icon          string
	IconPlacement string
	Size          string
	Content       string
	ColonCount    int
	Block         bool
	Disabled      bool
	Center        bool
	NewTab        *bool
	// HasLabel records whether the opening tag carried an explicit label, so
	// Continue knows to ignore body lines. It lives on the node because the
	// parser instance is shared across blocks (and across pages rendered
	// concurrently).
	HasLabel bool
}

func (n *LinkButtonBlock) Kind() gast.NodeKind { return KindLinkButtonBlock }

func (n *LinkButtonBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Href":          n.Href,
		"Variant":       n.Variant,
		"Icon":          n.Icon,
		"IconPlacement": n.IconPlacement,
		"Content":       n.Content,
	}, nil)
}
