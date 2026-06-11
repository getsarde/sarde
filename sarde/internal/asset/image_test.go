package asset

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/frostybee/sarde/internal/config"
)

func createTestImage(t *testing.T, dir, name string, width, height int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with a solid color.
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating test image: %v", err)
	}
	defer f.Close()
	png.Encode(f, img)
	return path
}

func TestImageProcessor_DevMode(t *testing.T) {
	p := &ImageProcessor{
		Config:  &config.ImageSettings{},
		Cache:   NewCache(t.TempDir()),
		DevMode: true,
	}

	variants, lqip, err := p.ProcessImage("dummy.png", ImageOptions{})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}
	if len(variants) != 0 {
		t.Errorf("DevMode should produce no variants, got %d", len(variants))
	}
	if lqip != "" {
		t.Errorf("DevMode should produce no LQIP, got %q", lqip)
	}
}

func TestImageProcessor_ResizesImage(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()
	cacheDir := t.TempDir()

	srcPath := createTestImage(t, srcDir, "hero.png", 1600, 900)

	p := &ImageProcessor{
		Config: &config.ImageSettings{
			Widths:  []int{400, 800},
			Quality: 80,
		},
		Cache:   NewCache(cacheDir),
		DevMode: false,
	}

	variants, lqip, err := p.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	// Should have 2 resized + 1 original = 3 variants.
	if len(variants) != 3 {
		t.Errorf("variant count = %d, want 3", len(variants))
	}

	// Check widths. Default format is "jpeg" (from effectiveFormats).
	widths := make(map[int]bool)
	for _, v := range variants {
		widths[v.Width] = true
		if v.Format != "jpeg" {
			t.Errorf("format = %q, want jpeg", v.Format)
		}
		if !strings.HasPrefix(v.URL, "/assets/images/") {
			t.Errorf("URL = %q, should start with /assets/images/", v.URL)
		}
	}

	// Verify files are in cache, then copy to output via WriteProcessedImages.
	if _, err := p.WriteProcessedImages(outDir, nil); err != nil {
		t.Fatalf("WriteProcessedImages failed: %v", err)
	}
	for _, v := range variants {
		diskPath := filepath.Join(outDir, filepath.FromSlash(v.URL))
		if _, err := os.Stat(diskPath); err != nil {
			t.Errorf("variant file not found after WriteProcessedImages: %s", diskPath)
		}
	}

	if !widths[400] {
		t.Error("missing 400w variant")
	}
	if !widths[800] {
		t.Error("missing 800w variant")
	}
	if !widths[1600] {
		t.Error("missing original (1600w) variant")
	}

	// LQIP should be a base64 data URI.
	if !strings.HasPrefix(lqip, "data:image/jpeg;base64,") {
		t.Errorf("LQIP should be a base64 data URI, got prefix: %q", lqip[:30])
	}
}

func TestImageProcessor_NoUpscale(t *testing.T) {
	srcDir := t.TempDir()

	srcPath := createTestImage(t, srcDir, "small.png", 300, 200)

	p := &ImageProcessor{
		Config: &config.ImageSettings{
			Widths:  []int{400, 800, 1200},
			Quality: 80,
		},
		Cache:   NewCache(t.TempDir()),
		DevMode: false,
	}

	variants, _, err := p.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	// All configured widths are larger than 300px, so only the original should be output.
	if len(variants) != 1 {
		t.Errorf("variant count = %d, want 1 (original only)", len(variants))
	}
	if variants[0].Width != 300 {
		t.Errorf("variant width = %d, want 300", variants[0].Width)
	}
}

func TestImageProcessor_CacheHit(t *testing.T) {
	srcDir := t.TempDir()

	srcPath := createTestImage(t, srcDir, "cached.png", 1000, 600)

	p := &ImageProcessor{
		Config: &config.ImageSettings{
			Widths:  []int{400},
			Quality: 80,
		},
		Cache:   NewCache(t.TempDir()),
		DevMode: false,
	}

	// First call — processes.
	variants1, lqip1, err := p.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("first ProcessImage failed: %v", err)
	}

	// Second call — should hit cache.
	variants2, lqip2, err := p.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("second ProcessImage failed: %v", err)
	}

	if len(variants1) != len(variants2) {
		t.Errorf("variant count mismatch: %d vs %d", len(variants1), len(variants2))
	}
	if lqip1 != lqip2 {
		t.Error("LQIP mismatch between calls")
	}
}

