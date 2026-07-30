package socialcards

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
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
		DateExplicit:   true,
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

func TestLighten(t *testing.T) {
	black := color.NRGBA{0, 0, 0, 255}
	lightened := lighten(black, 0.3)
	if lightened.R < 50 {
		t.Errorf("lightened black should be noticeably lighter, got R=%d", lightened.R)
	}

	white := color.NRGBA{255, 255, 255, 255}
	clamped := lighten(white, 0.5)
	if clamped.R != 255 || clamped.G != 255 || clamped.B != 255 {
		t.Errorf("lightening white should clamp at white, got %v", clamped)
	}
}

func TestGradientRowColor(t *testing.T) {
	darkBg := color.NRGBA{0x1a, 0x1a, 0x2e, 0xff}

	// Row 0 returns the base color exactly.
	if got := gradientRowColor(darkBg, 0, cardHeight, gradientAmount); got != darkBg {
		t.Errorf("row 0 = %v, want %v", got, darkBg)
	}

	// Dark backgrounds darken toward the bottom, where the light text sits.
	last := gradientRowColor(darkBg, cardHeight-1, cardHeight, gradientAmount)
	if last == darkBg {
		t.Error("last row should differ from the base color")
	}
	_, _, lBase := rgbToHSL(darkBg.R, darkBg.G, darkBg.B)
	_, _, lLast := rgbToHSL(last.R, last.G, last.B)
	if lLast >= lBase {
		t.Errorf("dark background should darken toward the bottom: base L=%f last L=%f", lBase, lLast)
	}

	// Light backgrounds lighten toward the bottom, where the dark text sits.
	lightBg := color.NRGBA{0xee, 0xee, 0xee, 0xff}
	lastLight := gradientRowColor(lightBg, cardHeight-1, cardHeight, gradientAmount)
	_, _, lLightBase := rgbToHSL(lightBg.R, lightBg.G, lightBg.B)
	_, _, lLightLast := rgbToHSL(lastLight.R, lastLight.G, lastLight.B)
	if lLightLast <= lLightBase {
		t.Errorf("light background should lighten toward the bottom: base L=%f last L=%f", lLightBase, lLightLast)
	}

	// Monotonic across rows (sampled): dark backgrounds only get darker.
	prev := 2.0
	for row := 0; row < cardHeight; row += 100 {
		c := gradientRowColor(darkBg, row, cardHeight, gradientAmount)
		_, _, l := rgbToHSL(c.R, c.G, c.B)
		if l > prev {
			t.Errorf("lightness regressed at row %d: %f > %f", row, l, prev)
		}
		prev = l
	}
}

