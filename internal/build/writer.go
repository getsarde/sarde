// Package build orchestrates the full site build pipeline.
package build

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/getsarde/sarde/internal/asset"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/outputpath"
	"github.com/getsarde/sarde/internal/plugin"
	"github.com/getsarde/sarde/internal/workers"
)

// RenderedPage holds a rendered page and its output path.
type RenderedPage struct {
	Page    *engine.Page
	HTML    []byte
	OutPath string // relative to output dir (e.g., "docs/intro/index.html")
}

// Writer handles the Write phase: outputting HTML, aliases, and static files.
type Writer struct {
	OutputDir  string
	ProjectDir string
	Clean      bool
	DevMode    bool
	Tracker    *OutputTracker
}

// resolvedPage holds a pre-resolved output path alongside the rendered HTML,
// so path computation happens once (serial) and goroutines only do I/O.
type resolvedPage struct {
	outPath string
	html    []byte
}

// resolvedAlias holds a pre-resolved alias redirect.
type resolvedAlias struct {
	outPath string
	html    []byte
}

// Write outputs all rendered pages, alias redirects, and static files.
// Returns the number of static files copied.
func (w *Writer) Write(pages []RenderedPage, aliases map[string]string) (int, error) {
	absOutputDir, err := filepath.Abs(w.OutputDir)
	if err != nil {
		return 0, fmt.Errorf("resolving output dir: %w", err)
	}

	// Pre-resolve all output paths in a single pass. Each path is computed
	// once and reused for directory collection, tracking, and the write itself.
	resolved := make([]resolvedPage, 0, len(pages))
	dirs := make(map[string]struct{}, len(pages)/4)

	for _, rp := range pages {
		outPath, err := outputpath.SafeJoinWithRoot(absOutputDir, rp.OutPath)
		if err != nil {
			return 0, err
		}
		dirs[filepath.Dir(outPath)] = struct{}{}
		resolved = append(resolved, resolvedPage{outPath: outPath, html: rp.HTML})
	}

	resolvedAliases := make([]resolvedAlias, 0, len(aliases))
	for aliasPath, target := range aliases {
		outPath, err := outputpath.SafeJoinWithRoot(absOutputDir, PageOutputPath(aliasPath))
		if err != nil {
			return 0, err
		}
		dirs[filepath.Dir(outPath)] = struct{}{}
		resolvedAliases = append(resolvedAliases, resolvedAlias{
			outPath: outPath,
			html:    []byte(redirectHTML(target)),
		})
	}

	// Pre-create all output directories in parallel.
	dirSlice := make([]string, 0, len(dirs))
	for d := range dirs {
		dirSlice = append(dirSlice, d)
	}
	mg := new(errgroup.Group)
	mg.SetLimit(workers.IOLimit(len(dirSlice)))
	for _, dir := range dirSlice {
		mg.Go(func() error {
			return os.MkdirAll(dir, 0o755)
		})
	}
	if err := mg.Wait(); err != nil {
		return 0, fmt.Errorf("creating directories: %w", err)
	}

	// Write rendered HTML pages in parallel.
	// Track paths via an indexed slice (lock-free) instead of per-goroutine
	// mutex calls.
	tracked := make([]string, len(resolved)+len(resolvedAliases))

	g := new(errgroup.Group)
	g.SetLimit(workers.IOLimit(len(resolved)))
	for i, rp := range resolved {
		i, rp := i, rp
		g.Go(func() error {
			tracked[i] = rp.outPath
			return os.WriteFile(rp.outPath, rp.html, 0o644)
		})
	}
	if err := g.Wait(); err != nil {
		return 0, fmt.Errorf("writing pages: %w", err)
	}

	// Write alias redirects in parallel.
	ag := new(errgroup.Group)
	ag.SetLimit(workers.IOLimit(len(resolvedAliases)))
	base := len(resolved)
	for i, ra := range resolvedAliases {
		i, ra := i, ra
		ag.Go(func() error {
			tracked[base+i] = ra.outPath
			return os.WriteFile(ra.outPath, ra.html, 0o644)
		})
	}
	if err := ag.Wait(); err != nil {
		return 0, fmt.Errorf("writing aliases: %w", err)
	}

	// Batch-register all written paths with the tracker (single lock acquisition).
	if w.Tracker != nil {
		w.Tracker.TrackBatch(tracked)
	}

	// Copy static files.
	staticCount, err := w.copyStatic()
	if err != nil {
		return 0, fmt.Errorf("copying static files: %w", err)
	}

	return staticCount, nil
}

