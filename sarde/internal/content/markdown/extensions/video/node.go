package video

import gast "github.com/yuin/goldmark/ast"

var KindVideoBlock = gast.NewNodeKind("VideoBlock")

type VideoBlock struct {
	gast.BaseBlock
	URL      string
	Platform string
	VideoID  string
}

func (n *VideoBlock) Kind() gast.NodeKind { return KindVideoBlock }

func (n *VideoBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"URL":      n.URL,
		"Platform": n.Platform,
		"VideoID":  n.VideoID,
	}, nil)
}
