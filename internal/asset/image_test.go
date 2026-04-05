package asset

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coderoo-dev/coderoo/internal/config"
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

	variants, lqip, err := p.ProcessImage("dummy.png", t.TempDir())
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

	variants, lqip, err := p.ProcessImage(srcPath, outDir)
	if err != nil {
		t.Fatalf("ProcessImage failed: %v", err)
	}

	// Should have 2 resized + 1 original = 3 variants.
	if len(variants) != 3 {
		t.Errorf("variant count = %d, want 3", len(variants))
	}

	// Check widths.
	widths := make(map[int]bool)
	for _, v := range variants {
		widths[v.Width] = true
		if v.Format != "png" {
			t.Errorf("format = %q, want png", v.Format)
		}
		if !strings.HasPrefix(v.URL, "/assets/images/") {
			t.Errorf("URL = %q, should start with /assets/images/", v.URL)
		}
	}

	// Verify files are in cache, then copy to output via WriteProcessedImages.
	if err := p.WriteProcessedImages(outDir); err != nil {
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
	outDir := t.TempDir()

	srcPath := createTestImage(t, srcDir, "small.png", 300, 200)

	p := &ImageProcessor{
		Config: &config.ImageSettings{
			Widths:  []int{400, 800, 1200},
			Quality: 80,
		},
		Cache:   NewCache(t.TempDir()),
		DevMode: false,
	}

	variants, _, err := p.ProcessImage(srcPath, outDir)
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
	outDir := t.TempDir()

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
	variants1, lqip1, err := p.ProcessImage(srcPath, outDir)
	if err != nil {
		t.Fatalf("first ProcessImage failed: %v", err)
	}

	// Second call — should hit cache.
	variants2, lqip2, err := p.ProcessImage(srcPath, outDir)
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
