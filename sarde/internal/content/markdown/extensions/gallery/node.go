package gallery

import gast "github.com/yuin/goldmark/ast"

var KindGalleryBlock = gast.NewNodeKind("GalleryBlock")

type GalleryImage struct {
	Src string
	Alt string
}

type GalleryBlock struct {
	gast.BaseBlock
	Label    string
	Images   []GalleryImage
	RawLines []string
}

func (n *GalleryBlock) Kind() gast.NodeKind { return KindGalleryBlock }

func (n *GalleryBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{"Label": n.Label}, nil)
}
