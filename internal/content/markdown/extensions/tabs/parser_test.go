package tabs

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func renderTabsMarkdown(t *testing.T, md string) string {
	t.Helper()
	gm := goldmark.New(goldmark.WithExtensions(&Extension{}))
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	return buf.String()
}

var tabLabelAttr = regexp.MustCompile(`<button class="sarde-tab-button[^"]*"[^>]*data-tab-label="([^"]*)"`)

func tabLabels(html string) []string {
	var labels []string
	for _, m := range tabLabelAttr.FindAllStringSubmatch(html, -1) {
		labels = append(labels, m[1])
	}
	return labels
}

func assertLabels(t *testing.T, html string, want ...string) {
	t.Helper()
	got := tabLabels(html)
	if len(got) != len(want) {
		t.Fatalf("got %d tabs %v, want %d %v:\n%s", len(got), got, len(want), want, html)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("tab %d label = %q, want %q:\n%s", i, got[i], want[i], html)
		}
	}
}

// Blank-line separated markers are the documented form and must keep working.
func TestTabs_BlankLineSeparated(t *testing.T) {
	out := renderTabsMarkdown(t, ":::tabs\n\n== Alpha\n\none\n\n== Beta\n\ntwo\n\n:::\n")
	assertLabels(t, out, "Alpha", "Beta")
	for _, body := range []string{"one", "two"} {
		if !strings.Contains(out, body) {
			t.Errorf("body %q missing:\n%s", body, out)
		}
	}
}

// Markers not separated by blank lines land in a single paragraph. Every tab
// and every body line must survive the split (they used to be discarded).
func TestTabs_CompactNoBlankLines(t *testing.T) {
	out := renderTabsMarkdown(t, ":::tabs\n== Alpha\none\n== Beta\ntwo\n:::\n")
	assertLabels(t, out, "Alpha", "Beta")
	for _, body := range []string{"one", "two"} {
		if !strings.Contains(out, body) {
			t.Errorf("body %q lost:\n%s", body, out)
		}
	}
	if strings.Contains(out, "== Beta") {
		t.Errorf("literal marker leaked into output:\n%s", out)
	}
}

// The (icon="name") attribute block is stripped from the label in both forms.
func TestTabs_IconAttributeStrippedFromLabel(t *testing.T) {
	out := renderTabsMarkdown(t, ":::tabs\n== Alpha (icon=\"leaf\")\none\n== Beta (icon=\"flask-conical\")\ntwo\n:::\n")
	assertLabels(t, out, "Alpha", "Beta")
	if strings.Contains(out, "icon=") {
		t.Errorf("attribute block leaked into output:\n%s", out)
	}
}

// Content before the first marker becomes the documented implicit tab.
func TestTabs_ContentBeforeFirstMarker(t *testing.T) {
	out := renderTabsMarkdown(t, ":::tabs\nintro text\n== Alpha\none\n:::\n")
	assertLabels(t, out, "Tab 1", "Alpha")
	if !strings.Contains(out, "intro text") {
		t.Errorf("leading content lost:\n%s", out)
	}
}

// Back-to-back markers render an empty panel for the first, per the docs.
func TestTabs_ConsecutiveMarkers(t *testing.T) {
	out := renderTabsMarkdown(t, ":::tabs\n== Alpha\n== Beta\ntwo\n:::\n")
	assertLabels(t, out, "Alpha", "Beta")
	if strings.Count(out, "two") != 1 {
		t.Errorf("expected body once, got:\n%s", out)
	}
}

// A fenced code block is its own node and must pass through untouched.
func TestTabs_CodeBlockInPanel(t *testing.T) {
	out := renderTabsMarkdown(t, ":::tabs\n== Alpha\n```go\nfunc main() {}\n```\n== Beta\ntwo\n:::\n")
	assertLabels(t, out, "Alpha", "Beta")
	if !strings.Contains(out, "<pre") || !strings.Contains(out, "func main()") {
		t.Errorf("code block lost:\n%s", out)
	}
}

// A block with no markers keeps the documented single-tab fallback.
func TestTabs_NoMarkersFallsBackToTabOne(t *testing.T) {
	out := renderTabsMarkdown(t, ":::tabs\njust some text\n:::\n")
	assertLabels(t, out, "Tab 1")
	if !strings.Contains(out, "just some text") {
		t.Errorf("content lost:\n%s", out)
	}
}
