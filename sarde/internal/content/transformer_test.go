package content

import (
	"html/template"
	"strings"
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

func TestTransform_WordCount(t *testing.T) {
	page := &engine.Page{RawContent: "Hello world this is a test of word counting"}
	tr := &Transformer{SummaryLength: 70}
	tr.Transform(page)

	if page.WordCount != 9 {
		t.Errorf("WordCount = %d, want 9", page.WordCount)
	}
}

func TestTransform_ReadingTime(t *testing.T) {
	// 200 words = 1 minute
	words := strings.Repeat("word ", 200)
	page := &engine.Page{RawContent: words}
	tr := &Transformer{SummaryLength: 70}
	tr.Transform(page)

	if page.ReadingTime != 1 {
		t.Errorf("ReadingTime = %d, want 1 for 200 words", page.ReadingTime)
	}

	// 201 words = 2 minutes (ceil)
	words = strings.Repeat("word ", 201)
	page = &engine.Page{RawContent: words}
	tr.Transform(page)

	if page.ReadingTime != 2 {
		t.Errorf("ReadingTime = %d, want 2 for 201 words", page.ReadingTime)
	}
}

func TestTransform_ReadingTimeMinimum(t *testing.T) {
	page := &engine.Page{RawContent: "Just a few words."}
	tr := &Transformer{SummaryLength: 70}
	tr.Transform(page)

	if page.ReadingTime != 1 {
		t.Errorf("ReadingTime = %d, want 1 (minimum)", page.ReadingTime)
	}
}

func TestTransform_ReadingTimeEmpty(t *testing.T) {
	page := &engine.Page{RawContent: ""}
	tr := &Transformer{SummaryLength: 70}
	tr.Transform(page)

	if page.ReadingTime != 0 {
		t.Errorf("ReadingTime = %d, want 0 for empty content", page.ReadingTime)
	}
}

func TestTransform_SummaryFromDescription(t *testing.T) {
	page := &engine.Page{
		Description: "This is the description.",
		RawContent:  "# Title\n\nThis is the body paragraph.\n",
	}
	tr := &Transformer{SummaryLength: 70}
	tr.Transform(page)

	if page.Summary != template.HTML("This is the description.") {
		t.Errorf("Summary = %q, want description", page.Summary)
	}
}

func TestTransform_SummaryFromFirstParagraph(t *testing.T) {
	page := &engine.Page{
		RawContent: "# My Title\n\nThis is the first paragraph of content.\n\nSecond paragraph.\n",
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
		Description: "Short desc.",
		RawContent:  "This is a long paragraph that goes on and on with many words to test truncation behavior.\n",
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
		RawContent: "This is a long paragraph that goes on and on with many words to test truncation behavior.\n",
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
		Summary:    "Existing summary.",
		RawContent: "# Title\n\nBody text.\n",
	}
	tr := &Transformer{SummaryLength: 70}
	tr.Transform(page)

	if page.Summary != "Existing summary." {
		t.Errorf("Summary = %q, should preserve existing", page.Summary)
	}
}
