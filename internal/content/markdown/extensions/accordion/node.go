package accordion

import (
	"fmt"

	gast "github.com/yuin/goldmark/ast"
)

var KindAccordionBlock = gast.NewNodeKind("AccordionBlock")

type AccordionBlock struct {
	gast.BaseBlock
	Independent bool
}

func (n *AccordionBlock) Kind() gast.NodeKind { return KindAccordionBlock }

func (n *AccordionBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Independent": fmt.Sprintf("%v", n.Independent),
	}, nil)
}
