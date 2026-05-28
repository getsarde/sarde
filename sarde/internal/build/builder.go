package build

import (
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
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
	"github.com/frostybee/sarde/internal/devlog"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/i18n"
	"github.com/frostybee/sarde/internal/plugin"
	"github.com/frostybee/sarde/internal/plugin/announcements"
	"github.com/frostybee/sarde/internal/plugin/clientplugins"
	"github.com/frostybee/sarde/internal/plugin/katex"
	"github.com/frostybee/sarde/internal/plugin/mermaid"
	"github.com/frostybee/sarde/internal/plugin/socialcards"
	"github.com/frostybee/sarde/internal/shortcode"
	"github.com/frostybee/sarde/internal/taxonomy"
	sardetemplate "github.com/frostybee/sarde/internal/template"
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

	// Last-build state for incremental rebuild.
	lastCollections    map[string]*engine.Collection
	lastAllPages       []*engine.Page
	lastSiteCtx        *engine.SiteContext
	lastTaxonomies     map[string]*engine.Taxonomy
	lastPageIndex      *content.PageIndex
	lastOutputDir      string
	lastAssetPipeline  *asset.Pipeline
	globalCSSURLs      []string
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

	// i18n: generate fallback pages and link translations
	if isMultiLang {
		langCodes := b.config.I18n.LanguageCodes()
		fallbacks := i18n.GenerateFallbacks(allPages, langCodes, defaultLang)
		allPages = append(allPages, fallbacks...)

		// Build language weight map for sorting
		weights := make(map[string]int)
		for code, lc := range b.config.I18n.Languages {
			weights[code] = lc.Weight
		}
		i18n.LinkTranslations(allPages, weights)
	}

	// Versioning: link version peers across versioned collections.
	collection.LinkVersions(allPages)

	// Build taxonomies.
	taxonomies := taxonomy.BuildTaxonomies(allPages, b.config.Taxonomies)

	// Enrich taxonomies with metadata from data/*.yml and validate.
	taxWarnings, err := taxonomy.EnrichTaxonomies(taxonomies, b.config.Taxonomies, b.projectDir)
	if err != nil {
		return nil, fmt.Errorf("enriching taxonomies: %w", err)
	}
	emitTaxonomyWarnings(taxWarnings)

	// Build SiteContext.
	siteCtx := &engine.SiteContext{
		Title:       b.config.Site.Title,
		BaseURL:     b.config.Site.URL,
		Language:    b.config.Site.Language,
		Config:      b.config,
		Collections: collections,
		Taxonomies:  taxonomies,
		Pages:       allPages,
		BuildTime:   time.Now(),
		EditURL:     b.config.Site.EditURL,
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

	var pageCache *PageCache
	if config.BoolVal(b.config.Build.Cache, true) {
		pageCache = NewPageCache(b.projectDir)
	}

	// Validation data: collected links per page for post-build link validation.
	var validationMu sync.Mutex
	validationData := make(map[string]engine.ValidationEntry)

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

		g := new(errgroup.Group)
		g.SetLimit(poolSize)
		for _, page := range allPages {
			if page.RawContent == "" {
				continue
			}
			g.Go(func() error {
				// Borrow a renderer from the pool (needed for both shortcode inner rendering and main render).
				renderer := <-b.rendererPool

				// Pre-process shortcodes before Goldmark.
				processed, scWarns := scProcessor.Process(page.RawContent, page, siteCtx, renderer)
				if len(scWarns) > 0 {
					validationMu.Lock()
					warnings = append(warnings, scWarns...)
					validationMu.Unlock()
				}

				// Check cache first.
				hash := ContentHash(processed + shortcodesHash)
				if pageCache != nil {
					if entry := pageCache.Get(hash); entry != nil {
						page.Content = htmltemplate.HTML(entry.HTML)
						page.Headings = entry.Headings
						page.HasCodeBlocks = entry.HasCodeBlocks
						page.HasImages = entry.HasImages
						if len(entry.Links) > 0 {
							validationMu.Lock()
							validationData[page.RelPermalink] = engine.ValidationEntry{Links: entry.Links, FilePath: page.FilePath}
							validationMu.Unlock()
						}
						b.rendererPool <- renderer
						return nil
					}
				}

				lookup := markdown.ImageLookupForPage(page, assetPipeline.ImageProcessor())
				renderer.SetImageLookup(lookup)

				result, err := renderer.Render(processed)
				b.rendererPool <- renderer // return to pool
				if err != nil {
					return fmt.Errorf("rendering markdown for %s: %w", page.FilePath, err)
				}
				page.Content = htmltemplate.HTML(result.HTML)
				page.Headings = result.Headings
				page.HasCodeBlocks = result.HasCodeBlocks
				page.HasImages = result.HasImages
				if len(result.Links) > 0 {
					validationMu.Lock()
					validationData[page.RelPermalink] = engine.ValidationEntry{Links: result.Links, FilePath: page.FilePath}
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
		deps := markdownRenderDeps{
			scProcessor:    scProcessor,
			shortcodesHash: shortcodesHash,
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
				validationData[page.RelPermalink] = engine.ValidationEntry{Links: links, FilePath: page.FilePath}
			}
		}
	}
	recordTiming("Rendering markdown")

	populatePageIndexHeadings(pageIndex, allPages)

	// Bundle global CSS/JS assets (processes head.custom_css/custom_js from config).
	if err := assetPipeline.BundleGlobalAssets(); err != nil {
		return nil, fmt.Errorf("bundling global assets: %w", err)
	}
	b.globalCSSURLs = assetPipeline.GlobalCSSURLs()

	// Load template engine (needs SiteContext + asset pipeline + plugin funcs + i18n for funcMap closures).
	b.tmplEngine.SetSiteContext(siteCtx)
	b.tmplEngine.SetAssetPipeline(assetPipeline.Resolver(), assetPipeline.Manifest())
	b.tmplEngine.SetImageProcessor(assetPipeline.ImageProcessor())
	b.tmplEngine.SetPluginFuncs(b.pluginMgr.TemplateFuncs())
	b.tmplEngine.SetPageIndex(pageIndex)
	if stringTable != nil {
		b.tmplEngine.SetI18nStrings(stringTable)
	}
	// Generate Chroma syntax highlighting CSS from config theme names.
	chromaCSS, err := syntax.GenerateChromaCSS(
		b.config.Markdown.Highlighting.LightTheme,
		b.config.Markdown.Highlighting.DarkTheme,
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
	var taxonomyPages []*engine.Page
	for _, taxName := range sortedTaxonomyNames(taxonomies) {
		tax := taxonomies[taxName]
		cfg := b.config.Taxonomies[taxName]
		if !cfg.ShouldRender() {
			continue
		}

		termEntries := taxonomy.ComputeTermEntries(tax)

		// Taxonomy index page (e.g. /tags/).
		taxStub := buildTaxonomyIndexStub(tax, termEntries)
		syntheticPages = append(syntheticPages, taxStub)
		taxonomyPages = append(taxonomyPages, taxStub)

		// Per-term pages (e.g. /tags/go/, /tags/go/page/2/).
		paginateBy := cfg.PaginateBy
		if paginateBy <= 0 {
			paginateBy = consts.DefaultPaginateBy
		}
		for _, term := range tax.Terms {
			totalTermPages := (len(term.Pages) + paginateBy - 1) / paginateBy
			if totalTermPages < 1 {
				totalTermPages = 1
			}

			// Page 1.
			termStub := buildTermStub(tax, term)
			syntheticPages = append(syntheticPages, termStub)
			taxonomyPages = append(taxonomyPages, termStub)

			// Pages 2..N.
			for n := 2; n <= totalTermPages; n++ {
				permalink := sardetemplate.PaginationURL(term.Permalink, n)
				paginatedStub := buildTermPaginatedStub(tax, term, permalink, n)
				syntheticPages = append(syntheticPages, paginatedStub)
				taxonomyPages = append(taxonomyPages, paginatedStub)
				paginatorPages++
			}
		}
	}

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
		html404, err := b.tmplEngine.Render(rd404.Template, rd404)
		if err == nil {
			rendered = append(rendered, RenderedPage{
				Page:    page404,
				HTML:    html404,
				OutPath: outPath,
			})
		}
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

	// Write embedded theme-level static assets (JS helpers like copy-code, spoiler, tabs, prefetch).
	if err := WriteEmbeddedAssets(b.embeddedFS, outputDir, tracker); err != nil {
		return nil, fmt.Errorf("writing embedded theme assets: %w", err)
	}

	// Write the embedded CSS bundle as an external file (fingerprinted).
	cssFilename := filepath.Base(b.tmplEngine.CSSURL())
	if err := WriteEmbeddedCSS(outputDir, b.tmplEngine.CachedCSS(), cssFilename, tracker); err != nil {
		return nil, fmt.Errorf("writing embedded CSS bundle: %w", err)
	}
	recordTiming("Writing assets")

	// Plugin hook: BuildDone (parallel, after all files written).
	buildLogger := engine.NewBuildLogger()
	var pluginWarnings []engine.ValidationWarning
	buildDoneCtx := &plugin.BuildDoneContext{
		Config:         b.config,
		OutputDir:      outputDir,
		Pages:          allPages,
		Collections:    collections,
		Site:           siteCtx,
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
	b.lastTaxonomies = taxonomies
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

// registerSubpackagePlugins wires plugins whose assets live in their own
// subpackage (and therefore can't be referenced from internal/plugin/registry.go
// without creating an import cycle).
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
