package timeline

import gast "github.com/yuin/goldmark/ast"

var KindTimelineBlock = gast.NewNodeKind("TimelineBlock")
var KindTimelineItem = gast.NewNodeKind("TimelineItem")

type TimelineBlock struct {
	gast.BaseBlock
}

func (n *TimelineBlock) Kind() gast.NodeKind { return KindTimelineBlock }

func (n *TimelineBlock) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, nil, nil)
}

type TimelineItem struct {
	gast.BaseBlock
	Title    string
	BodyText string
}

func (n *TimelineItem) Kind() gast.NodeKind { return KindTimelineItem }

func (n *TimelineItem) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Title": n.Title,
	}, nil)
}
