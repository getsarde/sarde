package asset

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/outputpath"
	"github.com/frostybee/sarde/internal/workers"
	"golang.org/x/sync/errgroup"
)

// PipelineOptions configures the asset pipeline.
type PipelineOptions struct {
	ProjectDir string
	OutputDir  string
	Config     *config.SiteConfig
	ThemeName  string
	EmbeddedFS fs.FS
	DevMode    bool
}

// Pipeline orchestrates all asset processing.
type Pipeline struct {
	resolver     *Resolver
	enhancer     *ResourceEnhancer
	processor    *ImageProcessor
	bundler      *Bundler
	cache        *Cache
	manifest     *Manifest
	bundledFiles []BundledFile // CSS/JS files to write after writer cleans
	opts         PipelineOptions
}

// WriteOptions controls bounded parallel asset writes.
type WriteOptions struct {
	Parallel    bool
	WorkerCount int
}

// NewPipeline creates and initializes the asset pipeline.
func NewPipeline(opts PipelineOptions) *Pipeline {
	cache := NewCache(opts.ProjectDir)
	resolver := &Resolver{
		ProjectDir: opts.ProjectDir,
		ThemeName:  opts.ThemeName,
		EmbeddedFS: opts.EmbeddedFS,
	}

	return &Pipeline{
		resolver: resolver,
		enhancer: &ResourceEnhancer{
			DevMode: opts.DevMode,
		},
		processor: &ImageProcessor{
			Config:  &opts.Config.Images,
			Cache:   cache,
			DevMode: opts.DevMode,
		},
		bundler: &Bundler{
			Resolver:  resolver,
			DevMode:   opts.DevMode,
			Minify:    config.BoolVal(opts.Config.Build.Minify, true),
			OutputDir: opts.OutputDir,
		},
		cache:    cache,
		manifest: NewManifest(),
		opts:     opts,
	}
}

// ImageProcessor returns the image processor for use in markdown rendering.
func (p *Pipeline) ImageProcessor() *ImageProcessor {
	return p.processor
}

// OutputDir returns the configured output directory.
func (p *Pipeline) OutputDir() string {
	return p.opts.OutputDir
}

// EnhanceResources populates resource metadata for a page's bundle assets.
func (p *Pipeline) EnhanceResources(page *engine.Page) error {
	return p.enhancer.EnhancePageResources(page)
}

// Resolver returns the asset resolver for template function use.
func (p *Pipeline) Resolver() *Resolver {
	return p.resolver
}

// Manifest returns the asset manifest for template function use.
func (p *Pipeline) Manifest() *Manifest {
	return p.manifest
}

// BundleGlobalAssets processes CSS/JS entry points from the site config (head.custom_css, head.custom_js).
// Bundled files are written to the output directory and registered in the manifest.
func (p *Pipeline) BundleGlobalAssets() error {
	cfg := p.opts.Config

	// Bundle custom CSS files.
	for _, entry := range cfg.Head.CustomCSS {
		result, err := p.bundler.BundleCSS(entry)
		if err != nil {
			return fmt.Errorf("bundling CSS %q: %w", entry, err)
		}
		for _, f := range result.OutputFiles {
			p.manifest.Add(f.OriginalPath, ManifestEntry{
				OriginalPath: f.OriginalPath,
				OutputPath:   "assets/css/" + f.Name,
				OutputURL:    f.OutputURL,
				Hash:         Fingerprint(f.Content),
			})
			p.bundledFiles = append(p.bundledFiles, f)
		}
	}

	// Bundle custom JS files.
	for _, entry := range cfg.Head.CustomJS {
		result, err := p.bundler.BundleJS(entry)
		if err != nil {
			return fmt.Errorf("bundling JS %q: %w", entry, err)
		}
		for _, f := range result.OutputFiles {
			p.manifest.Add(f.OriginalPath, ManifestEntry{
				OriginalPath: f.OriginalPath,
				OutputPath:   "assets/js/" + f.Name,
				OutputURL:    f.OutputURL,
				Hash:         Fingerprint(f.Content),
			})
			p.bundledFiles = append(p.bundledFiles, f)
		}
	}

	return nil
}

