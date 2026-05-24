package socialcards

import (
	"image/color"
	"image/png"
	"bytes"
	"testing"
	"time"

	"golang.org/x/image/font/opentype"
)

func loadTestFonts(t *testing.T) (*opentype.Font, *opentype.Font) {
	t.Helper()
	reg, bold, err := loadFonts()
	if err != nil {
		t.Fatalf("loadFonts: %v", err)
	}
	return reg, bold
}

func TestRenderCard_ProducesPNG(t *testing.T) {
	regFont, boldFont := loadTestFonts(t)
	faces, err := newFaceSet(regFont, boldFont)
	if err != nil {
		t.Fatal(err)
	}

	params := CardParams{
		Title:       "Hello World",
		Description: "A test social card",
		SiteTitle:   "My Site",
		BgColor:     color.NRGBA{26, 26, 46, 255},
		AccentColor: color.NRGBA{233, 69, 96, 255},
		TextColor:   color.NRGBA{255, 255, 255, 255},
		Faces:       faces,
		BoldFont:    boldFont,
	}

	img := renderCard(params)

	if img.Bounds().Dx() != 1200 {
		t.Errorf("expected width 1200, got %d", img.Bounds().Dx())
	}
	if img.Bounds().Dy() != 630 {
		t.Errorf("expected height 630, got %d", img.Bounds().Dy())
	}

	// Verify it encodes as valid PNG.
	data, err := encodeImage(img, "png", 0)
	if err != nil {
		t.Fatalf("encodeImage: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("encoded PNG is empty")
	}

	// Verify PNG is decodable.
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	if decoded.Bounds().Dx() != 1200 || decoded.Bounds().Dy() != 630 {
		t.Errorf("decoded dimensions: %dx%d", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}
}

func TestRenderCard_LongTitle(t *testing.T) {
	regFont, boldFont := loadTestFonts(t)
	faces, err := newFaceSet(regFont, boldFont)
	if err != nil {
		t.Fatal(err)
	}

	params := CardParams{
		Title:       "This Is A Very Long Title That Should Wrap To Multiple Lines And Trigger Font Size Reduction For Readability",
		Description: "Some description text that also wraps across multiple lines when the content is long enough",
		SiteTitle:   "My Site",
		BgColor:     color.NRGBA{26, 26, 46, 255},
		AccentColor: color.NRGBA{233, 69, 96, 255},
		TextColor:   color.NRGBA{255, 255, 255, 255},
		Faces:       faces,
		BoldFont:    boldFont,
	}

	// Should not panic.
	img := renderCard(params)
	if img == nil {
		t.Fatal("renderCard returned nil")
	}
}

func TestRenderCard_EmptyDescription(t *testing.T) {
	regFont, boldFont := loadTestFonts(t)
	faces, err := newFaceSet(regFont, boldFont)
	if err != nil {
		t.Fatal(err)
	}

	params := CardParams{
		Title:       "Title Only",
		Description: "",
		SiteTitle:   "Site",
		BgColor:     color.NRGBA{26, 26, 46, 255},
		AccentColor: color.NRGBA{233, 69, 96, 255},
		TextColor:   color.NRGBA{255, 255, 255, 255},
		Faces:       faces,
		BoldFont:    boldFont,
	}

	img := renderCard(params)
	if img == nil {
		t.Fatal("renderCard returned nil")
	}
}

func TestRenderCard_WithDate(t *testing.T) {
	regFont, boldFont := loadTestFonts(t)
	faces, err := newFaceSet(regFont, boldFont)
	if err != nil {
		t.Fatal(err)
	}

	params := CardParams{
		Title:          "Blog Post",
		CollectionName: "Blog",
		Date:           time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		BgColor:        color.NRGBA{26, 26, 46, 255},
		AccentColor:    color.NRGBA{233, 69, 96, 255},
		TextColor:      color.NRGBA{255, 255, 255, 255},
		Faces:          faces,
		BoldFont:       boldFont,
	}

	img := renderCard(params)
	if img == nil {
		t.Fatal("renderCard returned nil")
	}
}

func TestRenderCard_JPEGFormat(t *testing.T) {
	regFont, boldFont := loadTestFonts(t)
	faces, err := newFaceSet(regFont, boldFont)
	if err != nil {
		t.Fatal(err)
	}

	params := CardParams{
		Title:       "JPEG Card",
		SiteTitle:   "Site",
		BgColor:     color.NRGBA{26, 26, 46, 255},
		AccentColor: color.NRGBA{233, 69, 96, 255},
		TextColor:   color.NRGBA{255, 255, 255, 255},
		Faces:       faces,
		BoldFont:    boldFont,
	}

	img := renderCard(params)
	data, err := encodeImage(img, "jpeg", 85)
	if err != nil {
		t.Fatalf("encodeImage jpeg: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("encoded JPEG is empty")
	}
}

func TestDarken(t *testing.T) {
	white := color.NRGBA{255, 255, 255, 255}
	darkened := darken(white, 0.3)

	// Darkening white by 0.3 should give a light gray
	if darkened.R > 200 {
		t.Errorf("darkened white should be noticeably darker, got R=%d", darkened.R)
	}
}

func TestResolveBackground_ConfigOverride(t *testing.T) {
	cfg := map[string]any{"bg_color": "#ff0000"}
	c := resolveBackground(cfg, nil)
	if c.R != 255 || c.G != 0 || c.B != 0 {
		t.Errorf("expected red, got %v", c)
	}
}

func TestResolveAccent_Fallback(t *testing.T) {
	c := resolveAccent(nil, nil)
	expected := color.NRGBA{0xe9, 0x45, 0x60, 0xff}
	if c != expected {
		t.Errorf("expected fallback coral, got %v", c)
	}
}
