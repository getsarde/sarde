package timeline

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func renderTimelineMarkdown(t *testing.T, md string) string {
	t.Helper()
	gm := goldmark.New(goldmark.WithExtensions(&Extension{}))
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	return buf.String()
}

// A "### Heading" entry marker must render its title (was empty because the
// heading has no inline children at block-parse time).
func TestTimeline_HeadingEntryHasTitle(t *testing.T) {
	out := renderTimelineMarkdown(t, ":::timeline\n### Kickoff\nProject started.\n:::\n")
	if !strings.Contains(out, "Kickoff") {
		t.Errorf("heading entry title missing:\n%s", out)
	}
	if !strings.Contains(out, "sarde-timeline-date") {
		t.Errorf("expected a date/title span for the heading entry:\n%s", out)
	}
}

// Body text on lines following "== Title" in the same paragraph must be
// inline-rendered (bold/links become HTML, not literal markdown).
func TestTimeline_SameParagraphBodyInlineRendered(t *testing.T) {
	md := ":::timeline\n== 2024-01-15\nReleased with **breaking changes** and a [guide](/docs/migrate).\n:::\n"
	out := renderTimelineMarkdown(t, md)

	if !strings.Contains(out, "<strong>breaking changes</strong>") {
		t.Errorf("body bold not inline-rendered:\n%s", out)
	}
	if !strings.Contains(out, `href="/docs/migrate"`) {
		t.Errorf("body link not inline-rendered:\n%s", out)
	}
	if strings.Contains(out, "**breaking changes**") {
		t.Errorf("literal markdown leaked into output:\n%s", out)
	}
}

// Entries not separated by blank lines land in a single paragraph. Each
// "== Title" must still start its own entry instead of being swallowed into
// the previous entry's body as literal text.
func TestTimeline_CompactConsecutiveEntries(t *testing.T) {
	md := ":::timeline\n== 2024-01-15\nReleased v1.\n== 2024-02-01\nReleased v2.\n:::\n"
	out := renderTimelineMarkdown(t, md)

	for _, want := range []string{"2024-01-15", "2024-02-01", "Released v1.", "Released v2."} {
		if strings.Count(out, want) != 1 {
			t.Errorf("expected %q exactly once:\n%s", want, out)
		}
	}
	if strings.Contains(out, "== 2024-02-01") {
		t.Errorf("literal marker leaked into output:\n%s", out)
	}
}
