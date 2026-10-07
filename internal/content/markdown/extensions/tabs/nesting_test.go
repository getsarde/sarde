package tabs

import (
	"bytes"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/content/markdown/extensions/aside"
	"github.com/yuin/goldmark"
)

func renderTabsWithAside(t *testing.T, md string) string {
	t.Helper()
	gm := goldmark.New(goldmark.WithExtensions(&Extension{}, &aside.Extension{}))
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	return buf.String()
}

// afterIsOutside reports whether <p>after</p> comes after the last closing
// tag of the tabs block, i.e. the block closed where the fence said.
func afterIsOutside(html string) bool {
	i := strings.LastIndex(html, "</div>")
	j := strings.Index(html, "<p>after</p>")
	return i >= 0 && j > i
}

func TestNesting_NamedCloserWithClosedInner(t *testing.T) {
	// Asides cannot interrupt a paragraph, so a blank line precedes :::note.
	md := ":::tabs\n== One\n\n:::note\nin\n:::\n:::/tabs\nafter\n"
	out := renderTabsWithAside(t, md)
	if !strings.Contains(out, "<aside") || !strings.Contains(out, "in") {
		t.Fatalf("inner aside missing:\n%s", out)
	}
	if !afterIsOutside(out) {
		t.Errorf("content after :::/tabs must be outside the tabs block:\n%s", out)
	}
}

func TestNesting_NamedCloserRecoversUnclosedInner(t *testing.T) {
	md := ":::tabs\n== One\n\n:::note\nin\n:::/tabs\nafter\n"
	out := renderTabsWithAside(t, md)
	if !strings.Contains(out, "<aside") {
		t.Fatalf("inner aside missing:\n%s", out)
	}
	if !afterIsOutside(out) {
		t.Errorf(":::/tabs must close the tabs block over the unclosed note:\n%s", out)
	}
	if strings.Contains(out, ":::/tabs") {
		t.Errorf("the closer must not render as text:\n%s", out)
	}
}

func TestNesting_LongerOuterFenceStillWorks(t *testing.T) {
	md := "::::tabs\n== One\n\n:::note\nin\n:::\n::::\nafter\n"
	out := renderTabsWithAside(t, md)
	if !strings.Contains(out, "<aside") || !afterIsOutside(out) {
		t.Errorf("four-colon outer fence must keep working:\n%s", out)
	}
}
