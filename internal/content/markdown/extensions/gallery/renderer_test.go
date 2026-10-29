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

// A literal "</script>" in an image destination must not terminate the inline
// lightbox <script> block (script-context breakout).
func TestRenderGallery_ScriptBreakoutEscaped(t *testing.T) {
	// A mid-destination "<" stays in the parsed image destination (a leading
	// "<" would trigger goldmark's angle-bracket destination form instead).
	md := ":::gallery\n![x](cat.jpg</script><script>alert(1))\n:::\n"
	out := renderGalleryMarkdown(t, md)

	// The inline lightbox script data must not contain a raw script tag that
	// the HTML parser would act on.
	idx := strings.Index(out, "var images =")
	if idx == -1 {
		t.Fatalf("no images array in output:\n%s", out)
	}
	arrayEnd := strings.Index(out[idx:], ";")
	if arrayEnd == -1 {
		t.Fatalf("unterminated images array:\n%s", out)
	}
	arrayLiteral := out[idx : idx+arrayEnd]
	if strings.Contains(arrayLiteral, "</script") || strings.Contains(arrayLiteral, "<script") {
		t.Errorf("raw script tag inside inline script data:\n%s", arrayLiteral)
	}
}

// Site-absolute image paths go through AssetURL in the grid and in the
// lightbox data; a relative path is left as written.
func TestRenderGallery_SiteAbsoluteUsesAssetURL(t *testing.T) {
	prefixSub := func(src string) string {
		if strings.HasPrefix(src, "/") {
			return "/sub" + src
		}
		return src
	}
	gm := goldmark.New(goldmark.WithExtensions(&Extension{AssetURL: prefixSub}))
	var buf bytes.Buffer
	md := ":::gallery\n![Leaf](/images/leaf.jpg)\n![Root](root.jpg)\n:::\n"
	if err := gm.Convert([]byte(md), &buf); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	out := buf.String()
	for _, want := range []string{`src="/sub/images/leaf.jpg"`, `"src":"/sub/images/leaf.jpg"`, `src="root.jpg"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in:\n%s", want, out)
		}
	}
	if strings.Contains(out, `"/images/leaf.jpg"`) {
		t.Errorf("a site-absolute path kept no base path:\n%s", out)
	}
}
