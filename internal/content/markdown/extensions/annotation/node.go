package annotation

import gast "github.com/yuin/goldmark/ast"

var KindAnnotation = gast.NewNodeKind("Annotation")

type Annotation struct {
	gast.BaseInline
	Label       string
	Explanation string
	Style       string
}

func (n *Annotation) Kind() gast.NodeKind { return KindAnnotation }

func (n *Annotation) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Label":       n.Label,
		"Explanation": n.Explanation,
	}, nil)
}