// copyStatic copies the ProjectDir/static/ tree to OutputDir/ preserving structure.
// Returns the number of files copied.
func (w *Writer) copyStatic() (int, error) {
	staticDir := filepath.Join(w.ProjectDir, "static")
	info, err := os.Stat(staticDir)
	if err != nil || !info.IsDir() {
		return 0, nil
	}

	type filePair struct{ src, dst string }
	var pairs []filePair
	err = filepath.Walk(staticDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, _ := filepath.Rel(staticDir, path)
		dst, err := outputpath.SafeJoin(w.OutputDir, relPath)
		if err != nil {
			return err
		}
		pairs = append(pairs, filePair{path, dst})
		return nil
	})
	if err != nil {
		return 0, err
	}

	dirs := make(map[string]struct{}, len(pairs)/4)
	for _, p := range pairs {
		dirs[filepath.Dir(p.dst)] = struct{}{}
	}
	for dir := range dirs {
		os.MkdirAll(dir, 0o755)
	}

	tracked := make([]string, len(pairs))
	g := new(errgroup.Group)
	g.SetLimit(workers.IOLimit(len(pairs)))
	for i, p := range pairs {
		i, p := i, p
		g.Go(func() error {
			tracked[i] = p.dst
			return copyFile(p.src, p.dst)
		})
	}
	if err := g.Wait(); err != nil {
		return 0, err
	}
	if w.Tracker != nil {
		w.Tracker.TrackBatch(tracked)
	}
	return len(pairs), nil
}

// PageOutputPath converts a RelPermalink to an output file path.
// "/" → "index.html"
// "/docs/intro/" → "docs/intro/index.html"
// "/404.html" → "404.html"
func PageOutputPath(relPermalink string) string {
	if relPermalink == "/" {
		return "index.html"
	}

	p := strings.TrimPrefix(relPermalink, "/")

	// Already has a file extension (e.g., "404.html").
	if filepath.Ext(p) != "" {
		return p
	}

	p = strings.TrimSuffix(p, "/")
	return p + "/index.html"
}

// redirectHTML generates a minimal redirect page.
func redirectHTML(target string) string {
	safe := strings.ReplaceAll(target, `"`, "%22")
	safe = strings.ReplaceAll(safe, `<`, "%3C")
	safe = strings.ReplaceAll(safe, `>`, "%3E")
	return fmt.Sprintf(
		`<!DOCTYPE html><html><head><meta charset="utf-8"><meta http-equiv="refresh" content="0; url=%s"><link rel="canonical" href="%s"></head><body>Redirecting to <a href="%s">%s</a></body></html>`,
		safe, safe, safe, safe,
	)
}

func writeOutputFile(outputDir, relPath string, data []byte) (string, error) {
	path, err := outputpath.SafeJoin(outputDir, relPath)
	if err != nil {
		return "", err
	}
	return path, writeFile(path, data)
}

