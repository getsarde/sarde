package asset

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/disintegration/imaging"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/outputpath"
	"github.com/getsarde/sarde/internal/workers"
	"github.com/gen2brain/webp"
	_ "golang.org/x/image/webp" // register WebP decoder for image.Decode
	"golang.org/x/sync/errgroup"
)

// ResizeOp specifies how an image should be resized.
type ResizeOp string

const (
	ResizeOpScale     ResizeOp = "scale"
	ResizeOpFitWidth  ResizeOp = "fit_width"
	ResizeOpFitHeight ResizeOp = "fit_height"
	ResizeOpFit       ResizeOp = "fit"
	ResizeOpFill      ResizeOp = "fill"
)

// ImageOptions controls per-image processing parameters.
// Zero values mean "use config defaults".
type ImageOptions struct {
	Op      ResizeOp
	Width   int
	Height  int
	Quality int
	Formats []string
}

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
func (p *ImageProcessor) ProcessImage(srcPath string, opts ImageOptions) ([]ImageVariant, string, error) {
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
	params := p.paramsString(opts)
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

	baseName := strings.TrimSuffix(filepath.Base(srcPath), filepath.Ext(srcPath))

	widths := effectiveWidths(p.Config, opts)
	quality := effectiveQuality(p.Config, opts)
	formats := effectiveFormats(p.Config, opts)

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
		if p.Config.MaxWidth > 0 && w > p.Config.MaxWidth {
			continue
		}

		resized := applyResizeOp(src, opts.Op, w, opts.Height)
		resizedBounds := resized.Bounds()

		for _, format := range formats {
			ext := formatToExt(format)
			hash := hashImage(resized, ext, quality)
			outName := fmt.Sprintf("%s-%s-%dw%s", baseName, hash, resizedBounds.Dx(), ext)
			cachePath := filepath.Join(cacheImagesDir, outName)

			if err := saveImage(resized, cachePath, ext, quality); err != nil {
				if errors.Is(err, ErrAVIFNotAvailable) {
					logAVIFWarning()
					continue
				}
				return nil, "", fmt.Errorf("saving variant %dw %s: %w", w, format, err)
			}

			info, _ := os.Stat(cachePath)
			var fileSize int64
			if info != nil {
				fileSize = info.Size()
			}

			variants = append(variants, ImageVariant{
				Width:    resizedBounds.Dx(),
				Height:   resizedBounds.Dy(),
				Format:   format,
				URL:      "/assets/images/" + outName,
				FileSize: fileSize,
			})
		}
	}

	// Also include original size as a variant (one per format).
	if origWidth > 0 {
		origHeight := bounds.Dy()
		for _, format := range formats {
			ext := formatToExt(format)
			hash := hashImage(src, ext, quality)
			outName := fmt.Sprintf("%s-%s-%dw%s", baseName, hash, origWidth, ext)
			cachePath := filepath.Join(cacheImagesDir, outName)

			if err := saveImage(src, cachePath, ext, quality); err != nil {
				if errors.Is(err, ErrAVIFNotAvailable) {
					logAVIFWarning()
					continue
				}
				return nil, "", fmt.Errorf("saving original variant %s: %w", format, err)
			}

			info, _ := os.Stat(cachePath)
			var fileSize int64
			if info != nil {
				fileSize = info.Size()
			}

			variants = append(variants, ImageVariant{
				Width:    origWidth,
				Height:   origHeight,
				Format:   format,
				URL:      "/assets/images/" + outName,
				FileSize: fileSize,
			})
		}
	}

	// Generate LQIP (skip when placeholder is explicitly disabled).
	var lqip string
	if p.Config.Placeholder != "none" {
		lqip = p.generateLQIP(src)
	}

	// Store in cache. A failed write is non-fatal (the variants were already
	// produced) but means this image re-processes every build, so say so.
	if err := p.Cache.Put(cacheKey, &CacheEntry{
		SourceHash: srcHash,
		Params:     params,
		Variants:   variants,
		LQIP:       lqip,
	}); err != nil {
		devlog.Warn("images", "failed to cache image metadata for %s: %v", srcPath, err)
	}

	return variants, lqip, nil
}

