package plugin

import (
	"html/template"
	"strings"
	"testing"
)

func TestStripHTML_Basic(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<p>hello</p>", "hello"},
		{"no tags", "no tags"},
		{"<strong>bold</strong> and plain", "bold and plain"},
		{"", ""},
		// Adjacent elements get a separating space instead of running together.
		{"<h3>Getting Started</h3><p>Install now</p>", "Getting Started Install now"},
		// Whitespace collapses.
		{"<p>a\n\n  b</p>", "a b"},
	}
	for _, tt := range tests {
		if got := StripHTML(tt.input); got != tt.expected {
			t.Errorf("StripHTML(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestRenderedTextFallback(t *testing.T) {
	// Short content passes through stripped.
	got := RenderedTextFallback(template.HTML("<p>Just a sentence.</p>"), 160)
	if got != "Just a sentence." {
		t.Errorf("short content = %q", got)
	}

	// Long content truncates at a word boundary with an ellipsis.
	long := "<p>" + strings.Repeat("word ", 100) + "</p>"
	got = RenderedTextFallback(template.HTML(long), 40)
	if !strings.HasSuffix(got, "...") {
		t.Errorf("long content should end with ellipsis, got %q", got)
	}
	if len([]rune(got)) > 43 {
		t.Errorf("truncated length %d exceeds budget", len([]rune(got)))
	}
	if strings.Contains(got, "wor...") {
		t.Errorf("truncation should not split a word: %q", got)
	}

	// Entities are left encoded: the caller decodes exactly once.
	got = RenderedTextFallback(template.HTML("<p>Tips &amp; Tricks</p>"), 160)
	if got != "Tips &amp; Tricks" {
		t.Errorf("entities must stay encoded, got %q", got)
	}
}

func TestTrimTitlePrefix(t *testing.T) {
	tests := []struct {
		text, title, expected string
	}{
		{"Welcome Getting Started now", "Welcome", "Getting Started now"},
		{"Getting Started now", "Welcome", "Getting Started now"},
		{"anything", "", "anything"},
		{"Welcome", "Welcome", ""},
	}
	for _, tt := range tests {
		if got := TrimTitlePrefix(tt.text, tt.title); got != tt.expected {
			t.Errorf("TrimTitlePrefix(%q, %q) = %q, want %q", tt.text, tt.title, got, tt.expected)
		}
	}
}
