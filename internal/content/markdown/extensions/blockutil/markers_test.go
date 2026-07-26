package blockutil

import (
	"regexp"
	"strings"
	"testing"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var testBoundary = regexp.MustCompile(`^==\s+(.+)`)

// buildParagraph makes a paragraph whose Lines() mirror how goldmark stores
// one segment per physical source line, including the trailing newline.
func buildParagraph(source string) (*ast.Paragraph, []byte) {
	src := []byte(source)
	para := ast.NewParagraph()
	start := 0
	for i, b := range src {
		if b == '\n' {
			para.Lines().Append(text.NewSegment(start, i+1))
			start = i + 1
		}
	}
	if start < len(src) {
		para.Lines().Append(text.NewSegment(start, len(src)))
	}
	return para, src
}

func bodyText(t *testing.T, seg MarkerSegment, source []byte) string {
	t.Helper()
	if seg.Body == nil {
		return ""
	}
	var sb strings.Builder
	lines := seg.Body.Lines()
	for i := 0; i < lines.Len(); i++ {
		s := lines.At(i)
		sb.Write(s.Value(source))
	}
	return strings.TrimSpace(sb.String())
}

func TestSplitParagraphAtMarkers_NoMarkers(t *testing.T) {
	para, src := buildParagraph("plain text\nmore text\n")
	if got := SplitParagraphAtMarkers(para, src, testBoundary); got != nil {
		t.Errorf("expected nil so the caller reuses the node, got %d segments", len(got))
	}
}

func TestSplitParagraphAtMarkers_CompactTwoMarkers(t *testing.T) {
	para, src := buildParagraph("== Alpha\none\n== Beta\ntwo\n")
	segs := SplitParagraphAtMarkers(para, src, testBoundary)
	if len(segs) != 2 {
		t.Fatalf("got %d segments, want 2", len(segs))
	}
	for i, want := range []struct{ marker, body string }{{"Alpha", "one"}, {"Beta", "two"}} {
		if !segs[i].IsMarker || segs[i].Marker != want.marker {
			t.Errorf("segment %d marker = %q (isMarker=%v), want %q", i, segs[i].Marker, segs[i].IsMarker, want.marker)
		}
		if got := bodyText(t, segs[i], src); got != want.body {
			t.Errorf("segment %d body = %q, want %q", i, got, want.body)
		}
	}
}

func TestSplitParagraphAtMarkers_ConsecutiveMarkers(t *testing.T) {
	para, src := buildParagraph("== Alpha\n== Beta\ntwo\n")
	segs := SplitParagraphAtMarkers(para, src, testBoundary)
	if len(segs) != 2 {
		t.Fatalf("got %d segments, want 2", len(segs))
	}
	if segs[0].Body != nil {
		t.Errorf("first marker should have an empty body, got %q", bodyText(t, segs[0], src))
	}
	if got := bodyText(t, segs[1], src); got != "two" {
		t.Errorf("second body = %q, want %q", got, "two")
	}
}

func TestSplitParagraphAtMarkers_LeadingBody(t *testing.T) {
	para, src := buildParagraph("intro\n== Alpha\none\n")
	segs := SplitParagraphAtMarkers(para, src, testBoundary)
	if len(segs) != 2 {
		t.Fatalf("got %d segments, want 2", len(segs))
	}
	if segs[0].IsMarker {
		t.Errorf("leading segment should not be a marker")
	}
	if got := bodyText(t, segs[0], src); got != "intro" {
		t.Errorf("leading body = %q, want %q", got, "intro")
	}
	if !segs[1].IsMarker || segs[1].Marker != "Alpha" {
		t.Errorf("second segment = %q, want marker Alpha", segs[1].Marker)
	}
}