func TestParamsString_IncludesAllFields(t *testing.T) {
	p := &ImageProcessor{
		Config: &config.ImageSettings{
			Widths:  []int{400, 800},
			Quality: 85,
		},
	}

	// Default opts should produce a stable string with all fields.
	s1 := p.paramsString(ImageOptions{})
	if s1 == "" {
		t.Fatal("paramsString returned empty string")
	}

	// Changing any field in opts should produce a different cache key.
	s2 := p.paramsString(ImageOptions{Op: ResizeOpFill, Height: 400})
	if s1 == s2 {
		t.Error("different opts should produce different paramsString")
	}

	// Per-image quality override changes the key.
	s3 := p.paramsString(ImageOptions{Quality: 90})
	if s1 == s3 {
		t.Error("quality override should produce different paramsString")
	}

	// Per-image formats override changes the key.
	s4 := p.paramsString(ImageOptions{Formats: []string{"webp"}})
	if s1 == s4 {
		t.Error("formats override should produce different paramsString")
	}

	// Per-image width override changes the key.
	s5 := p.paramsString(ImageOptions{Width: 600})
	if s1 == s5 {
		t.Error("width override should produce different paramsString")
	}
}

func TestApplyResizeOp_Scale(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	resized := applyResizeOp(src, ResizeOpScale, 800, 0)
	b := resized.Bounds()
	if b.Dx() != 800 {
		t.Errorf("width = %d, want 800", b.Dx())
	}
	// Height should be proportionally scaled: 900 * 800/1600 = 450
	if b.Dy() != 450 {
		t.Errorf("height = %d, want 450", b.Dy())
	}
}

func TestApplyResizeOp_FitWidth(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	resized := applyResizeOp(src, ResizeOpFitWidth, 400, 0)
	b := resized.Bounds()
	if b.Dx() != 400 {
		t.Errorf("width = %d, want 400", b.Dx())
	}
	if b.Dy() != 225 {
		t.Errorf("height = %d, want 225", b.Dy())
	}
}

func TestApplyResizeOp_Fill(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	resized := applyResizeOp(src, ResizeOpFill, 800, 800)
	b := resized.Bounds()
	if b.Dx() != 800 {
		t.Errorf("width = %d, want 800", b.Dx())
	}
	if b.Dy() != 800 {
		t.Errorf("height = %d, want 800", b.Dy())
	}
}

func TestApplyResizeOp_Fill_NoHeight(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	resized := applyResizeOp(src, ResizeOpFill, 800, 0)
	b := resized.Bounds()
	if b.Dx() != 800 {
		t.Errorf("width = %d, want 800", b.Dx())
	}
	// Height derived from aspect ratio: 900 * 800/1600 = 450
	if b.Dy() != 450 {
		t.Errorf("height = %d, want 450", b.Dy())
	}
}

func TestApplyResizeOp_Fit(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	resized := applyResizeOp(src, ResizeOpFit, 600, 600)
	b := resized.Bounds()
	// 1600x900 fitted into 600x600 → 600x337 (width-constrained)
	if b.Dx() > 600 {
		t.Errorf("width = %d, exceeds bounding box 600", b.Dx())
	}
	if b.Dy() > 600 {
		t.Errorf("height = %d, exceeds bounding box 600", b.Dy())
	}
	if b.Dx() != 600 {
		t.Errorf("width = %d, want 600 (width-constrained)", b.Dx())
	}
}

func TestApplyResizeOp_FitHeight(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	resized := applyResizeOp(src, ResizeOpFitHeight, 0, 450)
	b := resized.Bounds()
	if b.Dy() != 450 {
		t.Errorf("height = %d, want 450", b.Dy())
	}
	// Width should be proportionally scaled: 1600 * 450/900 = 800
	if b.Dx() != 800 {
		t.Errorf("width = %d, want 800", b.Dx())
	}
}

