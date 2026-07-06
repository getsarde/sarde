package shortcode

import (
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/content/markdown"
	"github.com/getsarde/sarde/internal/engine"
)

// Regression tests for nested shortcodes. The old code rendered a paired
// shortcode's inner body through the markdown renderer BEFORE nested
// shortcodes were expanded; goldmark HTML-escapes "{{<" in text, so nested
// tags became inert escaped text no later pass could match. The original
// suite missed this because its mock renderer does not escape.

// escapingRenderer mimics the one goldmark behavior that triggered the bug:
// recognized raw-HTML tags pass through, but "{{<" never parses as a tag, so
// its "<" is escaped in text — permanently breaking the shortcode regexes.
type escapingRenderer struct{}

func (escapingRenderer) Render(md string) (engine.RenderResult, error) {
	s := strings.TrimSpace(md)
	s = strings.ReplaceAll(s, "{{<", "{{&lt;")
	return engine.RenderResult{HTML: "<p>" + s + "</p>"}, nil
}

func TestProcessor_NestedDifferentNames_EscapingRenderer(t *testing.T) {
	reg := newTestRegistry(t, map[string]string{
		"tabs": `<div class="tabs">{{ .Inner }}</div>`,
		"tab":  `<section class="tab">{{ .Inner }}</section>`,
	})
	proc := NewProcessor(reg)

	src := "{{< tabs >}}{{< tab >}}hi{{< /tab >}}{{< /tabs >}}"
	result, warnings := proc.Process(src, &engine.Page{}, &engine.SiteContext{}, escapingRenderer{})

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if !strings.Contains(result, `<section class="tab">`) {
		t.Errorf("nested shortcode was not expanded, got %q", result)
	}
	if strings.Contains(result, "{{<") || strings.Contains(result, "{{&lt;") {
		t.Errorf("literal shortcode syntax leaked into output: %q", result)
	}
}

// End-to-end with the real goldmark-based renderer, exactly as builds run it.
func TestProcessor_NestedDifferentNames_RealRenderer(t *testing.T) {
	reg := newTestRegistry(t, map[string]string{
		"callout": `<div class="callout">{{ .Inner }}</div>`,
		"badge":   `<span class="badge">{{ .Params.text }}</span>`,
	})
	proc := NewProcessor(reg)

	src := "{{< callout >}}Some **bold** text {{< badge text=\"new\" />}}\n\n{{< badge text=\"inner\" />}}{{< /callout >}}"
	result, warnings := proc.Process(src, &engine.Page{}, &engine.SiteContext{}, markdown.NewRenderer())

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	for _, want := range []string{
		`<div class="callout">`,
		`<span class="badge">new</span>`,
		`<span class="badge">inner</span>`,
		`<strong>bold</strong>`,
	} {
		if !strings.Contains(result, want) {
			t.Errorf("missing %q in output: %q", want, result)
		}
	}
	if strings.Contains(result, "{{<") || strings.Contains(result, "{{&lt;") {
		t.Errorf("literal shortcode syntax leaked into output: %q", result)
	}
}

// Paired-inside-paired with the real renderer (the tabs/tab docs pattern).
func TestProcessor_PairedInsidePaired_RealRenderer(t *testing.T) {
	reg := newTestRegistry(t, map[string]string{
		"tabs": `<div class="tabs">{{ .Inner }}</div>`,
		"tab":  `<section class="tab">{{ .Inner }}</section>`,
	})
	proc := NewProcessor(reg)

	src := "{{< tabs >}}\n{{< tab >}}first{{< /tab >}}\n{{< tab >}}second{{< /tab >}}\n{{< /tabs >}}"
	result, warnings := proc.Process(src, &engine.Page{}, &engine.SiteContext{}, markdown.NewRenderer())

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if got := strings.Count(result, `<section class="tab">`); got != 2 {
		t.Errorf("expected 2 expanded tab shortcodes, got %d: %q", got, result)
	}
	if strings.Contains(result, "{{<") || strings.Contains(result, "{{&lt;") {
		t.Errorf("literal shortcode syntax leaked into output: %q", result)
	}
}