// GlobalCSSURLs returns the output URLs of all CSS files bundled from head.custom_css.
func (p *Pipeline) GlobalCSSURLs() []string {
	var urls []string
	for _, f := range p.bundledFiles {
		if strings.HasSuffix(f.Name, ".css") {
			urls = append(urls, f.OutputURL)
		}
	}
	return urls
}

// WriteBundledFiles writes bundled CSS/JS files to the output directory.
// If trackFn is non-nil, each written path is reported for orphan tracking.
func (p *Pipeline) WriteBundledFiles(outputDir string, trackFn func(string)) error {
	return p.WriteBundledFilesWithOptions(outputDir, trackFn, WriteOptions{Parallel: true})
}

// WriteBundledFilesWithOptions writes bundled CSS/JS files to the output directory.
func (p *Pipeline) WriteBundledFilesWithOptions(outputDir string, trackFn func(string), opts WriteOptions) error {
	if !opts.Parallel || len(p.bundledFiles) < 2 {
		for _, f := range p.bundledFiles {
			if err := writeBundledFile(outputDir, f, trackFn); err != nil {
				return err
			}
		}
		return nil
	}
	limit := opts.WorkerCount
	if limit <= 0 {
		limit = workers.Limit(len(p.bundledFiles))
	}
	g := new(errgroup.Group)
	g.SetLimit(limit)
	for _, f := range p.bundledFiles {
		f := f
		g.Go(func() error {
			return writeBundledFile(outputDir, f, trackFn)
		})
	}
	return g.Wait()
}

// WriteProcessedImages copies cached image variants to the output directory.
// If trackFn is non-nil, each written path is reported for orphan tracking.
// Returns the number of image variants written.
func (p *Pipeline) WriteProcessedImages(outputDir string, trackFn func(string)) (int, error) {
	return p.processor.WriteProcessedImages(outputDir, trackFn)
}

// WriteProcessedImagesWithOptions copies cached image variants to the output directory.
func (p *Pipeline) WriteProcessedImagesWithOptions(outputDir string, trackFn func(string), opts WriteOptions) (int, error) {
	return p.processor.WriteProcessedImagesWithOptions(outputDir, trackFn, opts)
}

// WriteBundleAssets copies bundle assets from their source locations to the output directory.
// If trackFn is non-nil, each written path is reported for orphan tracking.
func (p *Pipeline) WriteBundleAssets(pages []*engine.Page, outputDir string, trackFn func(string)) error {
	return p.WriteBundleAssetsWithOptions(pages, outputDir, trackFn, WriteOptions{Parallel: true})
}

// WriteBundleAssetsWithOptions copies bundle assets from their source locations to the output directory.
func (p *Pipeline) WriteBundleAssetsWithOptions(pages []*engine.Page, outputDir string, trackFn func(string), opts WriteOptions) error {
	type assetCopy struct {
		src string
		dst string
	}
	var copies []assetCopy
	for _, page := range pages {
		if len(page.Resources) == 0 {
			continue
		}

		pageDir := filepath.Dir(page.FilePath)

		for _, res := range page.Resources {
			srcPath := filepath.Join(pageDir, res.Name)
			outPath, err := outputpath.SafeJoin(outputDir, res.RelPermalink)
			if err != nil {
				return err
			}
			copies = append(copies, assetCopy{src: srcPath, dst: outPath})
		}
	}

	writeOne := func(c assetCopy) error {
		if err := os.MkdirAll(filepath.Dir(c.dst), 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(c.src)
		if err != nil {
			return fmt.Errorf("reading bundle asset %s: %w", c.src, err)
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
		for _, c := range copies {
			if err := writeOne(c); err != nil {
				return err
			}
		}
		return nil
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
	return g.Wait()
}

func writeBundledFile(outputDir string, f BundledFile, trackFn func(string)) error {
	outPath, err := outputpath.SafeJoin(outputDir, f.OutputURL)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, f.Content, 0o644); err != nil {
		return err
	}
	if trackFn != nil {
		trackFn(outPath)
	}
	return nil
}
