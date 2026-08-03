package build

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// createAssetFixtureSite creates a site with page bundles (images) and global CSS assets.
func createAssetFixtureSite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Welcome\n")

	// Blog collection with a page bundle containing an image.
	writeFixture(t, dir, "content/blog/_index.md", "---\ntitle: Blog\n---\n")
	writeFixture(t, dir, "content/blog/my-post/index.md", `---
title: My Post
date: 2025-06-01T00:00:00Z
---
# My Post

Here is a photo:

![A sunset](./hero.png)

And some text after the image.

![A second photo](./second.png)
`)

	// Create test PNG images (800x600) in the page bundle.
	createTestPNG(t, filepath.Join(dir, "content", "blog", "my-post", "hero.png"), 800, 600)
	createTestPNG(t, filepath.Join(dir, "content", "blog", "my-post", "second.png"), 800, 600)

	// Also add a non-image resource.
	writeFixture(t, dir, "content/blog/my-post/notes.txt", "some notes")

	// Global CSS asset.
	writeFixture(t, dir, "assets/css/main.css", `
body {
  font-family: sans-serif;
  color: #333;
}
h1 {
  font-size: 2rem;
}
`)

	return dir
}

func createTestPNG(t *testing.T, path string, width, height int) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 80, G: 120, B: 200, A: 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating test PNG: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encoding test PNG: %v", err)
	}
}

func TestBuild_AssetPipeline_EndToEnd(t *testing.T) {
	projDir := createAssetFixtureSite(t)
	cfg := config.Defaults()
	cfg.Head.CustomCSS = []string{"css/main.css"}
	themeCfg := buildThemeConfig()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if result.PageCount == 0 {
		t.Fatal("expected at least 1 page")
	}

	distDir := filepath.Join(projDir, "dist")

	// ── Page bundle image → <picture> element ──

	postHTML := readFixture(t, distDir, "blog/my-post/index.html")

	// The Goldmark imagerender should have generated a <picture> element.
	if !strings.Contains(postHTML, "<picture>") {
		t.Error("expected <picture> element in post HTML for bundle image")
	}
	if !strings.Contains(postHTML, "</picture>") {
		t.Error("expected closing </picture> tag")
	}
	if !strings.Contains(postHTML, "srcset=") {
		t.Error("expected srcset attribute in <picture>")
	}
	if !strings.Contains(postHTML, "/assets/images/") {
		t.Error("expected processed image URLs in /assets/images/")
	}
	if !strings.Contains(postHTML, `alt="A sunset"`) {
		t.Error("expected alt text preserved")
	}
	if !strings.Contains(postHTML, "background-image") {
		t.Error("expected LQIP background-image style")
	}
	// The first image per page is the likely LCP element and loads eagerly;
	// subsequent images are lazy.
	firstImg := strings.Index(postHTML, "<picture>")
	secondImg := strings.Index(postHTML[firstImg+1:], "<picture>")
	if secondImg == -1 {
		t.Fatal("expected two <picture> elements")
	}
	firstHTML := postHTML[firstImg : firstImg+1+secondImg]
	secondHTML := postHTML[firstImg+1+secondImg:]
	if strings.Contains(firstHTML, "loading=\"lazy\"") {
		t.Error("expected first image to load eagerly (LCP heuristic)")
	}
	if !strings.Contains(secondHTML, "loading=\"lazy\"") {
		t.Error("expected lazy loading attribute on the second image")
	}

	// ── Processed image files on disk ──

	imagesDir := filepath.Join(distDir, "assets", "images")
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		t.Fatalf("reading images dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected processed image files in dist/assets/images/")
	}

	// Should have multiple variants (400w + original 800w = at least 2).
	// Default config format is WebP (from embedded defaults).
	variantCount := 0
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".webp") || strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".png") {
			variantCount++
		}
	}
	if variantCount < 2 {
		t.Errorf("expected at least 2 image variants, got %d", variantCount)
	}

	// ── Bundle resource copying (non-image) ──

	assertFixtureFileExists(t, distDir, "blog/my-post/notes.txt")
	assertFixtureFileContains(t, distDir, "blog/my-post/notes.txt", "some notes")

	// ── CSS bundling ──

	cssDir := filepath.Join(distDir, "assets", "css")
	cssEntries, err := os.ReadDir(cssDir)
	if err != nil {
		t.Fatalf("reading CSS dir: %v", err)
	}
	if len(cssEntries) == 0 {
		t.Error("expected bundled CSS in dist/assets/css/")
	}

	// Should be fingerprinted (main.HASH.css).
	var bundledCSS string
	for _, e := range cssEntries {
		if strings.HasPrefix(e.Name(), "main.") && strings.HasSuffix(e.Name(), ".css") {
			bundledCSS = e.Name()
			break
		}
	}
	if bundledCSS == "" {
		t.Error("expected bundled main.*.css file")
	}
	// Fingerprinted names have 3 dot-separated parts: main.HASH.css
	if parts := strings.Split(bundledCSS, "."); len(parts) != 3 {
		t.Errorf("expected fingerprinted CSS name (3 parts), got %q (%d parts)", bundledCSS, len(parts))
	}

	// Bundled CSS should contain our styles (minified).
	cssContent := readFixture(t, cssDir, bundledCSS)
	if !strings.Contains(cssContent, "sans-serif") {
		t.Error("bundled CSS should contain original styles")
	}
}

func TestBuild_AssetPipeline_DevMode(t *testing.T) {
	projDir := createAssetFixtureSite(t)
	cfg := config.Defaults()
	themeCfg := buildThemeConfig()

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
		DevMode:     true,
	})

	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if result.PageCount == 0 {
		t.Fatal("expected at least 1 page")
	}

	distDir := filepath.Join(projDir, "dist")
	postHTML := readFixture(t, distDir, "blog/my-post/index.html")

	// In dev mode, images should NOT produce <picture> with srcset
	// (no resize/LQIP processing). Should fall back to simple <img>.
	if strings.Contains(postHTML, "srcset=") {
		t.Error("dev mode should not produce srcset (no image processing)")
	}

	// Should still have the image reference.
	if !strings.Contains(postHTML, "hero.png") {
		t.Error("expected image reference in dev mode output")
	}
}

func readFixture(t *testing.T, base, rel string) string {
	t.Helper()
	path := filepath.Join(base, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}
