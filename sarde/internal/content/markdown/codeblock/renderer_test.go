package codeblock

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	gmrenderer "github.com/yuin/goldmark/renderer"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

func renderMarkdown(input string) string {
	md := goldmark.New(
		goldmark.WithRendererOptions(
			gmhtml.WithUnsafe(),
			gmrenderer.WithNodeRenderers(
				util.Prioritized(NewRenderer(), 100),
			),
		),
	)
	var buf bytes.Buffer
	md.Convert([]byte(input), &buf)
	return buf.String()
}

func TestBasicCodeBlock(t *testing.T) {
	input := "```js\nconst x = 1;\n```"
	out := renderMarkdown(input)

	for _, want := range []string{`class="sarde-code-block"`, `class="chroma"`} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

func TestCodeBlockWithTitle(t *testing.T) {
	input := "```js title=\"app.js\"\nconst x = 1;\n```"
	out := renderMarkdown(input)

	for _, want := range []string{`class="sarde-code-title"`, "app.js"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n%s", want, out)
		}
	}
}

func TestTerminalFrame(t *testing.T) {
	input := "```bash\necho hello\n```"
	out := renderMarkdown(input)

	if !strings.Contains(out, "sarde-terminal-frame") {
		t.Errorf("output missing terminal-frame\n%s", out)
	}
}

func TestLineHighlighting(t *testing.T) {
	input := "```js {2}\nconst a = 1;\nconst b = 2;\nconst c = 3;\n```"
	out := renderMarkdown(input)

	if !strings.Contains(out, `class="sarde-code-line sarde-highlight"`) {
		t.Errorf("output missing highlighted line\n%s", out)
	}
}

func TestInsDelMarkers(t *testing.T) {
	input := "```js ins={1} del={2}\nconst a = 1;\nconst b = 2;\n```"
	out := renderMarkdown(input)

	if !strings.Contains(out, `class="sarde-code-line ins"`) {
		t.Errorf("output missing ins line\n%s", out)
	}
	if !strings.Contains(out, `class="sarde-code-line del"`) {
		t.Errorf("output missing del line\n%s", out)
	}
}

func TestShowLineNumbers(t *testing.T) {
	input := "```js showLineNumbers\nconst x = 1;\n```"
	out := renderMarkdown(input)

	if !strings.Contains(out, "sarde-show-line-numbers") {
		t.Errorf("output missing show-line-numbers\n%s", out)
	}
}

func TestCopyButton(t *testing.T) {
	input := "```js\nconst x = 1;\n```"
	out := renderMarkdown(input)

	if !strings.Contains(out, `class="sarde-copy-btn"`) {
		t.Errorf("output missing copy-btn\n%s", out)
	}
}

func TestUnknownLanguage(t *testing.T) {
	input := "```unknownlang\nsome code here\n```"
	out := renderMarkdown(input)

	if !strings.Contains(out, `code-block`) {
		t.Errorf("output missing code-block for unknown language\n%s", out)
	}
}

func TestEmptyCodeBlock(t *testing.T) {
	input := "```\n```"
	out := renderMarkdown(input)

	// Should not panic and should produce some output.
	if out == "" {
		t.Error("expected non-empty output for empty code block")
	}
}

func TestMermaidCodeBlock(t *testing.T) {
	input := "```mermaid\ngraph TD;\nA-->B;\n```"
	out := renderMarkdown(input)

	if !strings.Contains(out, `class="sarde-mermaid"`) {
		t.Errorf("output missing mermaid class\n%s", out)
	}
	if strings.Contains(out, `class="chroma"`) {
		t.Errorf("mermaid block should not contain chroma class\n%s", out)
	}
}
