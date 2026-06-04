package build

import (
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/frostybee/sarde/embedded"
	"github.com/frostybee/sarde/internal/asset"
	"github.com/frostybee/sarde/internal/collection"
	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/consts"
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/content/markdown"
	"github.com/frostybee/sarde/internal/content/markdown/icons"
	"github.com/frostybee/sarde/internal/devlog"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/i18n"
	"github.com/frostybee/sarde/internal/links"
	"github.com/frostybee/sarde/internal/plugin"
	"github.com/frostybee/sarde/internal/plugin/announcements"
	"github.com/frostybee/sarde/internal/plugin/clientplugins"
	"github.com/frostybee/sarde/internal/plugin/katex"
	"github.com/frostybee/sarde/internal/plugin/mermaid"
	"github.com/frostybee/sarde/internal/plugin/socialcards"
	"github.com/frostybee/sarde/internal/shortcode"
	"github.com/frostybee/sarde/internal/taxonomy"
	sardetemplate "github.com/frostybee/sarde/internal/template"
	"github.com/frostybee/sarde/internal/theme"
	"github.com/frostybee/sarde/internal/theme/syntax"
	"github.com/frostybee/sarde/internal/workers"
)

// BuildOptions provides the inputs for creating a SiteBuilder.
type BuildOptions struct {
	ProjectDir  string
	Config      *config.SiteConfig
	ThemeConfig *engine.ThemeConfig
	EmbeddedFS  fs.FS
	DevMode     bool // Skip heavy optimizations during serve (image processing, minification, etc.)
}

// SiteBuilder orchestrates the full six-phase build pipeline.
type SiteBuilder struct {
	projectDir  string
	config      *config.SiteConfig
	themeConfig *engine.ThemeConfig
	embeddedFS  fs.FS
	devMode     bool

	scanner      *content.Scanner
	mdRenderer   *markdown.Renderer
	tmplEngine   *sardetemplate.Engine
	pluginMgr    *plugin.Manager
	rendererPool chan *markdown.Renderer // lazily initialized pool, persisted across rebuilds
	built        bool                    // true after first Build(); gates one-time registrations

	urlResolver   *engine.URLResolver // URL resolver for basePath prefixing
	resolutionKey string              // digest of resolver state; folded into page-cache keys
	linkGraph     *links.LinkGraph
	lastCoverage  links.CoverageSummary

	checkOnly         bool                // when true, Build() returns after link validation
	checkReportResult *links.ReportResult // stored for Check() to read after Build() returns

	// Last-build state for incremental rebuild.
	lastCollections    map[string]*engine.Collection
	lastAllPages       []*engine.Page
	lastSiteCtx        *engine.SiteContext
	lastTaxByLang      map[string]map[string]*engine.Taxonomy
	lastPageIndex      *content.PageIndex
	lastOutputDir      string
	lastAssetPipeline  *asset.Pipeline
	globalCSSURLs      []string
	themeJSURL         string
	themeJSContent     []byte
	themeJSFilename    string
	tokenCSSURL        string
	tokenCSSContent    string
	tokenCSSFilename   string
	lastScProcessor    *shortcode.Processor
	lastShortcodesHash string
	lastPageCache      *PageCache
	lastValidationData map[string]engine.ValidationEntry
}

// NewSiteBuilder creates a SiteBuilder with all dependencies initialized.
func NewSiteBuilder(opts BuildOptions) *SiteBuilder {
	mgr := plugin.NewManager()
	mgr.RegisterBuiltins(opts.Config.Plugins.Enabled, opts.Config.Plugins.Config)
	registerSubpackagePlugins(mgr, opts.Config.Plugins.Enabled, opts.Config.Plugins.Config)

	return &SiteBuilder{
		projectDir:  opts.ProjectDir,
		config:      opts.Config,
		themeConfig: opts.ThemeConfig,
		embeddedFS:  opts.EmbeddedFS,
		devMode:     opts.DevMode,
		scanner:     &content.Scanner{},
		mdRenderer: markdown.NewRendererFromConfig(markdown.RendererConfig{
			BlockedHrefSchemes: opts.Config.Security.BlockedHrefSchemes,
			HeadingLinks:       config.BoolVal(opts.Config.Site.HeadingLinks, true),
		}),
		tmplEngine: sardetemplate.NewEngine(),
		pluginMgr:  mgr,
	}
}

// resolveContentDir returns the absolute path to the content directory,
// respecting the optional Content.Dir override in site config.
func (b *SiteBuilder) resolveContentDir() string {
	contentDir := filepath.Join(b.projectDir, consts.DirContent)
	if b.config.Content.Dir != "" {
		if filepath.IsAbs(b.config.Content.Dir) {
			contentDir = b.config.Content.Dir
		} else {
			contentDir = filepath.Join(b.projectDir, b.config.Content.Dir)
		}
	}
	return contentDir
}

