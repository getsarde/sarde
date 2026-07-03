package imagecompare

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func renderImageCompareMarkdown(t *testing.T, md string) string {
	t.Helper()
	gm := goldmark.New(goldmark.WithExtensions(&Extension{}))
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	return buf.String()
}

// A single image must render the "requires two images" error, not a slider
// with the one image duplicated on both sides.
func TestImageCompare_SingleImageErrors(t *testing.T) {
	out := renderImageCompareMarkdown(t, ":::image-compare\n![Before only](one.png)\n:::\n")
	if !strings.Contains(out, "sarde-image-compare-error") {
		t.Errorf("single image should render the two-images error:\n%s", out)
	}
	if strings.Contains(out, "sarde-image-compare-container") {
		t.Errorf("single image should not render a comparison slider:\n%s", out)
	}
}

// Two images render a slider with both distinct sources.
func TestImageCompare_TwoImagesRenderSlider(t *testing.T) {
	out := renderImageCompareMarkdown(t, ":::image-compare\n![B](before.png)\n![A](after.png)\n:::\n")
	if strings.Contains(out, "sarde-image-compare-error") {
		t.Errorf("two images should not error:\n%s", out)
	}
	if !strings.Contains(out, "before.png") || !strings.Contains(out, "after.png") {
		t.Errorf("both image sources must appear:\n%s", out)
	}
}
