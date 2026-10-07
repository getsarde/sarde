package aside

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func renderAside(t *testing.T, md string) string {
	t.Helper()
	gm := goldmark.New(goldmark.WithExtensions(&Extension{}))
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	return buf.String()
}

// closedBeforeAfter reports whether every <aside> closed before <p>after</p>.
func closedBeforeAfter(html string) bool {
	i := strings.LastIndex(html, "</aside>")
	j := strings.Index(html, "<p>after</p>")
	return i >= 0 && j > i
}

func TestClose_NamedVariants(t *testing.T) {
	for _, closer := range []string{":::", ":::/note", ":::/NOTE", ":::/aside", "::::"} {
		out := renderAside(t, ":::note\nbody\n"+closer+"\nafter\n")
		if strings.Count(out, "<aside") != 1 || !closedBeforeAfter(out) || strings.Contains(out, closer) {
			t.Errorf("closer %q did not close the note:\n%s", closer, out)
		}
	}
}

func TestClose_MismatchedNameIsText(t *testing.T) {
	out := renderAside(t, ":::note\nbody\n:::/tip\n:::\nafter\n")
	if !strings.Contains(out, ":::/tip") {
		t.Errorf(":::/tip inside a note must render as text:\n%s", out)
	}
	if !closedBeforeAfter(out) {
		t.Errorf("the bare ::: must still close the note:\n%s", out)
	}
}

func TestClose_OuterNamedCloserRecoversInner(t *testing.T) {
	out := renderAside(t, ":::note\n:::tip\nin\n:::/note\nafter\n")
	if strings.Count(out, "<aside") != 2 || strings.Count(out, "</aside>") != 2 {
		t.Fatalf("expected two asides, both closed:\n%s", out)
	}
	if !closedBeforeAfter(out) {
		t.Errorf(":::/note must close both asides:\n%s", out)
	}
}

func TestClose_SameTypeNesting(t *testing.T) {
	out := renderAside(t, ":::note\n:::note\nin\n:::/note\nout\n:::/note\nafter\n")
	if strings.Count(out, "<aside") != 2 {
		t.Fatalf("expected two asides:\n%s", out)
	}
	inner := strings.Index(out, "in")
	firstClose := strings.Index(out, "</aside>")
	outText := strings.Index(out, "out")
	if !(inner < firstClose && firstClose < outText) {
		t.Errorf("first :::/note must close only the inner note:\n%s", out)
	}
	if !closedBeforeAfter(out) {
		t.Errorf("second :::/note must close the outer note:\n%s", out)
	}
}

func TestClose_AsideAliasPicksInnermost(t *testing.T) {
	out := renderAside(t, ":::note\n:::tip\nin\n:::/aside\nout\n:::/aside\nafter\n")
	if strings.Count(out, "<aside") != 2 {
		t.Fatalf("expected two asides:\n%s", out)
	}
	firstClose := strings.Index(out, "</aside>")
	if outText := strings.Index(out, "out"); !(firstClose < outText) {
		t.Errorf("first :::/aside must close only the tip:\n%s", out)
	}
	if !closedBeforeAfter(out) {
		t.Errorf("second :::/aside must close the note:\n%s", out)
	}
}
