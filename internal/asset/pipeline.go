package asset

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/internal/engine"
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
	resolver    *Resolver
	enhancer    *ResourceEnhancer
	processor   *ImageProcessor
	bundler     *Bundler
	cache       *Cache
	manifest    *Manifest
	bundledFiles []BundledFile // CSS/JS files to write after writer cleans
	opts        PipelineOptions
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

// WriteBundledFiles writes bundled CSS/JS files to the output directory.
// Must be called after the writer has cleaned and written HTML files.
func (p *Pipeline) WriteBundledFiles(outputDir string) error {
	for _, f := range p.bundledFiles {
		outPath := filepath.Join(outputDir, filepath.FromSlash(f.OutputURL))
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(outPath, f.Content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// WriteProcessedImages copies cached image variants to the output directory.
// Must be called after the writer has cleaned and written HTML files.
func (p *Pipeline) WriteProcessedImages(outputDir string) error {
	return p.processor.WriteProcessedImages(outputDir)
}

// WriteBundleAssets copies bundle assets from their source locations to the output directory.
// Each resource is written to the path matching its RelPermalink.
func (p *Pipeline) WriteBundleAssets(pages []*engine.Page, outputDir string) error {
	for _, page := range pages {
		if len(page.Resources) == 0 {
			continue
		}

		pageDir := filepath.Dir(page.FilePath)

		for _, res := range page.Resources {
			srcPath := filepath.Join(pageDir, res.Name)

			// Derive output path from RelPermalink.
			outPath := filepath.Join(outputDir, filepath.FromSlash(res.RelPermalink))

			// Ensure parent directory exists.
			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				return err
			}

			// Copy the file.
			data, err := os.ReadFile(srcPath)
			if err != nil {
				continue // skip missing resources
			}
			if err := os.WriteFile(outPath, data, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}
