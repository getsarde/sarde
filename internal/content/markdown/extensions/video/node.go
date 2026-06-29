package video

import (
	"fmt"

	gast "github.com/yuin/goldmark/ast"
)

var KindVideoBlock = gast.NewNodeKind("VideoBlock")

type VideoBlock struct {
	gast.BaseBlock
	URL      string
	Platform string
	VideoID  string
	Title    string
	Ratio    string
	Autoplay bool
	Muted    bool
	Loop     bool
}

func (n *VideoBlock) Kind() gast.NodeKind { return KindVideoBlock }

func (n *VideoBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"URL":      n.URL,
		"Platform": n.Platform,
		"VideoID":  n.VideoID,
		"Title":    n.Title,
		"Ratio":    n.Ratio,
		"Autoplay": fmt.Sprintf("%v", n.Autoplay),
		"Muted":    fmt.Sprintf("%v", n.Muted),
		"Loop":     fmt.Sprintf("%v", n.Loop),
	}, nil)
}