// fit_height without a target height must degrade to a width-driven scale,
// not produce a 0x0 image.
func TestApplyResizeOp_FitHeight_NoHeight(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	resized := applyResizeOp(src, ResizeOpFitHeight, 800, 0)
	b := resized.Bounds()
	if b.Dx() != 800 {
		t.Errorf("width = %d, want 800", b.Dx())
	}
	if b.Dy() != 450 {
		t.Errorf("height = %d, want 450 (proportional to width)", b.Dy())
	}
}

func TestApplyResizeOp_DefaultIsScale(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1600, 900))
	resized := applyResizeOp(src, "", 800, 0)
	b := resized.Bounds()
	if b.Dx() != 800 {
		t.Errorf("width = %d, want 800", b.Dx())
	}
	if b.Dy() != 450 {
		t.Errorf("height = %d, want 450 (same as scale)", b.Dy())
	}
}

func TestProcessImage_VariantsHaveHeight(t *testing.T) {
	srcDir := t.TempDir()
	srcPath := createTestImage(t, srcDir, "dims.png", 1600, 900)

	p := &ImageProcessor{
		Config: &config.ImageSettings{
			Widths:  []int{800},
			Quality: 80,
		},
		Cache:   NewCache(t.TempDir()),
		DevMode: false,
	}

	variants, _, err := p.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	for _, v := range variants {
		if v.Height == 0 {
			t.Errorf("variant %dw has Height=0, want non-zero", v.Width)
		}
	}

	// Check the 800w variant specifically: 900 * 800/1600 = 450
	for _, v := range variants {
		if v.Width == 800 && v.Height != 450 {
			t.Errorf("800w variant Height = %d, want 450", v.Height)
		}
	}
}

func TestProcessImage_FillOp(t *testing.T) {
	srcDir := t.TempDir()
	srcPath := createTestImage(t, srcDir, "fill.png", 1600, 900)

	p := &ImageProcessor{
		Config: &config.ImageSettings{
			Widths:  []int{400},
			Quality: 80,
		},
		Cache:   NewCache(t.TempDir()),
		DevMode: false,
	}

	variants, _, err := p.ProcessImage(srcPath, ImageOptions{Op: ResizeOpFill, Height: 400})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	// Should have 1 resized (400w) + 1 original = 2 variants.
	if len(variants) < 1 {
		t.Fatal("expected at least 1 variant")
	}

	// The 400w variant should be exactly 400x400 (fill crop).
	found := false
	for _, v := range variants {
		if v.Width == 400 {
			found = true
			if v.Height != 400 {
				t.Errorf("fill variant Height = %d, want 400", v.Height)
			}
		}
	}
	if !found {
		t.Error("missing 400w fill variant")
	}
}

func TestProcessImage_GeneratesWebPVariants(t *testing.T) {
	srcDir := t.TempDir()
	srcPath := createTestImage(t, srcDir, "webptest.png", 1600, 900)

	p := &ImageProcessor{
		Config: &config.ImageSettings{
			Widths:  []int{800},
			Quality: 80,
			Formats: []string{"jpeg", "webp"},
		},
		Cache:   NewCache(t.TempDir()),
		DevMode: false,
	}

	variants, _, err := p.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	// 1 width x 2 formats + 1 original x 2 formats = 4 variants.
	if len(variants) != 4 {
		t.Errorf("variant count = %d, want 4", len(variants))
	}

	formatCounts := make(map[string]int)
	for _, v := range variants {
		formatCounts[v.Format]++
	}

	if formatCounts["jpeg"] != 2 {
		t.Errorf("jpeg variant count = %d, want 2", formatCounts["jpeg"])
	}
	if formatCounts["webp"] != 2 {
		t.Errorf("webp variant count = %d, want 2", formatCounts["webp"])
	}
}