// writeFile creates parent directories and writes data to path.
func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (b *SiteBuilder) phaseWrite(s *buildState) (*engine.BuildResult, error) {
	clean := config.BoolVal(b.config.Build.Clean, true)
	var tracker *OutputTracker
	if clean {
		tracker = NewOutputTracker(len(s.rendered) + 256)
	}
	trackFn := func(path string) {
		if tracker != nil {
			tracker.Track(path)
		}
	}

	assetWriteOpts := asset.WriteOptions{Parallel: s.parallel, WorkerCount: s.workerCount}

	if err := s.assetPipeline.WriteBundleAssetsWithOptions(s.allPages, s.outputDir, trackFn, assetWriteOpts); err != nil {
		return nil, fmt.Errorf("writing bundle assets: %w", err)
	}

	processedImages, err := s.assetPipeline.WriteProcessedImagesWithOptions(s.outputDir, trackFn, assetWriteOpts)
	if err != nil {
		return nil, fmt.Errorf("writing processed images: %w", err)
	}

	if err := s.assetPipeline.WriteBundledFilesWithOptions(s.outputDir, trackFn, assetWriteOpts); err != nil {
		return nil, fmt.Errorf("writing bundled files: %w", err)
	}

	if b.themeJSFilename != "" {
		destPath, err := outputpath.SafeJoin(s.outputDir, "assets/js/"+b.themeJSFilename)
		if err != nil {
			return nil, err
		}
		if tracker != nil {
			tracker.Track(destPath)
		}
		if err := writeFile(destPath, b.themeJSContent); err != nil {
			return nil, fmt.Errorf("writing bundled theme JS: %w", err)
		}
	}

	if b.kazariJSFilename != "" {
		destPath, err := outputpath.SafeJoin(s.outputDir, "assets/js/"+b.kazariJSFilename)
		if err != nil {
			return nil, err
		}
		if tracker != nil {
			tracker.Track(destPath)
		}
		if err := writeFile(destPath, b.kazariJSContent); err != nil {
			return nil, fmt.Errorf("writing Kazari JS: %w", err)
		}
	}

	if b.tokenCSSFilename != "" {
		if err := WriteEmbeddedCSS(s.outputDir, b.tokenCSSContent, b.tokenCSSFilename, tracker); err != nil {
			return nil, fmt.Errorf("writing theme token CSS: %w", err)
		}
	}

	if err := WriteEmbeddedAssets(b.embeddedFS, s.outputDir, tracker, []string{"assets/js/"}); err != nil {
		return nil, fmt.Errorf("writing embedded theme assets: %w", err)
	}

	cssFilename := filepath.Base(b.tmplEngine.CSSURL())
	if err := WriteEmbeddedCSS(s.outputDir, b.tmplEngine.CachedCSS(), cssFilename, tracker); err != nil {
		return nil, fmt.Errorf("writing embedded CSS bundle: %w", err)
	}
	s.recordTiming("Writing assets")

	b.warnIconAttribution()

	buildLogger := engine.NewBuildLogger()
	var pluginWarnings []engine.ValidationWarning
	buildDoneCtx := &plugin.BuildDoneContext{
		Config:         b.config,
		OutputDir:      s.outputDir,
		Pages:          s.allPages,
		Collections:    s.collections,
		Site:           s.siteCtx,
		Resolver:       b.urlResolver,
		PageIndex:      s.pageIndex,
		ValidationData: s.validationData,
		DevMode:        b.devMode,
		TrackFn:        trackFn,
	}
	buildDoneCtx.SetWarnings(&pluginWarnings)
	buildDoneCtx.SetLogger(buildLogger)
	if err := b.pluginMgr.RunBuildDone(buildDoneCtx); err != nil {
		return nil, err
	}
	s.recordTiming("Running plugins")

	writer := &Writer{
		OutputDir:  s.outputDir,
		ProjectDir: b.projectDir,
		Clean:      clean,
		DevMode:    b.devMode,
		Tracker:    tracker,
	}
	staticFiles, err := writer.Write(s.rendered, s.aliases)
	if err != nil {
		return nil, fmt.Errorf("writing output: %w", err)
	}
	s.recordTiming("Writing output")

	if tracker != nil && !b.devMode {
		if err := tracker.Prune(s.outputDir); err != nil {
			return nil, fmt.Errorf("pruning output: %w", err)
		}
	}
	s.recordTiming("Pruning output")

	s.warnings = append(s.warnings, pluginWarnings...)

	// Snapshot last-build state for incremental rebuild.
	b.lastCollections = s.collections
	b.lastAllPages = s.allPages
	b.lastSiteCtx = s.siteCtx
	b.lastTaxByLang = s.taxByLang
	b.lastPageIndex = s.pageIndex
	b.lastOutputDir = s.outputDir
	b.lastAssetPipeline = s.assetPipeline
	b.lastScProcessor = s.scProcessor
	b.lastShortcodesHash = s.shortcodesHash
	b.lastPageCache = s.pageCache
	b.lastIconRenderKey = s.iconRenderKey
	b.lastValidationData = s.validationData
	b.built = true

	bundleAssets := 0
	for _, p := range s.allPages {
		bundleAssets += len(p.Resources)
	}
	sitemapCount := 0
	if slices.Contains(b.config.Plugins.Enabled, "sitemap") {
		sitemapCount = 1
	}

	if b.config.I18n.Strict && s.stringTable != nil {
		for lang, keys := range s.stringTable.Misses() {
			for _, key := range keys {
				s.warnings = append(s.warnings, engine.ValidationWarning{
					File:    "i18n/" + lang + ".yaml",
					Field:   key,
					Message: fmt.Sprintf("missing translation for key %q in language %q", key, lang),
					Level:   "warning",
				})
			}
		}
	}

	return &engine.BuildResult{
		PageCount:       len(s.rendered),
		Warnings:        s.warnings,
		OutputDir:       s.outputDir,
		PaginatorPages:  s.paginatorPages,
		Collections:     len(s.collections),
		BundleAssets:    bundleAssets,
		StaticFiles:     staticFiles,
		ProcessedImages: processedImages,
		AliasCount:      len(s.aliases),
		SitemapCount:    sitemapCount,
		LogMessages:     buildLogger.Messages(),
	}, nil
}

// copyFile copies a single file from src to dst.
// Parent directories must already exist.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	// Close explicitly and propagate the close error: io.Copy can succeed
	// while the deferred flush on Close fails (e.g. disk full), which would
	// otherwise leave a truncated file recorded as successfully written.
	_, err = io.Copy(out, in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	return err
}
