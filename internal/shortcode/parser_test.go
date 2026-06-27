package shortcode

import (
	"strings"
	"testing"
)

func TestParseParams_DoubleQuoted(t *testing.T) {
	params := ParseParams(` type="warning" title="Heads up"`)
	if params["type"] != "warning" {
		t.Errorf("expected type=warning, got %q", params["type"])
	}
	if params["title"] != "Heads up" {
		t.Errorf("expected title='Heads up', got %q", params["title"])
	}
}

func TestParseParams_SingleQuoted(t *testing.T) {
	params := ParseParams(` id='abc-123'`)
	if params["id"] != "abc-123" {
		t.Errorf("expected id=abc-123, got %q", params["id"])
	}
}

func TestParseParams_BareValues(t *testing.T) {
	params := ParseParams(` width=800 height=600`)
	if params["width"] != "800" {
		t.Errorf("expected width=800, got %q", params["width"])
	}
	if params["height"] != "600" {
		t.Errorf("expected height=600, got %q", params["height"])
	}
}

func TestParseParams_Empty(t *testing.T) {
	params := ParseParams("")
	if len(params) != 0 {
		t.Errorf("expected empty params, got %v", params)
	}
}

func TestParseParams_HyphenatedKeys(t *testing.T) {
	params := ParseParams(` data-src="image.png"`)
	if params["data-src"] != "image.png" {
		t.Errorf("expected data-src=image.png, got %q", params["data-src"])
	}
}

func TestStripCodeBlocks_FencedBacktick(t *testing.T) {
	input := "before\n```\n{{< alert />}}\n```\nafter"
	sanitized, blocks := stripCodeBlocks(input)

	if strings.Contains(sanitized, "{{< alert") {
		t.Error("shortcode syntax should be stripped from sanitized output")
	}
	if !strings.Contains(sanitized, "before") {
		t.Error("text before code block should be preserved")
	}
	if !strings.Contains(sanitized, "after") {
		t.Error("text after code block should be preserved")
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if !strings.Contains(blocks[0], "{{< alert") {
		t.Error("block should contain the original shortcode text")
	}
}

func TestStripCodeBlocks_FencedTilde(t *testing.T) {
	input := "before\n~~~\n{{< youtube id=\"abc\" />}}\n~~~\nafter"
	sanitized, blocks := stripCodeBlocks(input)

	if strings.Contains(sanitized, "{{< youtube") {
		t.Error("shortcode in tilde-fenced block should be stripped")
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
}

func TestStripCodeBlocks_MultipleFences(t *testing.T) {
	input := "text\n```go\ncode1\n```\nmiddle\n```\ncode2\n```\nend"
	sanitized, blocks := stripCodeBlocks(input)

	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if !strings.Contains(sanitized, "middle") {
		t.Error("text between code blocks should be preserved")
	}
}

func TestStripCodeBlocks_NoFences(t *testing.T) {
	input := "hello {{< alert />}} world"
	sanitized, blocks := stripCodeBlocks(input)

	if sanitized != input {
		t.Errorf("expected unchanged output, got %q", sanitized)
	}
	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestStripCodeBlocks_UnclosedFence(t *testing.T) {
	input := "before\n```\n{{< alert />}}\nno closing"
	sanitized, blocks := stripCodeBlocks(input)

	if strings.Contains(sanitized, "{{< alert") {
		t.Error("shortcode in unclosed fence should be stripped")
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block (unclosed fence treated as block), got %d", len(blocks))
	}
}

func TestRestoreCodeBlocks(t *testing.T) {
	original := "before\n```\ncode here\n```\nafter"
	sanitized, blocks := stripCodeBlocks(original)
	restored := restoreCodeBlocks(sanitized, blocks)

	if restored != original {
		t.Errorf("restored output doesn't match original.\nGot:  %q\nWant: %q", restored, original)
	}
}

func TestStripCodeBlocks_LongerFence(t *testing.T) {
	input := "before\n````\n{{< alert />}}\n````\nafter"
	sanitized, blocks := stripCodeBlocks(input)

	if strings.Contains(sanitized, "{{< alert") {
		t.Error("shortcode in 4-backtick fenced block should be stripped")
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
}

func TestRegexSelfClosing(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
	}{
		{`{{< youtube id="abc" />}}`, "youtube"},
		{`{{< hr />}}`, "hr"},
		{`{{<alert type="info"/>}}`, "alert"},
	}
	for _, tt := range tests {
		m := reSelfClosing.FindStringSubmatch(tt.input)
		if m == nil {
			t.Errorf("no match for %q", tt.input)
			continue
		}
		if m[1] != tt.wantName {
			t.Errorf("got name %q, want %q for input %q", m[1], tt.wantName, tt.input)
		}
	}
}

func TestRegexOpening(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
	}{
		{`{{< alert type="warning" >}}`, "alert"},
		{`{{< tabs >}}`, "tabs"},
		{`{{<code-block lang="go">}}`, "code-block"},
	}
	for _, tt := range tests {
		m := reOpening.FindStringSubmatch(tt.input)
		if m == nil {
			t.Errorf("no match for %q", tt.input)
			continue
		}
		if m[1] != tt.wantName {
			t.Errorf("got name %q, want %q for input %q", m[1], tt.wantName, tt.input)
		}
	}
}

func TestRegexClosing(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
	}{
		{`{{< /alert >}}`, "alert"},
		{`{{</tabs>}}`, "tabs"},
		{`{{< /code-block >}}`, "code-block"},
	}
	for _, tt := range tests {
		m := reClosing.FindStringSubmatch(tt.input)
		if m == nil {
			t.Errorf("no match for %q", tt.input)
			continue
		}
		if m[1] != tt.wantName {
			t.Errorf("got name %q, want %q for input %q", m[1], tt.wantName, tt.input)
		}
	}
}

func TestParseParams_EmptyQuotedValues(t *testing.T) {
	params := ParseParams(` a="" b=2 c='' d=raw`)
	if v, ok := params["a"]; !ok || v != "" {
		t.Errorf(`expected a="" present and empty, got %q (present=%v)`, v, ok)
	}
	if v, ok := params["c"]; !ok || v != "" {
		t.Errorf(`expected c='' present and empty, got %q (present=%v)`, v, ok)
	}
	if params["b"] != "2" {
		t.Errorf("expected b=2, got %q", params["b"])
	}
	if params["d"] != "raw" {
		t.Errorf("expected d=raw, got %q", params["d"])
	}
}
