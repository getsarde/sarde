package blockutil

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MarkerSegment is one chunk of a paragraph split at boundary-marker lines.
// A marker segment carries the regex capture in Marker; a leading segment
// (body lines that appeared before the first marker) has IsMarker false and
// an empty Marker. Body holds the lines that followed, or nil when the
// marker was immediately followed by another marker or ended the paragraph.
type MarkerSegment struct {
	IsMarker bool
	Marker   string
	Body     *ast.Paragraph
}

// SplitParagraphAtMarkers splits para at every line matching boundary,
// returning the resulting sequence of marker and body chunks. It returns nil
// when no line matches, so callers can keep using the original node.
//
// Call this only from a block parser's Close: goldmark runs every block
// parser's Close during the block phase, before the separate inline pass
// consumes Paragraph.Lines(). Because of that ordering the line segments are
// still authoritative here, and the body paragraphs built below go through
// inline parsing normally, so markup inside them renders as HTML rather than
// literal text.
func SplitParagraphAtMarkers(para *ast.Paragraph, source []byte, boundary *regexp.Regexp) []MarkerSegment {
	lines := para.Lines()
	n := lines.Len()

	matched := false
	for i := 0; i < n; i++ {
		seg := lines.At(i)
		if boundary.MatchString(strings.TrimSpace(string(seg.Value(source)))) {
			matched = true
			break
		}
	}
	if !matched {
		return nil
	}

	var segments []MarkerSegment
	var pending []text.Segment

	flush := func() {
		if len(pending) == 0 {
			return
		}
		body := ast.NewParagraph()
		for _, seg := range pending {
			body.Lines().Append(seg)
		}
		if len(segments) == 0 {
			segments = append(segments, MarkerSegment{Body: body})
		} else {
			segments[len(segments)-1].Body = body
		}
		pending = nil
	}

	for i := 0; i < n; i++ {
		seg := lines.At(i)
		trimmed := strings.TrimSpace(string(seg.Value(source)))
		if m := boundary.FindStringSubmatch(trimmed); m != nil {
			flush()
			segments = append(segments, MarkerSegment{
				IsMarker: true,
				Marker:   strings.TrimSpace(m[1]),
			})
			continue
		}
		pending = append(pending, seg)
	}
	flush()

	return segments
}
