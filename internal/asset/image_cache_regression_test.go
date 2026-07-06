package asset

import (
	"image"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/getsarde/sarde/internal/config"
)

// Regression test: the cache key must include MaxWidth. Reusing a cache dir
// across builds with a changed images.max_width used to serve the previous
// build's oversized variants.
func TestCacheKey_IncludesMaxWidth(t *testing.T) {
	srcDir := t.TempDir()
	cacheDir := t.TempDir()
	srcPath := createTestImage(t, srcDir, "shared.png", 2000, 1000)

	first := &ImageProcessor{
		Config: &config.ImageSettings{Widths: []int{400, 800, 1200}, Quality: 80},
		Cache:  NewCache(cacheDir),
	}
	variants, _, err := first.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("first ProcessImage: %v", err)
	}
	saw1200 := false
	for _, v := range variants {
		if v.Width == 1200 {
			saw1200 = true
		}
	}
	if !saw1200 {
		t.Fatal("setup: expected a 1200w variant with MaxWidth unset")
	}

	// Same cache dir, MaxWidth now caps at 800.
	second := &ImageProcessor{
		Config: &config.ImageSettings{Widths: []int{400, 800, 1200}, Quality: 80, MaxWidth: 800},
		Cache:  NewCache(cacheDir),
	}
	variants, _, err = second.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("second ProcessImage: %v", err)
	}
	for _, v := range variants {
		if v.Width == 2000 {
			continue // original is always included
		}
		if v.Width > 800 {
			t.Errorf("stale cache served %dw variant despite MaxWidth 800", v.Width)
		}
	}
}

// Regression test: the cache key must include Placeholder. Toggling
// images.placeholder between builds used to return the previously cached
// LQIP state regardless of the new setting.
func TestCacheKey_IncludesPlaceholder(t *testing.T) {
	srcDir := t.TempDir()
	cacheDir := t.TempDir()
	srcPath := createTestImage(t, srcDir, "ph.png", 800, 400)

	none := &ImageProcessor{
		Config: &config.ImageSettings{Widths: []int{400}, Quality: 80, Placeholder: "none"},
		Cache:  NewCache(cacheDir),
	}
	_, lqip, err := none.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("ProcessImage (none): %v", err)
	}
	if lqip != "" {
		t.Fatalf("setup: expected empty LQIP with placeholder none, got %d bytes", len(lqip))
	}

	blur := &ImageProcessor{
		Config: &config.ImageSettings{Widths: []int{400}, Quality: 80, Placeholder: "blur"},
		Cache:  NewCache(cacheDir),
	}
	_, lqip, err = blur.ProcessImage(srcPath, ImageOptions{})
	if err != nil {
		t.Fatalf("ProcessImage (blur): %v", err)
	}
	if lqip == "" {
		t.Error("stale cache served empty LQIP despite placeholder blur")
	}
}

// Regression test: concurrent ProcessImage calls for the same source used to
// share one fixed "<path>.tmp" temp file, letting interleaved writes ship a
// corrupt image. Every cached artifact must decode cleanly and no temp files
// may remain.
func TestProcessImage_ConcurrentSameSource(t *testing.T) {
	srcDir := t.TempDir()
	cacheDir := t.TempDir()
	srcPath := createTestImage(t, srcDir, "logo.png", 1200, 600)

	p := &ImageProcessor{
		Config: &config.ImageSettings{Widths: []int{400, 800}, Quality: 80},
		Cache:  NewCache(cacheDir),
	}

	const goroutines = 8
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := p.ProcessImage(srcPath, ImageOptions{}); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent ProcessImage: %v", err)
	}

	// Every produced image must decode; no temp files may survive.
	err := filepath.WalkDir(cacheDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasSuffix(path, ".tmp") {
			t.Errorf("leftover temp file: %s", path)
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".png", ".jpg", ".jpeg", ".webp":
			f, err := os.Open(path)
			if err != nil {
				t.Errorf("open %s: %v", path, err)
				return nil
			}
			defer f.Close()
			if _, _, err := image.Decode(f); err != nil {
				t.Errorf("corrupt cached image %s: %v", path, err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking cache dir: %v", err)
	}
}
