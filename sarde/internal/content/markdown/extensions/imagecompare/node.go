package imagecompare

import gast "github.com/yuin/goldmark/ast"

var KindImageCompareBlock = gast.NewNodeKind("ImageCompareBlock")

type ImageCompareBlock struct {
	gast.BaseBlock
	Label     string
	BeforeSrc string
	BeforeAlt string
	AfterSrc  string
	AfterAlt  string
	RawLines  []string
}

func (n *ImageCompareBlock) Kind() gast.NodeKind { return KindImageCompareBlock }

func (n *ImageCompareBlock) HasImages() bool {
	return n.BeforeSrc != "" && n.AfterSrc != ""
}

func (n *ImageCompareBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{"Label": n.Label}, nil)
}
