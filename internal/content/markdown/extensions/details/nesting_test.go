package details

import (
	"bytes"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/tabs"
	"github.com/yuin/goldmark"
)

func renderDetails(t *testing.T, md string) string {
	t.Helper()
	gm := goldmark.New(goldmark.WithExtensions(&Extension{}, &tabs.Extension{}))
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	return buf.String()
}

func TestNesting_SameNameWithNamedClosers(t *testing.T) {
	md := ":::details[Outer]\n:::details[Inner]\nhidden twice\n:::/details\nouter text\n:::/details\nafter\n"
	out := renderDetails(t, md)
	if strings.Count(out, "<details") != 2 || strings.Count(out, "</details>") != 2 {
		t.Fatalf("expected two closed details blocks:\n%s", out)
	}
	first := strings.Index(out, "</details>")
	if outer := strings.Index(out, "outer text"); outer < first {
		t.Errorf("the first :::/details must close the inner block only:\n%s", out)
	}
	if after := strings.Index(out, "<p>after</p>"); after < strings.LastIndex(out, "</details>") {
		t.Errorf("content after the second :::/details must be outside:\n%s", out)
	}
}

func TestNesting_SameNameWithThreeColonsAndBareClosers(t *testing.T) {
	// details.md used to claim this needs four colons on the outer fence.
	md := ":::details[Outer]\n:::details[Inner]\nin\n:::\n:::\nafter\n"
	out := renderDetails(t, md)
	if strings.Count(out, "<details") != 2 || strings.Count(out, "</details>") != 2 {
		t.Fatalf("expected two closed details blocks:\n%s", out)
	}
	if after := strings.Index(out, "<p>after</p>"); after < strings.LastIndex(out, "</details>") {
		t.Errorf("bare closers must close both blocks:\n%s", out)
	}
}

func TestNesting_KitchenSinkShapeUnchanged(t *testing.T) {
	md := ":::details[Tabs inside]\n:::tabs\n== A\na\n== B\nb\n:::\n:::\nafter\n"
	out := renderDetails(t, md)
	if !strings.Contains(out, "sarde-tab") {
		t.Fatalf("tabs missing:\n%s", out)
	}
	if after := strings.Index(out, "<p>after</p>"); after < strings.LastIndex(out, "</details>") {
		t.Errorf("same-length nesting with bare closers must keep working:\n%s", out)
	}
}