// WriteProcessedImages copies all cached image variants to the output directory.
// Call this after the writer has cleaned and written HTML files.
func (p *ImageProcessor) WriteProcessedImages(outputDir string, trackFn func(string)) (int, error) {
	return p.WriteProcessedImagesWithOptions(outputDir, trackFn, WriteOptions{Parallel: true})
}

// WriteProcessedImagesWithOptions copies all cached image variants to the output directory.
// Call this after the writer has cleaned and written HTML files.
func (p *ImageProcessor) WriteProcessedImagesWithOptions(outputDir string, trackFn func(string), opts WriteOptions) (int, error) {
	if p.DevMode {
		return 0, nil
	}

	cacheImagesDir := filepath.Join(p.Cache.Dir, "variants")
	entries, err := os.ReadDir(cacheImagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	type imageCopy struct {
		src string
		dst string
	}
	var copies []imageCopy
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := filepath.Join(cacheImagesDir, entry.Name())
		dst, err := outputpath.SafeJoin(outputDir, filepath.ToSlash(filepath.Join("assets", "images", entry.Name())))
		if err != nil {
			return 0, err
		}
		copies = append(copies, imageCopy{src: src, dst: dst})
	}

	writeOne := func(c imageCopy) error {
		data, err := os.ReadFile(c.src)
		if err != nil {
			return fmt.Errorf("reading processed image %s: %w", c.src, err)
		}
		if err := os.MkdirAll(filepath.Dir(c.dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(c.dst, data, 0o644); err != nil {
			return err
		}
		if trackFn != nil {
			trackFn(c.dst)
		}
		return nil
	}

	if !opts.Parallel || len(copies) < 2 {
		count := 0
		for _, c := range copies {
			if err := writeOne(c); err != nil {
				return 0, err
			}
			count++
		}
		return count, nil
	}

	limit := opts.WorkerCount
	if limit <= 0 {
		limit = workers.Limit(len(copies))
	}
	g := new(errgroup.Group)
	g.SetLimit(limit)
	for _, c := range copies {
		c := c
		g.Go(func() error {
			return writeOne(c)
		})
	}
	if err := g.Wait(); err != nil {
		return 0, err
	}

	return len(copies), nil
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

func (p *ImageProcessor) paramsString(opts ImageOptions) string {
	widths := effectiveWidths(p.Config, opts)
	quality := effectiveQuality(p.Config, opts)
	formats := effectiveFormats(p.Config, opts)
	op := opts.Op
	if op == "" {
		op = ResizeOpScale
	}
	// MaxWidth and Placeholder change the processing output, so they must be
	// part of the cache key — otherwise editing either setting between builds
	// serves stale variants/LQIPs from the on-disk cache.
	return fmt.Sprintf("w=%v,q=%d,f=%v,op=%s,h=%d,mw=%d,ph=%s",
		widths, quality, formats, op, opts.Height, p.Config.MaxWidth, p.Config.Placeholder)
}

// ParseImageOptionsFromQuery parses a query string like "width=800&op=fill&format=webp"
// into ImageOptions. Used by the resize_image template function.
func ParseImageOptionsFromQuery(s string) ImageOptions {
	var opts ImageOptions
	for _, pair := range strings.Split(s, "&") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key, val := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		switch key {
		case "op":
			opts.Op = ResizeOp(val)
		case "width":
			fmt.Sscanf(val, "%d", &opts.Width)
		case "height":
			fmt.Sscanf(val, "%d", &opts.Height)
		case "quality":
			fmt.Sscanf(val, "%d", &opts.Quality)
		case "format":
			opts.Formats = []string{val}
		}
	}
	return opts
}

func effectiveWidths(cfg *config.ImageSettings, opts ImageOptions) []int {
	if opts.Width > 0 {
		return []int{opts.Width}
	}
	if len(cfg.Widths) > 0 {
		return cfg.Widths
	}
	return []int{400, 800, 1200}
}

func effectiveQuality(cfg *config.ImageSettings, opts ImageOptions) int {
	if opts.Quality > 0 {
		return opts.Quality
	}
	if cfg.Quality > 0 {
		return cfg.Quality
	}
	return 80
}

func effectiveFormats(cfg *config.ImageSettings, opts ImageOptions) []string {
	if len(opts.Formats) > 0 {
		return opts.Formats
	}
	if len(cfg.Formats) > 0 {
		return cfg.Formats
	}
	return []string{"jpeg"}
}

// applyResizeOp resizes src according to the given operation.
// For width-based ops (scale, fit_width, default), height is derived from the aspect ratio.
// For fit, the image fits within the width x height bounding box.
// For fill, the image is cropped to exactly width x height.
func applyResizeOp(src image.Image, op ResizeOp, width, height int) image.Image {
	switch op {
	case ResizeOpFill:
		if height == 0 {
			height = scaleHeight(src, width)
		}
		return imaging.Fill(src, width, height, imaging.Center, imaging.Lanczos)
	case ResizeOpFit:
		if height == 0 {
			height = scaleHeight(src, width)
		}
		return imaging.Fit(src, width, height, imaging.Lanczos)
	case ResizeOpFitHeight:
		if height == 0 {
			// No target height given — degrade to a width-driven
			// proportional scale instead of producing a 0x0 image.
			return imaging.Resize(src, width, 0, imaging.Lanczos)
		}
		return imaging.Resize(src, 0, height, imaging.Lanczos)
	default: // ResizeOpScale, ResizeOpFitWidth, ""
		return imaging.Resize(src, width, 0, imaging.Lanczos)
	}
}

// scaleHeight computes the proportional height for a given target width.
func scaleHeight(src image.Image, width int) int {
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW == 0 {
		return 0
	}
	return (srcH * width) / srcW
}

// avifWarnOnce guards the one-time AVIF-unavailable warning. Image processing
// runs across concurrent workers, so a plain bool would be a data race.
var avifWarnOnce sync.Once

func logAVIFWarning() {
	avifWarnOnce.Do(func() {
		devlog.Warn("images", "AVIF format configured but encoder not available. Rebuild with: go build -tags avif. Skipping AVIF variants.")
	})
}

// formatToExt maps a format name to its file extension.
func formatToExt(format string) string {
	switch format {
	case "jpeg", "jpg":
		return ".jpg"
	case "png":
		return ".png"
	case "webp":
		return ".webp"
	case "avif":
		return ".avif"
	default:
		return ".jpg"
	}
}

// savePathLocks serializes saveImage per destination path. Concurrent
// ProcessImage calls for byte-identical sources compute the same output path
// (the name embeds a content hash); without serialization the writers race —
// on Windows a rename onto a target being renamed-over fails with access
// denied, and a shared temp name would interleave writes into one file.
var savePathLocks sync.Map // map[string]*sync.Mutex

// saveImage writes an image to disk in the specified format using an atomic
// tmp+rename so a crash or encode failure never leaves a corrupt file at
// path. Destination names are content-derived, so an already-existing path
// holds identical bytes and the write is skipped.
func saveImage(img image.Image, path, ext string, quality int) error {
	muAny, _ := savePathLocks.LoadOrStore(path, &sync.Mutex{})
	mu := muAny.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	if _, err := os.Stat(path); err == nil {
		return nil
	}

	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := f.Name()

	var encErr error
	switch ext {
	case ".png":
		encErr = png.Encode(f, img)
	case ".webp":
		encErr = webp.Encode(f, img, webp.Options{Quality: quality})
	case ".avif":
		encErr = encodeAVIF(f, img, quality)
	default:
		encErr = jpeg.Encode(f, img, &jpeg.Options{Quality: quality})
	}

	closeErr := f.Close()
	if encErr == nil {
		encErr = closeErr
	}
	if encErr != nil {
		os.Remove(tmpPath)
		return encErr
	}
	// CreateTemp uses 0600; restore the usual artifact permissions.
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

// hashImage computes a fingerprint of an in-memory image by encoding it.
func hashImage(img image.Image, ext string, quality int) string {
	var buf bytes.Buffer
	switch ext {
	case ".png":
		png.Encode(&buf, img)
	case ".webp":
		webp.Encode(&buf, img, webp.Options{Quality: quality})
	case ".avif":
		if err := encodeAVIF(&buf, img, quality); err != nil {
			jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
		}
	default:
		jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	}
	return Fingerprint(buf.Bytes())
}