func TestSaveImage_WebP(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	path := filepath.Join(t.TempDir(), "test.webp")

	if err := saveImage(img, path, ".webp", 80); err != nil {
		t.Fatalf("saveImage webp failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading webp file: %v", err)
	}

	// WebP files start with "RIFF" magic bytes.
	if len(data) < 4 || string(data[:4]) != "RIFF" {
		t.Errorf("webp file does not start with RIFF header, got %q", data[:4])
	}
}

func TestProcessImage_WebPOnlyFormat(t *testing.T) {
	srcDir := t.TempDir()
	srcPath := createTestImage(t, srcDir, "wponly.png", 800, 600)

	p := &ImageProcessor{
		Config: &config.ImageSettings{
			Widths:  []int{400},
			Quality: 75,
			Formats: []string{"webp"},
		},
		Cache:   NewCache(t.TempDir()),
		DevMode: false,
	}

	variants, _, err := p.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	for _, v := range variants {
		if v.Format != "webp" {
			t.Errorf("format = %q, want webp", v.Format)
		}
		if !strings.HasSuffix(v.URL, ".webp") {
			t.Errorf("URL = %q, should end with .webp", v.URL)
		}
	}
}

func TestProcessImage_OptsFormatsOverride(t *testing.T) {
	srcDir := t.TempDir()
	srcPath := createTestImage(t, srcDir, "override.png", 1600, 900)

	p := &ImageProcessor{
		Config: &config.ImageSettings{
			Widths:  []int{800},
			Quality: 80,
			Formats: []string{"jpeg"},
		},
		Cache:   NewCache(t.TempDir()),
		DevMode: false,
	}

	// Per-image opts override config formats.
	variants, _, err := p.ProcessImage(srcPath, ImageOptions{Formats: []string{"webp"}})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	for _, v := range variants {
		if v.Format != "webp" {
			t.Errorf("format = %q, want webp (opts override)", v.Format)
		}
	}
}

func TestMaxWidth_FiltersVariants(t *testing.T) {
	srcDir := t.TempDir()
	srcPath := createTestImage(t, srcDir, "maxw.png", 2000, 1000)

	p := &ImageProcessor{
		Config: &config.ImageSettings{
			Widths:   []int{400, 800, 1200, 1600},
			Quality:  80,
			MaxWidth: 1000,
		},
		Cache:   NewCache(t.TempDir()),
		DevMode: false,
	}

	variants, _, err := p.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	for _, v := range variants {
		if v.Width == 2000 {
			continue // original is always included
		}
		if v.Width > 1000 {
			t.Errorf("variant %dw exceeds MaxWidth 1000", v.Width)
		}
	}

	// Should have 400w, 800w (filtered 1200, 1600) + original 2000w = 3.
	if len(variants) != 3 {
		t.Errorf("variant count = %d, want 3", len(variants))
	}
}

func TestPlaceholder_None(t *testing.T) {
	srcDir := t.TempDir()
	srcPath := createTestImage(t, srcDir, "nolqip.png", 800, 600)

	p := &ImageProcessor{
		Config: &config.ImageSettings{
			Widths:      []int{400},
			Quality:     80,
			Placeholder: "none",
		},
		Cache:   NewCache(t.TempDir()),
		DevMode: false,
	}

	_, lqip, err := p.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	if lqip != "" {
		t.Errorf("LQIP should be empty when placeholder=none, got %q", lqip[:30])
	}
}

func TestPlaceholder_Default(t *testing.T) {
	srcDir := t.TempDir()
	srcPath := createTestImage(t, srcDir, "lqip.png", 800, 600)

	p := &ImageProcessor{
		Config: &config.ImageSettings{
			Widths:  []int{400},
			Quality: 80,
		},
		Cache:   NewCache(t.TempDir()),
		DevMode: false,
	}

	_, lqip, err := p.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	if !strings.HasPrefix(lqip, "data:image/jpeg;base64,") {
		t.Errorf("LQIP should be generated by default, got %q", lqip)
	}
}

