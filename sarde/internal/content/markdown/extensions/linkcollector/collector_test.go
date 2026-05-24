package linkcollector

import (
	"bytes"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/util"
)

func TestCollectorLinks(t *testing.T) {
	c := NewCollector()
	md := goldmark.New(
		goldmark.WithExtensions(&Extension{Collector: c}),
	)

	source := []byte(`# Hello

[link1](/docs/guide/)
[link2](/blog/post/)
[external](https://example.com)

Check [inline](/about/) link.
`)

	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		t.Fatal(err)
	}

	if len(c.Links) != 4 {
		t.Fatalf("collected %d links, want 4", len(c.Links))
	}

	want := []string{"/docs/guide/", "/blog/post/", "https://example.com", "/about/"}
	for i, w := range want {
		if c.Links[i].Href != w {
			t.Errorf("link[%d].Href = %q, want %q", i, c.Links[i].Href, w)
		}
		if c.Links[i].IsImage {
			t.Errorf("link[%d].IsImage = true, want false", i)
		}
	}
}

func TestCollectorImages(t *testing.T) {
	c := NewCollector()
	md := goldmark.New(
		goldmark.WithExtensions(&Extension{Collector: c}),
	)

	source := []byte(`![alt text](/images/hero.png)
![another](./photo.jpg)
`)

	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		t.Fatal(err)
	}

	if len(c.Links) != 2 {
		t.Fatalf("collected %d links, want 2", len(c.Links))
	}

	if c.Links[0].Href != "/images/hero.png" || !c.Links[0].IsImage {
		t.Errorf("link[0] = %+v, want Href=/images/hero.png IsImage=true", c.Links[0])
	}
	if c.Links[1].Href != "./photo.jpg" || !c.Links[1].IsImage {
		t.Errorf("link[1] = %+v, want Href=./photo.jpg IsImage=true", c.Links[1])
	}
}

func TestCollectorAutoLinks(t *testing.T) {
	c := NewCollector()
	md := goldmark.New(
		goldmark.WithExtensions(&Extension{Collector: c}),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)

	source := []byte(`Visit <https://example.com/page>.
`)

	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		t.Fatal(err)
	}

	if len(c.Links) != 1 {
		t.Fatalf("collected %d links, want 1", len(c.Links))
	}
	if c.Links[0].Href != "https://example.com/page" {
		t.Errorf("autolink Href = %q, want %q", c.Links[0].Href, "https://example.com/page")
	}
}

func TestCollectorReset(t *testing.T) {
	c := NewCollector()
	md := goldmark.New(
		goldmark.WithExtensions(&Extension{Collector: c}),
	)

	source := []byte(`[link](/first/)`)
	var buf bytes.Buffer
	md.Convert(source, &buf)

	if len(c.Links) != 1 {
		t.Fatalf("after first render: %d links, want 1", len(c.Links))
	}

	c.Reset()
	if len(c.Links) != 0 {
		t.Fatalf("after Reset: %d links, want 0", len(c.Links))
	}

	buf.Reset()
	source = []byte(`[a](/second/) [b](/third/)`)
	md.Convert(source, &buf)

	if len(c.Links) != 2 {
		t.Fatalf("after second render: %d links, want 2", len(c.Links))
	}
}

func TestCollectorDisabled(t *testing.T) {
	c := NewCollector()
	c.Enabled = false
	md := goldmark.New(
		goldmark.WithExtensions(&Extension{Collector: c}),
	)

	source := []byte(`[link](/page/)`)
	var buf bytes.Buffer
	md.Convert(source, &buf)

	if len(c.Links) != 0 {
		t.Errorf("disabled collector collected %d links, want 0", len(c.Links))
	}
}

func TestCollectorNoLinks(t *testing.T) {
	c := NewCollector()
	md := goldmark.New(
		goldmark.WithExtensions(&Extension{Collector: c}),
	)

	source := []byte(`Just plain text with **bold** and *italic*.`)
	var buf bytes.Buffer
	md.Convert(source, &buf)

	if len(c.Links) != 0 {
		t.Errorf("collected %d links from linkless content, want 0", len(c.Links))
	}
}

func TestCollectorPriority(t *testing.T) {
	c := NewCollector()
	ext := &Extension{Collector: c}

	md := goldmark.New()
	ext.Extend(md)

	// Verify the transformer is registered (it runs during Convert)
	source := []byte(`[test](/verify/)`)
	var buf bytes.Buffer
	md.Convert(source, &buf)

	if len(c.Links) != 1 {
		t.Errorf("collected %d links after manual Extend, want 1", len(c.Links))
	}
}

// Ensure util.Prioritized is importable (compile-time check).
var _ = util.Prioritized(nil, 0)
