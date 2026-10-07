package columns

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func renderColumns(t *testing.T, md string) string {
	t.Helper()
	gm := goldmark.New(goldmark.WithExtensions(&Extension{}))
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	return buf.String()
}

func TestClose_ColumnsCloserOverOpenColumn(t *testing.T) {
	out := renderColumns(t, ":::columns\n:::column\none\n:::/columns\nafter\n")
	if strings.Contains(out, ":::/columns") {
		t.Fatalf("closer rendered as text:\n%s", out)
	}
	if i, j := strings.LastIndex(out, "</div>"), strings.Index(out, "<p>after</p>"); j < i {
		t.Errorf(":::/columns must close the open column and the container:\n%s", out)
	}
}

func TestClose_ColumnCloserClosesOnlyTheColumn(t *testing.T) {
	out := renderColumns(t, ":::columns\n:::column\none\n:::/column\n:::column\ntwo\n:::/column\n:::/columns\nafter\n")
	if strings.Count(out, "one") != 1 || strings.Count(out, "two") != 1 {
		t.Fatalf("both columns must render:\n%s", out)
	}
	if strings.Contains(out, ":::/") {
		t.Errorf("no closer may render as text:\n%s", out)
	}
	if i, j := strings.LastIndex(out, "</div>"), strings.Index(out, "<p>after</p>"); j < i {
		t.Errorf("content after :::/columns must be outside:\n%s", out)
	}
}
