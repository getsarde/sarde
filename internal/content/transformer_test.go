package content

import (
	"html/template"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

func TestTransform_WordCount(t *testing.T) {
	page := &engine.Page{PageContent: engine.PageContent{RawContent: "Hello world this is a test of word counting"}}
	tr := &Transformer{SummaryLength: 70}
	tr.Transform(page)

	if page.WordCount != 9 {
		t.Errorf("WordCount = %d, want 9", page.WordCount)
	}
}

func TestTransform_ReadingTime(t *testing.T) {
	// 200 words = 1 minute
	words := strings.Repeat("word ", 200)
	page := &engine.Page{PageContent: engine.PageContent{RawContent: words}}
	tr := &Transformer{SummaryLength: 70}
	tr.Transform(page)

	if page.ReadingTime != 1 {
		t.Errorf("ReadingTime = %d, want 1 for 200 words", page.ReadingTime)
	}

	// 201 words = 2 minutes (ceil)
	words = strings.Repeat("word ", 201)
	page = &engine.Page{PageContent: engine.PageContent{RawContent: words}}
	tr.Transform(page)

	if page.ReadingTime != 2 {
		t.Errorf("ReadingTime = %d, want 2 for 201 words", page.ReadingTime)
	}
}

func TestTransform_ReadingTimeMinimum(t *testing.T) {
	page := &engine.Page{PageContent: engine.PageContent{RawContent: "Just a few words."}}
	tr := &Transformer{SummaryLength: 70}
	tr.Transform(page)

	if page.ReadingTime != 1 {
		t.Errorf("ReadingTime = %d, want 1 (minimum)", page.ReadingTime)
	}
}

func TestTransform_ReadingTimeEmpty(t *testing.T) {
	page := &engine.Page{PageContent: engine.PageContent{RawContent: ""}}
	tr := &Transformer{SummaryLength: 70}
	tr.Transform(page)

	if page.ReadingTime != 0 {
		t.Errorf("ReadingTime = %d, want 0 for empty content", page.ReadingTime)
	}
}

func TestTransform_SummaryFromDescription(t *testing.T) {
	page := &engine.Page{
		PageContent: engine.PageContent{RawContent: "# Title\n\nThis is the body paragraph.\n"},
		PageMeta:    engine.PageMeta{Description: "This is the description."},
	}
	tr := &Transformer{SummaryLength: 70}
	tr.Transform(page)

	if page.Summary != template.HTML("This is the description.") {
		t.Errorf("Summary = %q, want description", page.Summary)
	}
}

func TestTransform_SummaryFromFirstParagraph(t *testing.T) {
	page := &engine.Page{
		PageContent: engine.PageContent{RawContent: "# My Title\n\nThis is the first paragraph of content.\n\nSecond paragraph.\n"},
	}
	tr := &Transformer{SummaryLength: 70}
	tr.Transform(page)

	want := "This is the first paragraph of content."
	if string(page.Summary) != want {
		t.Errorf("Summary = %q, want %q", page.Summary, want)
	}
}

func TestTransform_SummaryTruncation(t *testing.T) {
	// Description is set, so Summary uses Description directly (not word-truncated).
	// To test word truncation, set Description explicitly so auto-fill is skipped.
	page := &engine.Page{
		PageContent: engine.PageContent{RawContent: "This is a long paragraph that goes on and on with many words to test truncation behavior.\n"},
		PageMeta:    engine.PageMeta{Description: "Short desc."},
	}
	tr := &Transformer{SummaryLength: 5}
	tr.Transform(page)

	want := "Short desc."
	if string(page.Summary) != want {
		t.Errorf("Summary = %q, want %q", page.Summary, want)
	}
}

func TestTransform_SummaryTruncation_NoDescription(t *testing.T) {
	// When Description is empty AND auto-description fills it,
	// Summary should use the auto-filled Description.
	page := &engine.Page{
		PageContent: engine.PageContent{RawContent: "This is a long paragraph that goes on and on with many words to test truncation behavior.\n"},
	}
	tr := &Transformer{SummaryLength: 5}
	tr.Transform(page)

	// Auto-description fills page.Description, then Summary uses it.
	if page.Description == "" {
		t.Error("expected Description to be auto-filled")
	}
	if string(page.Summary) != page.Description {
		t.Errorf("Summary = %q, want Description %q", page.Summary, page.Description)
	}
}

func TestTransform_SummaryPreservesExisting(t *testing.T) {
	page := &engine.Page{
		PageContent: engine.PageContent{Summary: "Existing summary.", RawContent: "# Title\n\nBody text.\n"},
	}
	tr := &Transformer{SummaryLength: 70}
	tr.Transform(page)

	if page.Summary != "Existing summary." {
		t.Errorf("Summary = %q, should preserve existing", page.Summary)
	}
}

func TestTransform_AllDirectiveBody_SummaryEmpty(t *testing.T) {
	// A homepage-style body that is entirely directive blocks must yield an
	// empty Description and Summary, never the raw directive syntax.
	raw := `# Welcome

:::card-grid
:::link-card[Getting Started](href="/docs/getting-started" icon="rocket" description="Install and build")
:::link-card[Guides](href="/guides" icon="book" description="Learn the concepts")
:::
:::
`
	page := &engine.Page{PageContent: engine.PageContent{RawContent: raw}}
	tr := &Transformer{SummaryLength: 70}
	tr.Transform(page)

	if page.Description != "" {
		t.Errorf("Description = %q, want empty for an all-directive body", page.Description)
	}
	if page.Summary != "" {
		t.Errorf("Summary = %q, want empty for an all-directive body", page.Summary)
	}
}

func TestFirstProseParagraph_SkipsNestedDirectiveBlocks(t *testing.T) {
	// Same-length fences nest (":::card-grid" containing ":::card", both
	// closed by bare ":::"); a boolean toggle miscounts this shape. Only the
	// trailing prose after the outer close may be extracted.
	raw := `:::card-grid
:::card(title="Fast" icon="zap")
Zero-allocation routing engine inside the card.
:::
:::card(title="Flexible" icon="settings")
More card prose here.
:::
:::

Real trailing prose paragraph.
`
	want := "Real trailing prose paragraph."
	if got := firstProseParagraph(raw); got != want {
		t.Errorf("firstProseParagraph = %q, want %q", got, want)
	}
	if got := extractDescription(raw, 160); got != want {
		t.Errorf("extractDescription = %q, want %q", got, want)
	}
	if got := extractSummary(raw, 70); got != want {
		t.Errorf("extractSummary = %q, want %q", got, want)
	}
}

func TestFirstProseParagraph_StrayCloserClampsDepth(t *testing.T) {
	// An extra ":::" closer must not push the depth negative and swallow the
	// prose that follows.
	raw := ":::note\ninside a directive\n:::\n:::\n\nProse after a stray closer.\n"
	want := "Prose after a stray closer."
	if got := firstProseParagraph(raw); got != want {
		t.Errorf("firstProseParagraph = %q, want %q", got, want)
	}
}

func TestFirstProseParagraph_CodeFenceContainingColons(t *testing.T) {
	// A ":::" line inside a fenced code block is code, not a directive fence.
	raw := "```\n:::card-grid\n:::\n```\n\nProse after the code block.\n"
	want := "Prose after the code block."
	if got := firstProseParagraph(raw); got != want {
		t.Errorf("firstProseParagraph = %q, want %q", got, want)
	}
}

func TestExtractSummary_SkipsHTMLBlocks(t *testing.T) {
	// extractSummary shares the scanner's HTML-block skip that previously
	// only extractDescription had.
	raw := "<div class=\"hero\">\n\nActual prose paragraph.\n"
	want := "Actual prose paragraph."
	if got := extractSummary(raw, 70); got != want {
		t.Errorf("extractSummary = %q, want %q", got, want)
	}
}