func TestParseImageOptionsFromQuery(t *testing.T) {
	tests := []struct {
		name   string
		query  string
		expect ImageOptions
	}{
		{
			name:  "all fields",
			query: "width=800&height=400&op=fill&quality=90&format=webp",
			expect: ImageOptions{
				Op: ResizeOpFill, Width: 800, Height: 400,
				Quality: 90, Formats: []string{"webp"},
			},
		},
		{
			name:   "empty string",
			query:  "",
			expect: ImageOptions{},
		},
		{
			name:  "width only",
			query: "width=600",
			expect: ImageOptions{
				Width: 600,
			},
		},
		{
			name:  "op and format",
			query: "op=fit&format=png",
			expect: ImageOptions{
				Op: ResizeOpFit, Formats: []string{"png"},
			},
		},
		{
			name:   "invalid values ignored",
			query:  "width=abc&quality=xyz",
			expect: ImageOptions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseImageOptionsFromQuery(tt.query)
			if got.Op != tt.expect.Op {
				t.Errorf("Op = %q, want %q", got.Op, tt.expect.Op)
			}
			if got.Width != tt.expect.Width {
				t.Errorf("Width = %d, want %d", got.Width, tt.expect.Width)
			}
			if got.Height != tt.expect.Height {
				t.Errorf("Height = %d, want %d", got.Height, tt.expect.Height)
			}
			if got.Quality != tt.expect.Quality {
				t.Errorf("Quality = %d, want %d", got.Quality, tt.expect.Quality)
			}
			if len(got.Formats) != len(tt.expect.Formats) {
				t.Errorf("Formats len = %d, want %d", len(got.Formats), len(tt.expect.Formats))
			} else {
				for i := range got.Formats {
					if got.Formats[i] != tt.expect.Formats[i] {
						t.Errorf("Formats[%d] = %q, want %q", i, got.Formats[i], tt.expect.Formats[i])
					}
				}
			}
		})
	}
}

func TestAVIF_StubSkipsVariants(t *testing.T) {
	srcDir := t.TempDir()
	srcPath := createTestImage(t, srcDir, "aviftest.png", 1600, 900)

	p := &ImageProcessor{
		Config: &config.ImageSettings{
			Widths:  []int{800},
			Quality: 80,
			Formats: []string{"avif"},
		},
		Cache:   NewCache(t.TempDir()),
		DevMode: false,
	}

	variants, _, err := p.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("ProcessImage should not fail for unavailable AVIF, got: %v", err)
	}

	for _, v := range variants {
		if v.Format == "avif" {
			t.Error("should not produce AVIF variants without the avif build tag")
		}
	}
}

func TestAVIF_MixedFormatsSkipsAVIF(t *testing.T) {
	srcDir := t.TempDir()
	srcPath := createTestImage(t, srcDir, "mixed.png", 1600, 900)

	p := &ImageProcessor{
		Config: &config.ImageSettings{
			Widths:  []int{800},
			Quality: 80,
			Formats: []string{"jpeg", "avif"},
		},
		Cache:   NewCache(t.TempDir()),
		DevMode: false,
	}

	variants, _, err := p.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	jpegCount := 0
	avifCount := 0
	for _, v := range variants {
		if v.Format == "jpeg" {
			jpegCount++
		}
		if v.Format == "avif" {
			avifCount++
		}
	}

	if jpegCount != 2 {
		t.Errorf("jpeg variant count = %d, want 2", jpegCount)
	}
	if avifCount != 0 {
		t.Errorf("avif variant count = %d, want 0 (stub build)", avifCount)
	}
}

func TestFormatToExt_AVIF(t *testing.T) {
	if got := formatToExt("avif"); got != ".avif" {
		t.Errorf("formatToExt(avif) = %q, want .avif", got)
	}
}

func TestEncodeAVIF_Stub(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	var buf bytes.Buffer
	err := encodeAVIF(&buf, img, 80)
	if err == nil {
		t.Error("stub encodeAVIF should return an error")
	}
	if !errors.Is(err, ErrAVIFNotAvailable) {
		t.Errorf("error = %v, want ErrAVIFNotAvailable", err)
	}
}
