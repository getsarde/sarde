package sourceline

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

func render(t *testing.T, offset int, source string) string {
	t.Helper()
	md := goldmark.New(goldmark.WithExtensions(extension.GFM, &Extension{Marker: &Marker{Offset: offset}}))
	var buf bytes.Buffer
	if err := md.Convert([]byte(source), &buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestMarksBlocksWithTheirLine(t *testing.T) {
	src := strings.Join([]string{
		"Intro paragraph.", // 1
		"",
		"## Install", // 3
		"",
		"> Quoted", // 5
		"> text.",
		"",
		"- one", // 8
		"- two", // 9
		"",
		"| a | b |", // 11
		"| - | - |",
		"| 1 | 2 |",
		"",
		"```go", // 15
		"x := 1",
		"```",
		"",
		"Setext", // 19
		"------",
		"",
	}, "\n")
	html := render(t, 0, src)
	for _, want := range []string{
		`<p data-sd-line="1">Intro paragraph.</p>`,
		`<h2 data-sd-line="3">Install</h2>`,
		`<blockquote data-sd-line="5">`,
		`<p data-sd-line="5">Quoted`,
		`<ul data-sd-line="8">`,
		`<li data-sd-line="8">one</li>`,
		`<li data-sd-line="9">two</li>`,
		`<table data-sd-line="11">`,
		`<h2 data-sd-line="19">Setext</h2>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %s in:\n%s", want, html)
		}
	}
	if strings.Contains(html, `<pre data-sd-line`) || strings.Contains(html, `<code data-sd-line`) {
		t.Errorf("code blocks are not marked:\n%s", html)
	}
}

func TestOffsetAddsTheFrontmatterLines(t *testing.T) {
	html := render(t, 4, "Body.\n\n- a\n  - nested\n")
	for _, want := range []string{
		`<p data-sd-line="5">Body.</p>`,
		`<ul data-sd-line="7">`,
		`<li data-sd-line="8">nested</li>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %s in:\n%s", want, html)
		}
	}
}
