package codegroup

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func renderCodeGroupMarkdown(t *testing.T, md string) string {
	t.Helper()
	gm := goldmark.New(goldmark.WithExtensions(&Extension{}))
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	return buf.String()
}

func TestRenderCodeGroup_Basic(t *testing.T) {
	md := ":::code-group\n```js [JavaScript]\nconsole.log(1)\n```\n```python\nprint(1)\n```\n:::\n"
	out := renderCodeGroupMarkdown(t, md)

	if !strings.Contains(out, "sarde-code-group") {
		t.Fatalf("output missing code group wrapper:\n%s", out)
	}
	if !strings.Contains(out, ">JavaScript</button>") {
		t.Errorf("output missing JavaScript tab label:\n%s", out)
	}
	if !strings.Contains(out, ">python</button>") {
		t.Errorf("output missing python tab label:\n%s", out)
	}
}

// A fenced block with no info string has a nil Info node; rendering must not
// panic and the tab label falls back to "code".
func TestRenderCodeGroup_BareFenceNoInfoString(t *testing.T) {
	md := ":::code-group\n```\nplain text\n```\n:::\n"
	out := renderCodeGroupMarkdown(t, md)

	if !strings.Contains(out, "sarde-code-group") {
		t.Fatalf("output missing code group wrapper:\n%s", out)
	}
	if !strings.Contains(out, ">code</button>") {
		t.Errorf("bare fence should fall back to label %q:\n%s", "code", out)
	}
	if !strings.Contains(out, "plain text") {
		t.Errorf("output missing code content:\n%s", out)
	}
}

func TestRenderCodeGroup_MixedBareAndLabeled(t *testing.T) {
	md := ":::code-group\n```go [Go]\nfmt.Println(1)\n```\n```\nno language\n```\n:::\n"
	out := renderCodeGroupMarkdown(t, md)

	if !strings.Contains(out, ">Go</button>") {
		t.Errorf("output missing Go tab label:\n%s", out)
	}
	if !strings.Contains(out, ">code</button>") {
		t.Errorf("output missing fallback tab label for bare fence:\n%s", out)
	}
}