func TestRenderCard_WithLogoAndWatermark(t *testing.T) {
	regFont, boldFont := loadTestFonts(t)
	faces, err := newFaceSet(regFont, boldFont)
	if err != nil {
		t.Fatal(err)
	}

	// Solid opaque synthetic mark, distinct from the background.
	logo := image.NewNRGBA(image.Rect(0, 0, logoDrawSize, logoDrawSize))
	fill := color.NRGBA{0, 255, 0, 255}
	for y := 0; y < logoDrawSize; y++ {
		for x := 0; x < logoDrawSize; x++ {
			logo.SetNRGBA(x, y, fill)
		}
	}

	bg := color.NRGBA{26, 26, 46, 255}
	params := CardParams{
		Title:            "Branded Card",
		Description:      "With a logo and watermark",
		SiteTitle:        "My Site",
		BgColor:          bg,
		AccentColor:      color.NRGBA{233, 69, 96, 255},
		TextColor:        color.NRGBA{255, 255, 255, 255},
		LogoImage:        logo,
		WatermarkImage:   logo,
		WatermarkOpacity: 0.07,
		Faces:            faces,
		BoldFont:         boldFont,
	}

	img := renderCard(params)
	if img.Bounds().Dx() != cardWidth || img.Bounds().Dy() != cardHeight {
		t.Fatalf("unexpected dimensions %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}

	// The pixel inside the logo box must no longer be the background color.
	got := img.NRGBAAt(padding+2, padding+2)
	if got == gradientRowColor(bg, padding+2, cardHeight, gradientAmount) {
		t.Error("logo position pixel still matches the background, logo not drawn")
	}
}

func TestRenderCard_MaxDensity_NoOverlap(t *testing.T) {
	regFont, boldFont := loadTestFonts(t)
	faces, err := newFaceSet(regFont, boldFont)
	if err != nil {
		t.Fatal(err)
	}

	logo := image.NewNRGBA(image.Rect(0, 0, logoDrawSize, logoDrawSize))
	params := CardParams{
		Title:          "An Exceptionally Long Page Title Designed To Occupy Three Full Wrapped Lines At The Largest Ladder Size Available",
		Description:    "A long description that will certainly need to wrap across the full two lines available to it in the new bottom-anchored layout of the redesigned social sharing card",
		SiteTitle:      "My Site",
		CollectionName: "Documentation",
		Date:           time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		DateExplicit:   true,
		BgColor:        color.NRGBA{26, 26, 46, 255},
		AccentColor:    color.NRGBA{233, 69, 96, 255},
		TextColor:      color.NRGBA{255, 255, 255, 255},
		LogoImage:      logo,
		Faces:          faces,
		BoldFont:       boldFont,
	}

	// Must not panic and must produce a full-size card even at max density.
	img := renderCard(params)
	if img == nil || img.Bounds().Dx() != cardWidth || img.Bounds().Dy() != cardHeight {
		t.Fatal("max-density card did not render at full size")
	}
}

func TestBackgroundRowColor_Override(t *testing.T) {
	bg := color.NRGBA{0x1a, 0x1a, 0x2e, 0xff}
	red := color.NRGBA{0xff, 0x00, 0x00, 0xff}
	blue := color.NRGBA{0x00, 0x00, 0xff, 0xff}

	// Nil override: the automatic contrast gradient.
	if got := backgroundRowColor(bg, nil, 0, cardHeight); got != bg {
		t.Errorf("nil override row 0 = %v, want %v", got, bg)
	}
	if got := backgroundRowColor(bg, nil, cardHeight-1, cardHeight); got != gradientRowColor(bg, cardHeight-1, cardHeight, gradientAmount) {
		t.Error("nil override must match the automatic gradient")
	}

	// Single color: solid fill, every row identical.
	for _, row := range []int{0, cardHeight / 2, cardHeight - 1} {
		if got := backgroundRowColor(bg, []color.NRGBA{red}, row, cardHeight); got != red {
			t.Errorf("solid override row %d = %v, want %v", row, got, red)
		}
	}

	// Two colors: row 0 is the first color exactly, the last row the second,
	// and the middle a blend of both.
	override := []color.NRGBA{red, blue}
	if got := backgroundRowColor(bg, override, 0, cardHeight); got != red {
		t.Errorf("gradient override row 0 = %v, want %v exactly", got, red)
	}
	if got := backgroundRowColor(bg, override, cardHeight-1, cardHeight); got != blue {
		t.Errorf("gradient override last row = %v, want %v", got, blue)
	}
	mid := backgroundRowColor(bg, override, cardHeight/2, cardHeight)
	if mid == red || mid == blue {
		t.Errorf("gradient override midpoint should blend, got %v", mid)
	}
}

func TestRenderCard_GradientOverride(t *testing.T) {
	regFont, boldFont := loadTestFonts(t)
	faces, err := newFaceSet(regFont, boldFont)
	if err != nil {
		t.Fatal(err)
	}

	red := color.NRGBA{0xff, 0x00, 0x00, 0xff}
	blue := color.NRGBA{0x00, 0x00, 0xff, 0xff}
	params := CardParams{
		Title:            "Override",
		SiteTitle:        "Site",
		BgColor:          color.NRGBA{26, 26, 46, 255},
		AccentColor:      color.NRGBA{233, 69, 96, 255},
		TextColor:        color.NRGBA{255, 255, 255, 255},
		GradientOverride: []color.NRGBA{red, blue},
		Faces:            faces,
		BoldFont:         boldFont,
	}

	img := renderCard(params)
	// The top-right corner is free of text, logo, and the accent strip.
	if got := img.NRGBAAt(cardWidth-1, 0); got != red {
		t.Errorf("top row = %v, want the first override color %v", got, red)
	}
	if got := img.NRGBAAt(cardWidth-1, cardHeight-accentStripH-1); got == red {
		t.Error("bottom rows should have blended away from the first color")
	}
}

func TestRenderCard_BgImageDrawn(t *testing.T) {
	regFont, boldFont := loadTestFonts(t)
	faces, err := newFaceSet(regFont, boldFont)
	if err != nil {
		t.Fatal(err)
	}

	green := color.NRGBA{0, 255, 0, 255}
	bgImg := image.NewNRGBA(image.Rect(0, 0, cardWidth, cardHeight))
	for y := 0; y < cardHeight; y++ {
		for x := 0; x < cardWidth; x++ {
			bgImg.SetNRGBA(x, y, green)
		}
	}

	params := CardParams{
		Title:       "With Background",
		SiteTitle:   "Site",
		BgColor:     color.NRGBA{26, 26, 46, 255},
		AccentColor: color.NRGBA{233, 69, 96, 255},
		TextColor:   color.NRGBA{255, 255, 255, 255},
		BgImage:     bgImg,
		Faces:       faces,
		BoldFont:    boldFont,
	}

	img := renderCard(params)
	// Zero BgImageOpacity defaults to fully opaque, covering the gradient.
	if got := img.NRGBAAt(cardWidth-1, 0); got != green {
		t.Errorf("bg image pixel = %v, want %v", got, green)
	}

	// A dimmed image blends with the gradient instead of replacing it.
	params.BgImageOpacity = 0.5
	dimmed := renderCard(params)
	if got := dimmed.NRGBAAt(cardWidth-1, 0); got == green {
		t.Error("dimmed bg image should blend with the gradient, not replace it")
	}
}

func TestResizeLogoMark_CustomSize(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 512, 512))
	if got := resizeLogoMark(src, 128).Bounds().Dx(); got != 128 {
		t.Errorf("size 128 mark width = %d, want 128", got)
	}
	if got := resizeLogoMark(src, 0).Bounds().Dx(); got != logoDrawSize {
		t.Errorf("size 0 must keep the default %d, got %d", logoDrawSize, got)
	}
}

