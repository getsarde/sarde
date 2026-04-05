package asset

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/disintegration/imaging"
)

// ImageProcessor handles image resize, LQIP generation, and variant production.
type ImageProcessor struct {
	Config  *config.ImageSettings
	Cache   *Cache
	DevMode bool
}

// ProcessedImage holds all data needed to render a responsive image.
type ProcessedImage struct {
	Src      string         // original/fallback src URL
	Alt      string         // alt text
	Width    int            // original width
	Height   int            // original height
	LQIP     string         // base64 data URI for blur-up placeholder
	Variants []ImageVariant // responsive variants with srcset
	Loading  string         // "lazy" or "eager"
}

// ProcessImage generates responsive variants and LQIP for a source image.
// Variant image files are saved to the cache directory (not the output directory).
// Use WriteProcessedImages to copy cached variants to the output directory after the writer runs.
// Returns variants, LQIP base64 string, and error.
// In DevMode, returns empty results (dimensions are already set by the enhancer).
func (p *ImageProcessor) ProcessImage(srcPath, outputDir string) ([]ImageVariant, string, error) {
	if p.DevMode {
		return nil, "", nil
	}

	// Read source file for hashing.
	srcData, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, "", fmt.Errorf("reading source image: %w", err)
	}
	srcHash := Fingerprint(srcData)

	// Build params string for cache key.
	params := p.paramsString()
	cacheKey := p.Cache.Key(srcHash, params)

	// Check cache.
	if cached, err := p.Cache.Get(cacheKey); err == nil && cached != nil {
		return cached.Variants, cached.LQIP, nil
	}

	// Open image.
	src, err := imaging.Open(srcPath)
	if err != nil {
		return nil, "", fmt.Errorf("opening image: %w", err)
	}
	bounds := src.Bounds()
	origWidth := bounds.Dx()

	// Determine output format from source extension.
	ext := strings.ToLower(filepath.Ext(srcPath))
	baseName := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))

	// Generate variants for each configured width.
	widths := p.Config.Widths
	if len(widths) == 0 {
		widths = []int{400, 800, 1200}
	}

	quality := p.Config.Quality
	if quality == 0 {
		quality = 80
	}

	// Write variant files to cache directory (not output — writer cleans output first).
	cacheImagesDir := filepath.Join(p.Cache.Dir, "variants")
	if err := os.MkdirAll(cacheImagesDir, 0o755); err != nil {
		return nil, "", err
	}

	var variants []ImageVariant
	for _, w := range widths {
		if w >= origWidth {
			continue // don't upscale
		}

		resized := imaging.Resize(src, w, 0, imaging.Lanczos)
		hash := hashImage(resized, ext, quality)
		outName := fmt.Sprintf("%s-%s-%dw%s", baseName, hash, w, ext)
		cachePath := filepath.Join(cacheImagesDir, outName)

		if err := saveImage(resized, cachePath, ext, quality); err != nil {
			return nil, "", fmt.Errorf("saving variant %dw: %w", w, err)
		}

		info, _ := os.Stat(cachePath)
		var fileSize int64
		if info != nil {
			fileSize = info.Size()
		}

		variants = append(variants, ImageVariant{
			Width:    w,
			Format:   strings.TrimPrefix(ext, "."),
			URL:      "/assets/images/" + outName,
			FileSize: fileSize,
		})
	}

	// Also include original size as a variant.
	if origWidth > 0 {
		hash := Fingerprint(srcData)
		outName := fmt.Sprintf("%s-%s-%dw%s", baseName, hash, origWidth, ext)
		cachePath := filepath.Join(cacheImagesDir, outName)

		if err := saveImage(src, cachePath, ext, quality); err != nil {
			return nil, "", fmt.Errorf("saving original variant: %w", err)
		}

		info, _ := os.Stat(cachePath)
		var fileSize int64
		if info != nil {
			fileSize = info.Size()
		}

		variants = append(variants, ImageVariant{
			Width:    origWidth,
			Format:   strings.TrimPrefix(ext, "."),
			URL:      "/assets/images/" + outName,
			FileSize: fileSize,
		})
	}

	// Generate LQIP.
	lqip := p.generateLQIP(src)

	// Store in cache.
	p.Cache.Put(cacheKey, &CacheEntry{
		SourceHash: srcHash,
		Params:     params,
		Variants:   variants,
		LQIP:       lqip,
	})

	return variants, lqip, nil
}

// WriteProcessedImages copies all cached image variants to the output directory.
// Call this after the writer has cleaned and written HTML files.
func (p *ImageProcessor) WriteProcessedImages(outputDir string) error {
	if p.DevMode {
		return nil
	}

	cacheImagesDir := filepath.Join(p.Cache.Dir, "variants")
	entries, err := os.ReadDir(cacheImagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no variants processed
		}
		return err
	}

	outImagesDir := filepath.Join(outputDir, "assets", "images")
	if err := os.MkdirAll(outImagesDir, 0o755); err != nil {
		return err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(cacheImagesDir, e.Name())
		dst := filepath.Join(outImagesDir, e.Name())

		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
	}

	return nil
}

// generateLQIP creates a tiny blurred JPEG placeholder, base64-encoded.
func (p *ImageProcessor) generateLQIP(src image.Image) string {
	// Resize to 20px wide.
	tiny := imaging.Resize(src, 20, 0, imaging.Lanczos)
	// Apply blur for smooth effect.
	blurred := imaging.Blur(tiny, 3)

	var buf bytes.Buffer
	jpeg.Encode(&buf, blurred, &jpeg.Options{Quality: 20})

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return "data:image/jpeg;base64," + encoded
}

func (p *ImageProcessor) paramsString() string {
	widths := p.Config.Widths
	if len(widths) == 0 {
		widths = []int{400, 800, 1200}
	}
	quality := p.Config.Quality
	if quality == 0 {
		quality = 80
	}
	return fmt.Sprintf("w=%v,q=%d", widths, quality)
}

// saveImage writes an image to disk in the specified format.
func saveImage(img image.Image, path, ext string, quality int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	switch ext {
	case ".png":
		return png.Encode(f, img)
	case ".jpg", ".jpeg":
		return jpeg.Encode(f, img, &jpeg.Options{Quality: quality})
	default:
		// Fallback to JPEG for unsupported formats.
		return jpeg.Encode(f, img, &jpeg.Options{Quality: quality})
	}
}

// hashImage computes a fingerprint of an in-memory image by encoding it.
func hashImage(img image.Image, ext string, quality int) string {
	var buf bytes.Buffer
	switch ext {
	case ".png":
		png.Encode(&buf, img)
	default:
		jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	}
	return Fingerprint(buf.Bytes())
}
