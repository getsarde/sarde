package genericdirective

import (
	"strings"
	"testing"
)

func TestClose_RecoveryKeepsSourceAtCloser(t *testing.T) {
	// The card inside is never closed; :::/example closes both, and the
	// example's Source must end right before the closer line.
	md := ":::example\n:::card[X]\n\nbody\n\n:::/example\nafter\n"
	out := render(t, testRegistry(t), md)
	if !strings.Contains(out, "sarde-card") || !strings.Contains(out, "body") {
		t.Fatalf("card missing:\n%s", out)
	}
	if !strings.Contains(out, "<pre class=\"src\">:::card[X]\n\nbody</pre>") {
		t.Errorf("Source must stop before :::/example:\n%s", out)
	}
	if strings.Contains(out, ":::/example") {
		t.Errorf("the closer must not render as text:\n%s", out)
	}
	if i, j := strings.LastIndex(out, "</div>"), strings.Index(out, "<p>after</p>"); j < i {
		t.Errorf("content after :::/example must be outside:\n%s", out)
	}
}

func TestClose_SameNameNestingKeepsOuterSource(t *testing.T) {
	md := ":::example\n:::example\n\ninner\n\n:::/example\n:::/example\n"
	out := render(t, testRegistry(t), md)
	if strings.Count(out, `<pre class="src">`) != 2 {
		t.Fatalf("expected two example blocks:\n%s", out)
	}
	// The outer Source holds the inner block verbatim, closer included.
	if !strings.Contains(out, "<pre class=\"src\">:::example\n\ninner\n\n:::/example</pre>") {
		t.Errorf("outer Source must contain the inner block:\n%s", out)
	}
	if !strings.Contains(out, "<pre class=\"src\">inner</pre>") {
		t.Errorf("inner Source must be just its body:\n%s", out)
	}
}

func TestClose_LeafNamedCloserIsCaseInsensitive(t *testing.T) {
	out := render(t, testRegistry(t), ":::rawbox\nraw\n:::/RawBox\nafter\n")
	if !strings.Contains(out, `<pre class="rawbox">raw</pre>`) {
		t.Fatalf("leaf body wrong:\n%s", out)
	}
	if strings.Contains(out, ":::/RawBox") {
		t.Errorf("the closer must not be part of the body:\n%s", out)
	}
}
