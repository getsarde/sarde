package markdown

import (
	"strings"
	"testing"
)

// Regression test: a standard ```mermaid fence must be claimed by Sarde's
// mermaid extension (class="sarde-mermaid") rather than falling through to
// Kazari's passthrough (<pre class="mermaid">), which nothing downstream
// (script injection, init.js, CSS) recognizes.
func TestRender_MermaidFence_UsesSardeMermaidBlock(t *testing.T) {
	r := NewRenderer()
	md := "```mermaid\ngraph TD\n  A-->B\n```\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if !strings.Contains(result.HTML, `class="sarde-mermaid`) {
		t.Errorf("expected sarde-mermaid wrapper, got: %s", result.HTML)
	}
	if strings.Contains(result.HTML, `<pre class="mermaid"`) {
		t.Errorf("mermaid fence fell through to Kazari passthrough: %s", result.HTML)
	}
	if !strings.Contains(result.HTML, "A--&gt;B") && !strings.Contains(result.HTML, "A-->B") {
		t.Errorf("diagram source missing from output: %s", result.HTML)
	}
}

// The legacy spelling must keep working.
func TestRender_SardeMermaidFence_StillSupported(t *testing.T) {
	r := NewRenderer()
	md := "```sarde-mermaid\ngraph TD\n  A-->B\n```\n"
	result, err := r.Render(md)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if !strings.Contains(result.HTML, `class="sarde-mermaid`) {
		t.Errorf("expected sarde-mermaid wrapper, got: %s", result.HTML)
	}
}