// Build executes the full six-phase pipeline and writes output to disk.
func (b *SiteBuilder) Build() (*engine.BuildResult, error) {
	start := time.Now()
	var timings []engine.PhaseTiming
	phaseStart := time.Now()
	recordTiming := func(phase string) {
		timings = append(timings, engine.PhaseTiming{Phase: phase, Duration: time.Since(phaseStart)})
		phaseStart = time.Now()
	}

	// Phase 1: INITIALIZE
	contentDir := b.resolveContentDir()

	outputDir, err := ResolveOutputDir(b.projectDir, b.config.Build.Output)
	if err != nil {
		return nil, err
	}
	parallel := config.BoolVal(b.config.Build.Parallel, true)
	workerCount := workers.Count()

	// i18n: load translation strings (always — UI strings are needed even for single-language sites)
	isMultiLang := b.config.I18n.IsMultiLang()
	defaultLang := b.config.I18n.GetDefaultLanguage()
	stringTable, err := i18n.LoadStrings(embedded.I18nFS(), b.projectDir, b.config.Theme.Name, defaultLang)
	if err != nil {
		return nil, fmt.Errorf("loading i18n strings: %w", err)
	}

	if isMultiLang {
		// Configure scanner for multi-language detection
		langCodes := make(map[string]bool)
		for code := range b.config.I18n.Languages {
			langCodes[code] = true
		}
		b.scanner.Languages = langCodes
		b.scanner.DefaultLang = defaultLang
	}

	// Configure scanner for version detection
	b.scanner.VersionIDs = buildScannerVersionIDs(b.config.Collections)

	// Register announcements plugin and run ConfigSetup only on first build.
	// Manager.Register appends without dedup — re-running would double-register hooks.
	if !b.built {
		for _, name := range b.config.Plugins.Enabled {
			if name == "announcements" {
				b.pluginMgr.Register(announcements.New(
					b.config.Plugins.Config[name],
					stringTable,
					b.tmplEngine.CurrentLangPtr(),
				))
				break
			}
		}

		// Plugin hook: ConfigSetup (serial, after config resolved).
		if err := b.pluginMgr.RunConfigSetup(b.config); err != nil {
			return nil, err
		}

		// Icon system: set the default prefix and load extra Iconify sets plus
		// the local icons/ directory. Run-once alongside the embedded sets;
		// editing icon source files needs a dev-server restart to take effect.
		b.loadIconSources()
	}

	// Phase 2: DISCOVER
	phaseStart = time.Now()
	files, err := b.scanner.DiscoverFiles(contentDir)
	if err != nil {
		return nil, fmt.Errorf("discovering content: %w", err)
	}
	recordTiming("Discovering content")

	// Phase 3: PARSE (collections + standalone)
	parseOpts := collection.BuildOptions{Parallel: parallel, WorkerCount: workerCount}
	collections, warnings, err := collection.BuildCollectionsWithOptions(files, b.config, contentDir, parseOpts)
	if err != nil {
		return nil, fmt.Errorf("building collections: %w", err)
	}

	standalones, err := collection.BuildStandalonePagesWithOptions(files, contentDir, b.config.Content.SummaryLength, string(b.config.Build.LastUpdated), parseOpts)
	if err != nil {
		return nil, fmt.Errorf("building standalone pages: %w", err)
	}
	recordTiming("Parsing content")

	// Phase 4: ASSEMBLE

	// Collect all pages.
	var allPages []*engine.Page
	for _, name := range sortedCollectionNames(collections) {
		col := collections[name]
		allPages = append(allPages, col.Pages...)
	}
	allPages = append(allPages, standalones...)

	// Plugin hook: ContentLoaded (serial, plugins can inject virtual pages).
	if err := b.pluginMgr.RunContentLoaded(b.config, collections, &allPages); err != nil {
		return nil, err
	}

	// i18n: link real translations, generate fallbacks, link all translations
	if isMultiLang {
		langCodes := b.config.I18n.LanguageCodes()
		weights := make(map[string]int)
		for code, lc := range b.config.I18n.Languages {
			weights[code] = lc.Weight
		}

		i18n.LinkTranslations(allPages, weights)

		collFallback := make(map[string]string)
		for colName, colCfg := range b.config.Collections {
			if colCfg == nil {
				continue
			}
			if colCfg.I18nFallback != "" {
				collFallback[colName] = colCfg.I18nFallback
			}
			// Versioning fallback overrides i18n fallback for versioned collections.
			if colCfg.Versioning != nil && config.BoolVal(colCfg.Versioning.Enabled, false) && colCfg.Versioning.Fallback != "" {
				collFallback[colName] = colCfg.Versioning.Fallback
			}
		}
		fbOpts := i18n.FallbackOptions{
			SiteFallback:       b.config.I18n.Fallback,
			CollectionFallback: collFallback,
		}

		fallbacks := i18n.GenerateFallbacks(allPages, langCodes, defaultLang, fbOpts)
		allPages = append(allPages, fallbacks...)

		collection.RebuildNavTreesWithFallbacks(collections, allPages, langCodes)

		i18n.LinkAllTranslations(allPages, weights)
	}

	collection.LinkVersions(allPages)

	// Build taxonomies. Multi-language sites get per-language maps; single-
	// language sites build one map with lang="". Permalink resolution on the
	// per-language instances happens after the URLResolver is created (below).
	var taxonomies map[string]*engine.Taxonomy
	var taxByLang map[string]map[string]*engine.Taxonomy

	if isMultiLang {
		langCodes := b.config.I18n.LanguageCodes()
		taxByLang = make(map[string]map[string]*engine.Taxonomy, len(langCodes))
		for _, code := range langCodes {
			langTax := taxonomy.BuildTaxonomies(allPages, b.config.Taxonomies, code)
			w, err := taxonomy.EnrichTaxonomies(langTax, b.config.Taxonomies, b.projectDir, code)
			if err != nil {
				return nil, fmt.Errorf("enriching taxonomies for %s: %w", code, err)
			}
			emitTaxonomyWarnings(w)
			taxByLang[code] = langTax
		}
		taxonomies = taxByLang[defaultLang]
	} else {
		taxonomies = taxonomy.BuildTaxonomies(allPages, b.config.Taxonomies, "")
		w, err := taxonomy.EnrichTaxonomies(taxonomies, b.config.Taxonomies, b.projectDir, "")
		if err != nil {
			return nil, fmt.Errorf("enriching taxonomies: %w", err)
		}
		emitTaxonomyWarnings(w)
	}

	// Build SiteContext.
	siteCtx := &engine.SiteContext{
		Title:            b.config.Site.Title,
		BaseURL:          b.config.Site.URL,
		BasePath:         b.config.Build.BasePath,
		Language:         b.config.Site.Language,
		Config:           b.config,
		Collections:      collections,
		Taxonomies:       taxonomies,
		TaxonomiesByLang: taxByLang,
		Pages:            allPages,
		BuildTime:        time.Now(),
		EditURL:          b.config.Site.EditURL,
		IconLicenses:     buildIconLicenses(),
	}

	// Resolve all Permalink fields through the URL resolver.
	// RelPermalink stays prefix-free; Permalink gets basePath + lang.
	langSet := make(map[string]bool, len(b.config.I18n.Languages))
	for code := range b.config.I18n.Languages {
		langSet[code] = true
	}
	// Build collection-mount and version-ID registries for the resolver.
	collMounts, versionIDs := buildResolverRegistries(collections)

	b.urlResolver = &engine.URLResolver{
		BasePath:         b.config.Build.BasePath,
		BaseURL:          b.config.Site.URL,
		I18nEnabled:      b.config.I18n.IsMultiLang(),
		DefaultLang:      b.config.I18n.GetDefaultLanguage(),
		Strategy:         b.config.I18n.Strategy,
		Languages:        langSet,
		CollectionMounts: collMounts,
		VersionIDs:       versionIDs,
	}
	urlResolver := b.urlResolver
	// Digest of all resolution-affecting state; folded into page-cache keys so a
	// base-path / i18n / version / mount / escape-prefix change busts stale
	// rendered HTML.
	b.resolutionKey = urlResolver.CacheKey() + "|escape=" + b.config.LinkValidation.SiteRootEscapePrefix
	resolvePermalinks(urlResolver, allPages)

	// Resolve per-language taxonomy permalinks through the URLResolver.
	// Each language has its own *Taxonomy/*TaxonomyTerm instances, so
	// mutation is safe. Downstream consumers (ComputeTermEntries, topTerms,
	// templates) read term.Permalink — resolving here makes them lang-aware.
	if taxByLang != nil {
		for code, langTax := range taxByLang {
			for _, tax := range langTax {
				tax.Permalink = urlResolver.URL(tax.Permalink, code, "")
				for _, term := range tax.Terms {
					term.Permalink = urlResolver.URL(term.Permalink, code, "")
				}
			}
		}
		siteCtx.Taxonomies = taxByLang[defaultLang]
	}

	// i18n: populate site-level language info
	if isMultiLang {
		siteCtx.DefaultLang = defaultLang
		for _, code := range b.config.I18n.LanguageCodes() {
			lc := b.config.I18n.Languages[code]
			dir := lc.Dir
			if dir == "" {
				dir = "ltr"
			}
			siteCtx.Languages = append(siteCtx.Languages, engine.Language{
				Code:   code,
				Name:   lc.Name,
				Dir:    dir,
				Weight: lc.Weight,
			})
		}
	}
	recordTiming("Assembling site")

	// Phase 4.5: ASSETS
	assetPipeline := asset.NewPipeline(asset.PipelineOptions{
		ProjectDir: b.projectDir,
		OutputDir:  outputDir,
		Config:     b.config,
		ThemeName:  b.config.Theme.Name,
		EmbeddedFS: b.embeddedFS,
		DevMode:    b.devMode,
	})
	if err := enhancePageResources(allPages, assetPipeline, parallel, workerCount); err != nil {
		return nil, err
	}
	recordTiming("Asset preparation")

	// Build page index for link validation and O(1) ref/relref lookups.
	pageIndex := content.BuildPageIndex(allPages)
	pageIndex.AddAssets(filepath.Join(b.projectDir, consts.DirStatic))

	// Build shortcode registry (three-layer overlay: embedded → theme → user).
	scFuncMap := sardetemplate.BuildShortcodeFuncMap(sardetemplate.ShortcodeFuncMapConfig{
		Site: &siteCtx,
		Resolver: &engine.ThemeResolver{
			ProjectDir: b.projectDir,
			ThemeName:  b.config.Theme.Name,
			EmbeddedFS: b.embeddedFS,
		},
		AssetResolver:  assetPipeline.Resolver(),
		AssetManifest:  assetPipeline.Manifest(),
		ImageProcessor: assetPipeline.ImageProcessor(),
		PageIndex:      &pageIndex,
	})
	scRegistry, err := shortcode.NewRegistry(b.embeddedFS, scFuncMap)
	if err != nil {
		return nil, fmt.Errorf("loading shortcode registry: %w", err)
	}
	if b.config.Theme.Name != "" {
		themeScDir := filepath.Join(b.projectDir, consts.DirThemes, b.config.Theme.Name, consts.DirLayouts, consts.DirShortcodes)
		if err := scRegistry.LoadOverridesFromDir(themeScDir); err != nil {
			return nil, fmt.Errorf("loading theme shortcode overrides: %w", err)
		}
	}
	userScDir := filepath.Join(b.projectDir, consts.DirLayouts, consts.DirShortcodes)
	if err := scRegistry.LoadOverridesFromDir(userScDir); err != nil {
		return nil, fmt.Errorf("loading user shortcode overrides: %w", err)
	}
	scProcessor := shortcode.NewProcessor(scRegistry)
	shortcodesHash := scRegistry.TemplateHash()

	// The icon render mode (inline <svg> vs sprite <use> refs) changes rendered
	// markdown, so it must participate in the page-cache key — otherwise toggling
	// icons.render would serve stale content for unchanged pages.
	iconRenderKey := "icon-inline"
	if icons.SpriteMode() {
		iconRenderKey = "icon-sprite"
	}

	var pageCache *PageCache
	// Disable the cache in check mode: a forced fresh render guarantees the link
	// graph is fully populated (a cache hit would skip link recording, making
	// `sarde check` report "0 links" on a warm cache).
	if config.BoolVal(b.config.Build.Cache, true) && !b.checkOnly {
		pageCache = NewPageCache(b.projectDir)
	}

	// Link graph: records every link resolution attempt for CHECK-phase validation.
	b.linkGraph = links.NewLinkGraph()

	// Validation data: collected links per page for post-build link validation.
	var validationMu sync.Mutex
	validationData := make(map[string]engine.ValidationEntry)

	// Internal link resolution: accumulated pending anchor checks.
	var pendingAnchors []links.PendingAnchorCheck

	// Render markdown for all pages (after asset enhancement so image
	// renderer can access processed resource data for <picture> generation).
	markdownPages := countMarkdownPages(allPages)
	if workers.ShouldParallelize(parallel, markdownPages, workerCount) {
		// Lazily initialize the renderer pool (Goldmark construction is expensive).
		poolSize := workerCount
		if b.rendererPool == nil {
			b.rendererPool = make(chan *markdown.Renderer, poolSize)
			for i := 0; i < poolSize; i++ {
				b.rendererPool <- markdown.NewRendererFromConfig(markdown.RendererConfig{
					BlockedHrefSchemes: b.config.Security.BlockedHrefSchemes,
					HeadingLinks:       config.BoolVal(b.config.Site.HeadingLinks, true),
				})
			}
		}

		// Cap concurrency to the actual pool capacity: the pool is created once
		// and reused across rebuilds, so if workerCount ever exceeds the pool
		// size, borrowers would block forever waiting for a renderer.
		g := new(errgroup.Group)
		g.SetLimit(cap(b.rendererPool))
		for _, page := range allPages {
			if page.RawContent == "" {
				continue
			}
			g.Go(func() error {
				// Borrow a renderer from the pool (needed for both shortcode inner rendering and main render).
				renderer := <-b.rendererPool
				// Configure link resolver with current build's page index and URL resolver.
				lr := renderer.LinkRenderer()
				lr.PageIndex = pageIndex
				lr.URLResolver = urlResolver
				lr.LinkGraph = b.linkGraph
				lr.Collections = collections
				lr.SiteRootEscapePrefix = b.config.LinkValidation.SiteRootEscapePrefix

				// Pre-process shortcodes before Goldmark.
				processed, scWarns := scProcessor.Process(page.RawContent, page, siteCtx, renderer)
				if len(scWarns) > 0 {
					validationMu.Lock()
					warnings = append(warnings, scWarns...)
					validationMu.Unlock()
				}

				// Check cache first.
				hash := ContentHash(processed + shortcodesHash + b.resolutionKey + iconRenderKey)
				if pageCache != nil {
					if entry := pageCache.Get(hash); entry != nil {
						page.Content = htmltemplate.HTML(entry.HTML)
						page.Headings = entry.Headings
						page.HasCodeBlocks = entry.HasCodeBlocks
						page.HasImages = entry.HasImages
						if len(entry.Links) > 0 {
							validationMu.Lock()
							validationData[page.Permalink] = engine.ValidationEntry{Links: entry.Links, FilePath: page.FilePath}
							validationMu.Unlock()
						}
						b.rendererPool <- renderer
						return nil
					}
				}

				lookup := markdown.ImageLookupForPage(page, assetPipeline.ImageProcessor())
				renderer.SetImageLookup(lookup)
				renderer.SetLinkContext(page)

				result, err := renderer.Render(processed)
				// Drain pending anchor checks before returning renderer to pool.
				pagePendingAnchors := renderer.LinkRenderer().DrainPendingAnchors()
				b.rendererPool <- renderer // return to pool
				if err != nil {
					return fmt.Errorf("rendering markdown for %s: %w", page.FilePath, err)
				}
				page.Content = htmltemplate.HTML(result.HTML)
				page.Headings = result.Headings
				page.HasCodeBlocks = result.HasCodeBlocks
				page.HasImages = result.HasImages
				if len(result.Links) > 0 || len(pagePendingAnchors) > 0 {
					validationMu.Lock()
					if len(result.Links) > 0 {
						validationData[page.Permalink] = engine.ValidationEntry{Links: result.Links, FilePath: page.FilePath}
					}
					pendingAnchors = append(pendingAnchors, pagePendingAnchors...)
					validationMu.Unlock()
				}

				// Store in cache.
				if pageCache != nil {
					pageCache.Put(hash, &CacheEntry{
						ContentHash:   hash,
						HTML:          result.HTML,
						Headings:      result.Headings,
						HasCodeBlocks: result.HasCodeBlocks,
						HasImages:     result.HasImages,
						Links:         result.Links,
					})
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
	} else {
		// Configure link resolver for serial rendering.
		lr := b.mdRenderer.LinkRenderer()
		lr.PageIndex = pageIndex
		lr.URLResolver = urlResolver
		lr.LinkGraph = b.linkGraph
		lr.Collections = collections
		lr.SiteRootEscapePrefix = b.config.LinkValidation.SiteRootEscapePrefix

		deps := markdownRenderDeps{
			scProcessor:    scProcessor,
			shortcodesHash: shortcodesHash,
			resolutionKey:  b.resolutionKey,
			iconRenderKey:  iconRenderKey,
			pageCache:      pageCache,
			assetPipeline:  assetPipeline,
		}
		for _, page := range allPages {
			links, scWarns, err := b.renderMarkdownPageSerial(page, deps, siteCtx)
			if err != nil {
				return nil, err
			}
			warnings = append(warnings, scWarns...)
			if len(links) > 0 {
				validationData[page.Permalink] = engine.ValidationEntry{Links: links, FilePath: page.FilePath}
			}
			// Drain pending anchor checks from serial renderer.
			pendingAnchors = append(pendingAnchors, lr.DrainPendingAnchors()...)
		}
	}
	recordTiming("Rendering markdown")

	populatePageIndexHeadings(pageIndex, allPages)
	emitCollisionWarnings(pageIndex.Collisions())

	// Validate pending anchor checks now that all heading IDs are populated.
	// ValidateAnchors writes definitive StatusOK or StatusBrokenAnchor entries
	// into the link graph.
	links.ValidateAnchors(b.linkGraph, pendingAnchors, pageIndex)

	// Compute link coverage summary across all rendered lanes.
	var langCodes []string
	if isMultiLang {
		langCodes = b.config.I18n.LanguageCodes()
	}
	expectedLanes := links.EnumerateLanes(collections, langCodes)
	b.lastCoverage = links.ComputeCoverage(b.linkGraph, allPages, expectedLanes)

	// Generate structured link validation report from the link graph.
	if config.BoolVal(b.config.LinkValidation.Enabled, true) {
		lvc := b.config.LinkValidation

		// CHECK-5: optional external link probing (runs before report generation).
		extCfg := lvc.External
		if config.BoolVal(extCfg.Check, false) {
			timeout, err := time.ParseDuration(extCfg.Timeout)
			if err != nil || timeout <= 0 {
				timeout = 10 * time.Second
			}
			cacheTTL, err := time.ParseDuration(extCfg.CacheTTL)
			if err != nil || cacheTTL <= 0 {
				cacheTTL = 72 * time.Hour
			}
			concurrency := extCfg.Concurrency
			if concurrency <= 0 {
				concurrency = 8
			}
			cachePath := extCfg.Cache
			if cachePath == "" {
				cachePath = filepath.Join(".sarde", "linkcache.json")
			}
			if !filepath.IsAbs(cachePath) {
				cachePath = filepath.Join(b.projectDir, cachePath)
			}
			if err := links.CheckExternalLinks(b.linkGraph, links.ExternalCheckConfig{
				Enabled:     true,
				Concurrency: concurrency,
				Timeout:     timeout,
				CachePath:   cachePath,
				CacheTTL:    cacheTTL,
				OnBroken:    lvc.EffectiveExternalOnBroken(),
				Ignore:      extCfg.Ignore,
				Method:      extCfg.Method,
			}); err != nil {
				fmt.Fprintf(os.Stderr, "warning: external link check failed: %v\n", err)
			}
		}

		siteURL := ""
		if b.config.Site.URL != "" {
			siteURL = strings.TrimRight(b.config.Site.URL, "/")
		}
		reportResult := links.GenerateReport(links.ReportInput{
			Graph:    b.linkGraph,
			Coverage: b.lastCoverage,
			Config: links.LinkCheckConfig{
				OnBroken:             lvc.EffectiveOnBroken(),
				OnBrokenAnchor:       lvc.EffectiveOnBrokenAnchor(),
				OnRelativeLinks:      lvc.EffectiveOnRelativeLinks(),
				OnLocalLinks:         lvc.EffectiveOnLocalLinks(),
				SameSitePolicy:       lvc.SameSitePolicy,
				ReportFormat:         lvc.EffectiveReport(),
				Exclude:              lvc.Exclude,
				OnExternalBroken:     lvc.EffectiveExternalOnBroken(),
				OnUnverifiedInternal: lvc.EffectiveOnUnverifiedInternal(),
			},
			SiteURL: siteURL,
		})
		b.checkReportResult = &reportResult
		if reportResult.Output != "" {
			fmt.Fprint(os.Stderr, reportResult.Output)
		}
		for _, f := range reportResult.Findings {
			if f.Policy == "warn" {
				warnings = append(warnings, engine.ValidationWarning{
					File:    f.Ref.FromFile,
					Message: fmt.Sprintf("%s: %s", f.Type.Label(), f.Ref.RawDest),
					Level:   "warn",
				})
			}
		}
		if reportResult.HasErrors {
			return nil, fmt.Errorf("build failed: link validation errors found")
		}
	}

	// Check-only mode: return after link validation without rendering or writing.
	if b.checkOnly {
		recordTiming("Link validation")
		return &engine.BuildResult{
			PageCount: len(allPages),
			Duration:  time.Since(start),
			Warnings:  warnings,
		}, nil
	}

	// Bundle global CSS/JS assets (processes head.custom_css/custom_js from config).
	if err := assetPipeline.BundleGlobalAssets(); err != nil {
		return nil, fmt.Errorf("bundling global assets: %w", err)
	}
	b.globalCSSURLs = assetPipeline.GlobalCSSURLs()

	// Bundle embedded theme JS into a single minified file.
	jsContent, jsFilename, err := BundleEmbeddedJS(b.embeddedFS, b.devMode)
	if err != nil {
		return nil, fmt.Errorf("bundling embedded theme JS: %w", err)
	}
	if jsFilename != "" {
		b.themeJSURL = "/assets/js/" + jsFilename
		b.themeJSContent = jsContent
		b.themeJSFilename = jsFilename
	}

	// Externalize theme token CSS (design tokens) as a fingerprinted file.
	if b.themeConfig != nil {
		tokenCSS := theme.GenerateCSS(b.themeConfig.Tokens, b.themeConfig.DarkTokens) +
			theme.GenerateLightDarkCSS(b.themeConfig.Tokens, b.themeConfig.DarkTokens)
		if tokenCSS != "" {
			if !b.devMode {
				if minified, err := asset.TransformCSS(tokenCSS, true); err == nil {
					tokenCSS = minified
				}
			}
			hash := asset.Fingerprint([]byte(tokenCSS))
			if b.devMode {
				b.tokenCSSFilename = "tokens-theme.css"
			} else {
				b.tokenCSSFilename = asset.FingerprintedName("tokens-theme.css", hash)
			}
			b.tokenCSSContent = tokenCSS
			b.tokenCSSURL = "/assets/css/" + b.tokenCSSFilename
		}
	}

	// Load template engine (needs SiteContext + asset pipeline + plugin funcs + i18n for funcMap closures).
	b.tmplEngine.SetTokenCSSURL(b.tokenCSSURL)
	b.tmplEngine.SetSiteContext(siteCtx)
	b.tmplEngine.SetURLResolver(urlResolver)
	b.tmplEngine.SetAssetPipeline(assetPipeline.Resolver(), assetPipeline.Manifest())
	b.tmplEngine.SetImageProcessor(assetPipeline.ImageProcessor())
	b.tmplEngine.SetPluginFuncs(b.pluginMgr.TemplateFuncs())
	b.tmplEngine.SetPageIndex(pageIndex)
	if stringTable != nil {
		b.tmplEngine.SetI18nStrings(stringTable)
	}
	// Generate Chroma syntax highlighting CSS from config theme names.
	chromaCSS, err := syntax.GenerateChromaCSS(
		b.config.Markdown.Codeblocks.LightTheme,
		b.config.Markdown.Codeblocks.DarkTheme,
	)
	if err != nil {
		devlog.Warn("build", "syntax highlighting: %v", err)
	} else {
		b.tmplEngine.SetChromaCSS(chromaCSS)
	}

	resolver := &engine.ThemeResolver{
		ProjectDir: b.projectDir,
		ThemeName:  b.config.Theme.Name,
		EmbeddedFS: b.embeddedFS,
	}
	if err := b.tmplEngine.Load(resolver, b.devMode); err != nil {
		return nil, fmt.Errorf("loading templates: %w", err)
	}

	recordTiming("Template setup")

	// Phase 5: RENDER
	var rendered []RenderedPage
	aliases := make(map[string]string)
	var paginatorPages int

	// Filter to renderable pages.
	var renderablePages []*engine.Page
	for _, page := range allPages {
		if page.Params != nil {
			if r, ok := page.Params["render"].(bool); ok && !r {
				continue
			}
		}
		renderablePages = append(renderablePages, page)
	}

	rendered, aliases, err = b.renderPages(renderablePages, siteCtx, parallel, workerCount)
	if err != nil {
		return nil, err
	}
	recordTiming("Rendering templates")

	var syntheticPages []*engine.Page

	// Synthesize numbered pagination pages (e.g. /blog/page/2/) for any collection
	// with Paginate > 0 and enough pages to warrant a second list page.
	for _, col := range collections {
		if col.Config == nil || col.Config.Paginate <= 0 {
			continue
		}
		var contentPages []*engine.Page
		for _, p := range col.Pages {
			if p.Kind != engine.KindSection {
				contentPages = append(contentPages, p)
			}
		}
		perPage := col.Config.Paginate
		total := (len(contentPages) + perPage - 1) / perPage
		if total <= 1 || col.IndexPage == nil {
			continue
		}
		idx := col.IndexPage
		base := idx.RelPermalink
		if base == "" {
			base = "/" + col.Name + "/"
		}
		for n := 2; n <= total; n++ {
			permalink := sardetemplate.PaginationURL(base, n)
			stub := &engine.Page{
				PageIdentity: engine.PageIdentity{
					Title:        idx.Title,
					Kind:         engine.KindSection,
					Permalink:    permalink,
					RelPermalink: permalink,
				},
				PageRelationships: engine.PageRelationships{
					Collection: col,
					Section:    idx.Section,
				},
				PageI18n: engine.PageI18n{Lang: idx.Lang},
				Params: map[string]any{
					consts.PaginationCurrentKey: n,
				},
			}
			syntheticPages = append(syntheticPages, stub)
			paginatorPages++
		}
	}

	// Synthesize taxonomy index and term pages.
	// Multi-language: one set per language, cross-linked for LanguageSwitcher.
	// Single-language: one set with lang="".
	var taxonomyPages []*engine.Page

	allTaxByLang := taxByLang
	if allTaxByLang == nil {
		allTaxByLang = map[string]map[string]*engine.Taxonomy{"": taxonomies}
	}

	// Determine which languages to iterate and their sorted order.
	var taxLangs []string
	if isMultiLang {
		taxLangs = b.config.I18n.LanguageCodes()
	} else {
		taxLangs = []string{""}
	}

	// Pass 1: build all index + term (page 1) stubs, keyed for cross-linking.
	// byLang[code][key] → stub page, where key = taxName or taxName+"/"+slug.
	stubsByLang := make(map[string]map[string]*engine.Page, len(taxLangs))
	for _, lang := range taxLangs {
		langTax := allTaxByLang[lang]
		if langTax == nil {
			continue
		}
		stubs := make(map[string]*engine.Page)
		for _, taxName := range sortedTaxonomyNames(langTax) {
			tax := langTax[taxName]
			cfg := b.config.Taxonomies[taxName]
			if !cfg.ShouldRender() {
				continue
			}
			termEntries := taxonomy.ComputeTermEntries(tax)
			taxStub := buildTaxonomyIndexStub(tax, termEntries, lang)
			stubs[taxName] = taxStub

			for _, term := range tax.Terms {
				termStub := buildTermStub(tax, term, lang)
				stubs[taxName+"/"+term.Slug] = termStub
			}
		}
		stubsByLang[lang] = stubs
	}

	// Pass 2: cross-link, then collect into syntheticPages + taxonomyPages.
	for _, lang := range taxLangs {
		langTax := allTaxByLang[lang]
		if langTax == nil {
			continue
		}
		stubs := stubsByLang[lang]
		for _, taxName := range sortedTaxonomyNames(langTax) {
			tax := langTax[taxName]
			cfg := b.config.Taxonomies[taxName]
			if !cfg.ShouldRender() {
				continue
			}

			taxStub := stubs[taxName]
			if isMultiLang {
				taxStub.Translations = crossLinkTaxStubs(stubsByLang, taxName, lang)
				taxStub.AllTranslations = taxStub.Translations
			}
			syntheticPages = append(syntheticPages, taxStub)
			taxonomyPages = append(taxonomyPages, taxStub)

			paginateBy := cfg.PaginateBy
			if paginateBy <= 0 {
				paginateBy = consts.DefaultPaginateBy
			}
			for _, term := range tax.Terms {
				key := taxName + "/" + term.Slug
				termStub := stubs[key]
				if isMultiLang {
					termStub.Translations = crossLinkTermStubs(stubsByLang, key, lang)
					termStub.AllTranslations = termStub.Translations
				}
				syntheticPages = append(syntheticPages, termStub)
				taxonomyPages = append(taxonomyPages, termStub)

				totalTermPages := (len(term.Pages) + paginateBy - 1) / paginateBy
				if totalTermPages < 1 {
					totalTermPages = 1
				}
				for n := 2; n <= totalTermPages; n++ {
					permalink := sardetemplate.PaginationURL(term.Permalink, n)
					paginatedStub := buildTermPaginatedStub(tax, term, permalink, n, lang)
					syntheticPages = append(syntheticPages, paginatedStub)
					taxonomyPages = append(taxonomyPages, paginatedStub)
					paginatorPages++
				}
			}
		}
	}

	// Resolve Permalinks on synthetic pages (pagination + taxonomy stubs).
	resolvePermalinks(urlResolver, syntheticPages)

	syntheticRendered, syntheticAliases, err := b.renderPages(syntheticPages, siteCtx, parallel, workerCount)
	if err != nil {
		return nil, err
	}
	rendered = append(rendered, syntheticRendered...)
	for alias, target := range syntheticAliases {
		aliases[alias] = target
	}
	recordTiming("Rendering synthetic pages")

	// Include taxonomy pages in allPages for sitemap and search index.
	allPages = append(allPages, taxonomyPages...)
	siteCtx.Pages = allPages

	// Render 404 page(s).
	render404 := func(lang, dir, outPath string) {
		page404 := &engine.Page{
			PageIdentity: engine.PageIdentity{Title: "Page Not Found", Kind: engine.KindPage},
			PageI18n:     engine.PageI18n{Lang: lang},
		}
		templateName := consts.DirDefault + "/404"

		// Convention: auto-detect content/404.md (or content/404.<lang>.md).
		candidates := []string{
			filepath.Join(contentDir, "404."+lang+".md"),
			filepath.Join(contentDir, "404.md"),
		}
		for _, path := range candidates {
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			fm, body, _ := content.ParseFrontmatter(raw)
			if fm != nil && fm.Title != "" {
				page404.Title = fm.Title
			}
			if fm != nil && fm.Description != "" {
				page404.Description = fm.Description
			}
			if fm != nil && fm.Template != "" {
				templateName = consts.DirDefault + "/" + fm.Template
			}
			if body != "" {
				renderResult, renderErr := b.mdRenderer.Render(body)
				if renderErr == nil {
					page404.Content = htmltemplate.HTML(renderResult.HTML)
					page404.Headings = renderResult.Headings
				}
			}
			break
		}

		rd404 := &engine.RouteData{
			Template: templateName,
			Layout:   engine.LayoutDefault,
			Site:     siteCtx,
			Theme:    b.themeConfig,
			Page:     page404,
			RouteI18n: engine.RouteI18n{
				Lang: lang,
				Dir:  dir,
			},
		}
		resolveRouteAssets(urlResolver, rd404)
		html404, err := b.tmplEngine.Render(rd404.Template, rd404)
		if err != nil {
			devlog.Warn("404", "failed to render 404 page (template %q): %v", rd404.Template, err)
			return
		}
		rendered = append(rendered, RenderedPage{
			Page:    page404,
			HTML:    html404,
			OutPath: outPath,
		})
	}

	if isMultiLang {
		// One 404 per language
		for _, code := range b.config.I18n.LanguageCodes() {
			lc := b.config.I18n.Languages[code]
			dir := lc.Dir
			if dir == "" {
				dir = "ltr"
			}
			if code == defaultLang {
				render404(code, dir, consts.Template404)
			} else {
				render404(code, dir, code+"/"+consts.Template404)
			}
		}
	} else {
		lang := siteCtx.Language
		if lang == "" {
			lang = "en"
		}
		render404(lang, "ltr", consts.Template404)
	}
	recordTiming("Rendering 404 pages")

	// PublishLatestAtVersionURL: duplicate latest-version pages at their
	// explicit versioned URL (e.g. /docs/v2/guides/auth/) with canonical
	// pointing to the alias (/docs/guides/auth/).
	rendered = appendVersionedLatestPages(rendered, collections, urlResolver)

	// HTML minification (production builds only).
	if !b.devMode && config.BoolVal(b.config.Build.Minify, true) {
		if err := minifyRendered(rendered, parallel, workerCount); err != nil {
			return nil, err
		}
	}
	recordTiming("Minifying HTML")

	// Phase 6: WRITE
	clean := config.BoolVal(b.config.Build.Clean, true)
	var tracker *OutputTracker
	if clean {
		tracker = NewOutputTracker(len(rendered) + 256)
	}
	trackFn := func(path string) {
		if tracker != nil {
			tracker.Track(path)
		}
	}

	writer := &Writer{
		OutputDir:  outputDir,
		ProjectDir: b.projectDir,
		Clean:      clean,
		DevMode:    b.devMode,
		Tracker:    tracker,
	}
	staticFiles, err := writer.Write(rendered, aliases)
	if err != nil {
		return nil, fmt.Errorf("writing output: %w", err)
	}
	recordTiming("Writing output")

	assetWriteOpts := asset.WriteOptions{Parallel: parallel, WorkerCount: workerCount}

	// Write bundle assets (images, PDFs, etc. co-located with content).
	if err := assetPipeline.WriteBundleAssetsWithOptions(allPages, outputDir, trackFn, assetWriteOpts); err != nil {
		return nil, fmt.Errorf("writing bundle assets: %w", err)
	}

	// Copy processed image variants from cache to output.
	processedImages, err := assetPipeline.WriteProcessedImagesWithOptions(outputDir, trackFn, assetWriteOpts)
	if err != nil {
		return nil, fmt.Errorf("writing processed images: %w", err)
	}

	// Write bundled CSS/JS files to output.
	if err := assetPipeline.WriteBundledFilesWithOptions(outputDir, trackFn, assetWriteOpts); err != nil {
		return nil, fmt.Errorf("writing bundled files: %w", err)
	}

	// Write the bundled theme JS file.
	if b.themeJSFilename != "" {
		destPath, err := safeOutputPath(outputDir, "assets/js/"+b.themeJSFilename)
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

	// Write the externalized theme token CSS file.
	if b.tokenCSSFilename != "" {
		if err := WriteEmbeddedCSS(outputDir, b.tokenCSSContent, b.tokenCSSFilename, tracker); err != nil {
			return nil, fmt.Errorf("writing theme token CSS: %w", err)
		}
	}

	// Write embedded theme-level static assets (skip individual JS files — they're bundled above).
	if err := WriteEmbeddedAssets(b.embeddedFS, outputDir, tracker, []string{"assets/js/"}); err != nil {
		return nil, fmt.Errorf("writing embedded theme assets: %w", err)
	}

	// Write the embedded CSS bundle as an external file (fingerprinted).
	cssFilename := filepath.Base(b.tmplEngine.CSSURL())
	if err := WriteEmbeddedCSS(outputDir, b.tmplEngine.CachedCSS(), cssFilename, tracker); err != nil {
		return nil, fmt.Errorf("writing embedded CSS bundle: %w", err)
	}
	recordTiming("Writing assets")

	// Warn if an attribution-required icon set was used without configuring
	// icons.attribution (after render, so used-set tracking is complete).
	b.warnIconAttribution()

	// Plugin hook: BuildDone (parallel, after all files written).
	buildLogger := engine.NewBuildLogger()
	var pluginWarnings []engine.ValidationWarning
	buildDoneCtx := &plugin.BuildDoneContext{
		Config:         b.config,
		OutputDir:      outputDir,
		Pages:          allPages,
		Collections:    collections,
		Site:           siteCtx,
		Resolver:       b.urlResolver,
		PageIndex:      pageIndex,
		ValidationData: validationData,
		DevMode:        b.devMode,
		TrackFn:        trackFn,
	}
	buildDoneCtx.SetWarnings(&pluginWarnings)
	buildDoneCtx.SetLogger(buildLogger)
	if err := b.pluginMgr.RunBuildDone(buildDoneCtx); err != nil {
		return nil, err
	}
	recordTiming("Running plugins")

	// Prune orphaned files from previous builds.
	if tracker != nil {
		if err := tracker.Prune(outputDir); err != nil {
			return nil, fmt.Errorf("pruning output: %w", err)
		}
	}
	recordTiming("Pruning output")
	warnings = append(warnings, pluginWarnings...)

	// Snapshot last-build state for incremental rebuild.
	b.lastCollections = collections
	b.lastAllPages = allPages
	b.lastSiteCtx = siteCtx
	b.lastTaxByLang = taxByLang
	b.lastPageIndex = pageIndex
	b.lastOutputDir = outputDir
	b.lastAssetPipeline = assetPipeline
	b.lastScProcessor = scProcessor
	b.lastShortcodesHash = shortcodesHash
	b.lastPageCache = pageCache
	b.lastValidationData = validationData
	b.built = true

	// Compute summary stats.
	bundleAssets := 0
	for _, p := range allPages {
		bundleAssets += len(p.Resources)
	}
	sitemapCount := 0
	if slices.Contains(b.config.Plugins.Enabled, "sitemap") {
		sitemapCount = 1
	}

	return &engine.BuildResult{
		PageCount:       len(rendered),
		Duration:        time.Since(start),
		Warnings:        warnings,
		OutputDir:       outputDir,
		PaginatorPages:  paginatorPages,
		Collections:     len(collections),
		BundleAssets:    bundleAssets,
		StaticFiles:     staticFiles,
		ProcessedImages: processedImages,
		AliasCount:      len(aliases),
		SitemapCount:    sitemapCount,
		LogMessages:     buildLogger.Messages(),
		PhaseTimings:    timings,
	}, nil
}

// loadIconSources configures the icon engine from b.config.Icons: the default
// prefix, any extra Iconify sets (explicit files plus a sets_dir of *.json),
// and the local icons/ directory. Failures are warnings, never fatal — a
// missing or malformed icon source must not break the build.
func (b *SiteBuilder) loadIconSources() {
	ic := b.config.Icons
	icons.SetDefaultPrefix(ic.DefaultPrefix)
	icons.SetRenderMode(ic.Render)

	resolve := func(p string) string {
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(b.projectDir, p)
	}
	loadSet := func(path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			devlog.Warn("icons", "read set %s: %v", path, err)
			return
		}
		if err := icons.LoadCollection(data); err != nil {
			devlog.Warn("icons", "load set %s: %v", path, err)
		}
	}

	for _, set := range ic.Sets {
		if set.File != "" {
			loadSet(resolve(set.File))
		}
	}
	if ic.SetsDir != "" {
		dir := resolve(ic.SetsDir)
		if entries, err := os.ReadDir(dir); err != nil {
			if !os.IsNotExist(err) {
				devlog.Warn("icons", "read sets_dir %s: %v", dir, err)
			}
		} else {
			for _, e := range entries {
				if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".json") {
					loadSet(filepath.Join(dir, e.Name()))
				}
			}
		}
	}

	localDir := ic.LocalDir
	if localDir == "" {
		localDir = "icons"
	}
	if err := icons.LoadIconDirectory(resolve(localDir)); err != nil {
		devlog.Warn("icons", "load local dir %s: %v", localDir, err)
	}
}

// buildIconLicenses converts the loaded icon sets' license metadata into the
// engine type exposed to templates as .Site.IconLicenses.
func buildIconLicenses() []engine.IconLicense {
	sets := icons.LoadedSetLicenses()
	out := make([]engine.IconLicense, 0, len(sets))
	for _, s := range sets {
		out = append(out, engine.IconLicense{Prefix: s.Prefix, Title: s.Title, SPDX: s.SPDX, URL: s.URL})
	}
	return out
}

// warnIconAttribution emits a build-end warning for any icon set actually used
// whose license requires attribution when no icons.attribution is configured.
// The bundled sets (Lucide ISC / Tabler MIT) never trip this; it guards opt-in
// sets added via icons.sets / sets_dir. Brand/logo icons may carry separate
// trademark constraints regardless of SPDX license.
func (b *SiteBuilder) warnIconAttribution() {
	if strings.TrimSpace(b.config.Icons.Attribution) != "" {
		return
	}
	for _, s := range icons.UsedSetLicenses() {
		if !requiresAttribution(s.SPDX) {
			continue
		}
		label := s.Title
		if label == "" {
			label = s.SPDX
		}
		devlog.Warn("icons", "set %q (%s) requires attribution; set icons.attribution or render a credits page from .Site.IconLicenses", s.Prefix, label)
	}
}

// requiresAttribution reports whether an SPDX license id needs an attribution
// notice (CC-BY family, OFL, or a GPL/copyleft license).
func requiresAttribution(spdx string) bool {
	s := strings.ToUpper(strings.TrimSpace(spdx))
	switch {
	case strings.HasPrefix(s, "CC-BY"):
		return true
	case strings.HasPrefix(s, "OFL"):
		return true
	case strings.Contains(s, "GPL"):
		return true
	}
	return false
}

// registerSubpackagePlugins wires plugins whose assets live in their own
// subpackage (and therefore can't be referenced from internal/plugin/registry.go
// without creating an import cycle).
// appendVersionedLatestPages appends duplicate RenderedPage entries for
// collections with PublishLatestAtVersionURL enabled. These duplicates emit
// the latest-version content at the explicit versioned URL (e.g.
// /docs/v2/guides/auth/) alongside the alias URL (/docs/guides/auth/).
func appendVersionedLatestPages(rendered []RenderedPage, collections map[string]*engine.Collection, r *engine.URLResolver) []RenderedPage {
	for _, col := range collections {
		vc := col.Config.Versioning
		if vc == nil || !vc.Enabled || !vc.PublishLatestAtVersionURL || vc.LastVersion == "" {
			continue
		}
		for i := range rendered {
			rp := &rendered[i]
			if rp.Page == nil || rp.Page.Collection != col {
				continue
			}
			if rp.Page.Version != vc.LastVersion {
				continue
			}
			versionedOutPath := PageOutputPath(r.OutputRelPath(rp.Page.RelPermalink, rp.Page.Lang, vc.LastVersion))
			rendered = append(rendered, RenderedPage{
				Page:    rp.Page,
				HTML:    rp.HTML,
				OutPath: versionedOutPath,
			})
		}
	}
	return rendered
}

// buildScannerVersionIDs constructs the collection→versionIDs map from config
// for the scanner's version detection pass.
func buildScannerVersionIDs(collections map[string]*config.CollectionSiteConfig) map[string]map[string]bool {
	if len(collections) == 0 {
		return nil
	}
	result := make(map[string]map[string]bool)
	for name, colCfg := range collections {
		if colCfg == nil || colCfg.Versioning == nil || !config.BoolVal(colCfg.Versioning.Enabled, false) {
			continue
		}
		ids := make(map[string]bool, len(colCfg.Versioning.Versions))
		for _, v := range colCfg.Versioning.Versions {
			if v.ID != "" {
				ids[v.ID] = true
			}
		}
		if len(ids) > 0 {
			result[name] = ids
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// buildResolverRegistries extracts collection mounts and a union of version IDs
// from all built collections for the URL resolver.
func buildResolverRegistries(collections map[string]*engine.Collection) ([]string, map[string]bool) {
	mounts := make([]string, 0, len(collections))
	versionIDs := make(map[string]bool)
	for name, col := range collections {
		mounts = append(mounts, "/"+name)
		if col.Config != nil && col.Config.Versioning != nil && col.Config.Versioning.Enabled {
			for _, vd := range col.Config.Versioning.Versions {
				versionIDs[vd.ID] = true
			}
		}
	}
	return mounts, versionIDs
}

func registerSubpackagePlugins(mgr *plugin.Manager, enabled []string, configs map[string]map[string]any) {
	for _, name := range enabled {
		switch name {
		case "katex":
			mgr.Register(katex.New(configs[name]))
		case "mermaid":
			mgr.Register(mermaid.New(configs[name]))
		case "announcements":
			// Registered in Build() after stringTable is available (needs i18n).
		case "social_cards":
			mgr.Register(socialcards.New(configs[name]))
		}
	}
	clientplugins.RegisterAll(mgr, enabled, configs)
}
