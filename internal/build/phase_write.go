package build

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/getsarde/sarde/internal/asset"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/outputpath"
	"github.com/getsarde/sarde/internal/plugin"
)

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

	// Write theme assets: prefer external theme over embedded.
	themeAssetsDir := ""
	if b.config.Theme.Name != "" {
		themeAssetsDir = filepath.Join(b.projectDir, "themes", b.config.Theme.Name, "assets")
	}
	if themeAssetsDir != "" && dirExists(themeAssetsDir) {
		if err := WriteThemeAssets(b.projectDir, b.config.Theme.Name, s.outputDir, tracker, []string{"assets/js/"}); err != nil {
			return nil, fmt.Errorf("writing theme assets: %w", err)
		}
	} else {
		if err := WriteEmbeddedAssets(b.embeddedFS, s.outputDir, tracker, []string{"assets/js/"}); err != nil {
			return nil, fmt.Errorf("writing embedded theme assets: %w", err)
		}
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
		Tracker:    tracker,
	}
	publicFiles, err := writer.Write(s.rendered, s.aliases)
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
		PublicFiles:     publicFiles,
		ProcessedImages: processedImages,
		AliasCount:      len(s.aliases),
		SitemapCount:    sitemapCount,
		LogMessages:     buildLogger.Messages(),
	}, nil
}
