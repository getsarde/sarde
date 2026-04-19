package asset

import (
	"strings"
	"testing"
)

func TestRenderSrcset(t *testing.T) {
	variants := []ImageVariant{
		{Width: 800, Format: "jpeg", URL: "/img/hero-800w.jpg"},
		{Width: 400, Format: "jpeg", URL: "/img/hero-400w.jpg"},
		{Width: 1200, Format: "jpeg", URL: "/img/hero-1200w.jpg"},
	}

	srcset := RenderSrcset(variants)

	// Should be sorted by width ascending.
	if !strings.Contains(srcset, "/img/hero-400w.jpg 400w") {
		t.Errorf("missing 400w entry in srcset: %q", srcset)
	}
	idx400 := strings.Index(srcset, "400w")
	idx800 := strings.Index(srcset, "800w")
	idx1200 := strings.Index(srcset, "1200w")
	if idx400 > idx800 || idx800 > idx1200 {
		t.Errorf("srcset not sorted by width: %q", srcset)
	}
}

func TestRenderPicture_WithVariants(t *testing.T) {
	variants := []ImageVariant{
		{Width: 400, Format: "jpeg", URL: "/img/hero-400w.jpg"},
		{Width: 800, Format: "jpeg", URL: "/img/hero-800w.jpg"},
	}

	html := RenderPicture("/img/hero.jpg", "A hero image", 1200, 800, variants, "data:image/jpeg;base64,abc", true)

	if !strings.Contains(html, "<picture>") {
		t.Error("expected <picture> tag")
	}
	if !strings.Contains(html, "</picture>") {
		t.Error("expected closing </picture> tag")
	}
	if !strings.Contains(html, "<source") {
		t.Error("expected <source> element")
	}
	if !strings.Contains(html, "srcset=") {
		t.Error("expected srcset attribute")
	}
	if !strings.Contains(html, `alt="A hero image"`) {
		t.Error("expected alt text")
	}
	if !strings.Contains(html, `width="1200"`) {
		t.Error("expected width attribute")
	}
	if !strings.Contains(html, `loading="lazy"`) {
		t.Error("expected lazy loading")
	}
	if !strings.Contains(html, "background-image") {
		t.Error("expected LQIP background")
	}
}

func TestRenderPicture_NoVariants(t *testing.T) {
	html := RenderPicture("/img/photo.jpg", "A photo", 800, 600, nil, "", true)

	// Should fallback to simple <img>.
	if strings.Contains(html, "<picture>") {
		t.Error("should not contain <picture> without variants")
	}
	if !strings.Contains(html, "<img") {
		t.Error("expected <img> tag")
	}
	if !strings.Contains(html, `src="/img/photo.jpg"`) {
		t.Error("expected src attribute")
	}
}

func TestRenderPicture_EscapesAlt(t *testing.T) {
	html := RenderPicture("/img/test.jpg", `A "quoted" & <special> image`, 100, 100, nil, "", false)

	if strings.Contains(html, `"quoted"`) {
		t.Error("alt text should escape quotes")
	}
	if !strings.Contains(html, "&amp;") {
		t.Error("alt text should escape ampersands")
	}
}

func TestRenderPicture_MultiFormat(t *testing.T) {
	variants := []ImageVariant{
		{Width: 400, Format: "webp", URL: "/img/hero-400w.webp"},
		{Width: 800, Format: "webp", URL: "/img/hero-800w.webp"},
		{Width: 400, Format: "jpeg", URL: "/img/hero-400w.jpg"},
		{Width: 800, Format: "jpeg", URL: "/img/hero-800w.jpg"},
	}

	html := RenderPicture("/img/hero.jpg", "Hero", 1200, 800, variants, "", true)

	// Should have two <source> elements (webp and jpeg).
	sourceCount := strings.Count(html, "<source")
	if sourceCount != 2 {
		t.Errorf("source count = %d, want 2", sourceCount)
	}

	// WebP source should come before JPEG (higher priority).
	webpIdx := strings.Index(html, "image/webp")
	jpegIdx := strings.Index(html, "image/jpeg")
	if webpIdx > jpegIdx {
		t.Error("webp source should appear before jpeg")
	}
}