func TestRenderCard_LargeLogoNoOverlap(t *testing.T) {
	regFont, boldFont := loadTestFonts(t)
	faces, err := newFaceSet(regFont, boldFont)
	if err != nil {
		t.Fatal(err)
	}

	// A 128px mark with max-density content: the branding-row clamp must
	// keep the layout intact (no panic, full-size card).
	logo := image.NewNRGBA(image.Rect(0, 0, 128, 128))
	params := CardParams{
		Title:          "An Exceptionally Long Page Title Designed To Occupy Three Full Wrapped Lines At The Largest Ladder Size Available",
		Description:    "A long description that will certainly need to wrap across the full two lines available to it under the enlarged logo mark",
		SiteTitle:      "My Site",
		CollectionName: "Documentation",
		Date:           time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
		DateExplicit:   true,
		BgColor:        color.NRGBA{26, 26, 46, 255},
		AccentColor:    color.NRGBA{233, 69, 96, 255},
		TextColor:      color.NRGBA{255, 255, 255, 255},
		LogoImage:      logo,
		Faces:          faces,
		BoldFont:       boldFont,
	}

	img := renderCard(params)
	if img == nil || img.Bounds().Dx() != cardWidth || img.Bounds().Dy() != cardHeight {
		t.Fatal("large-logo card did not render at full size")
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
