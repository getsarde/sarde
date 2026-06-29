package linkcard

import gast "github.com/yuin/goldmark/ast"

var KindLinkCardBlock = gast.NewNodeKind("LinkCardBlock")

type LinkCardBlock struct {
	gast.BaseBlock
	Title       string
	Href        string
	Description string
	Icon        string
	Image       string
	NewTab      *bool
}

func (n *LinkCardBlock) Kind() gast.NodeKind { return KindLinkCardBlock }

func (n *LinkCardBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Title":       n.Title,
		"Href":        n.Href,
		"Description": n.Description,
		"Icon":        n.Icon,
		"Image":       n.Image,
	}, nil)
}
