package gallery

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func renderGalleryMarkdown(t *testing.T, md string) string {
	t.Helper()
	gm := goldmark.New(goldmark.WithExtensions(&Extension{}))
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	return buf.String()
}

// The alt attribute must come from the image's alt text, not the tooltip
// title in ![alt](src "title").
func TestRenderGallery_AltNotTitle(t *testing.T) {
	md := ":::gallery\n![My Cat](cat.jpg \"Photo tooltip\")\n:::\n"
	out := renderGalleryMarkdown(t, md)

	if !strings.Contains(out, `alt="My Cat"`) {
		t.Errorf("alt should be the alt text %q:\n%s", "My Cat", out)
	}
	if strings.Contains(out, `alt="Photo tooltip"`) {
		t.Errorf("alt must not be the tooltip title:\n%s", out)
	}
	if !strings.Contains(out, `src="cat.jpg"`) {
		t.Errorf("missing src:\n%s", out)
	}
}

func TestRenderGallery_AltOnly(t *testing.T) {
	md := ":::gallery\n![A dog](dog.jpg)\n:::\n"
	out := renderGalleryMarkdown(t, md)

	if !strings.Contains(out, `alt="A dog"`) {
		t.Errorf("alt should be %q:\n%s", "A dog", out)
	}
}

func TestRenderGallery_NoAltNoTitle(t *testing.T) {
	md := ":::gallery\n![](plain.jpg)\n:::\n"
	out := renderGalleryMarkdown(t, md)

	if !strings.Contains(out, `src="plain.jpg"`) {
		t.Errorf("missing src:\n%s", out)
	}
	if !strings.Contains(out, `alt=""`) {
		t.Errorf("alt should be empty:\n%s", out)
	}
}
