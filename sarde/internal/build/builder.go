package build

import (
	"context"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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
	contentDir := filepath.Join(b.projectDir, "content")
	if b.config.Content.Dir != "" {
		if filepath.IsAbs(b.config.Content.Dir) {
			contentDir = b.config.Content.Dir
		} else {
			contentDir = filepath.Join(b.projectDir, b.config.Content.Dir)
		}
	}

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
	var scDataCache sync.Map
	scFuncMap := sardetemplate.BuildShortcodeFuncMap(
		&siteCtx,
		&engine.ThemeResolver{
			ProjectDir: b.projectDir,
			ThemeName:  b.config.Theme.Name,
			EmbeddedFS: b.embeddedFS,
		},
		&scDataCache,
		assetPipeline.Resolver(),
		assetPipeline.Manifest(),
		assetPipeline.ImageProcessor(),
		&pageIndex,
	)
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

		g, _ := errgroup.WithContext(context.Background())
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
		for _, page := range allPages {
			if page.RawContent == "" {
				continue
			}

			// Pre-process shortcodes before Goldmark.
			processed, scWarns := scProcessor.Process(page.RawContent, page, siteCtx, b.mdRenderer)
			warnings = append(warnings, scWarns...)

			hash := ContentHash(processed + shortcodesHash)
			if pageCache != nil {
				if entry := pageCache.Get(hash); entry != nil {
					page.Content = htmltemplate.HTML(entry.HTML)
					page.Headings = entry.Headings
					page.HasCodeBlocks = entry.HasCodeBlocks
					page.HasImages = entry.HasImages
					if len(entry.Links) > 0 {
						validationData[page.RelPermalink] = engine.ValidationEntry{Links: entry.Links, FilePath: page.FilePath}
					}
					continue
				}
			}

			lookup := markdown.ImageLookupForPage(page, assetPipeline.ImageProcessor())
			b.mdRenderer.SetImageLookup(lookup)

			result, err := b.mdRenderer.Render(processed)
			if err != nil {
				return nil, fmt.Errorf("rendering markdown for %s: %w", page.FilePath, err)
			}
			page.Content = htmltemplate.HTML(result.HTML)
			page.Headings = result.Headings
			page.HasCodeBlocks = result.HasCodeBlocks
			page.HasImages = result.HasImages
			if len(result.Links) > 0 {
				validationData[page.RelPermalink] = engine.ValidationEntry{Links: result.Links, FilePath: page.FilePath}
			}

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
		}
	}
	recordTiming("Rendering markdown")

	// Populate heading index from rendered pages.
	for _, page := range allPages {
		if len(page.Headings) > 0 {
			ids := make([]string, len(page.Headings))
			for i, h := range page.Headings {
				ids[i] = h.ID
			}
			pageIndex.SetHeadings(page.RelPermalink, ids)
		}
	}

	// Bundle global CSS/JS assets (processes head.custom_css/custom_js from config).
	if err := assetPipeline.BundleGlobalAssets(); err != nil {
		return nil, fmt.Errorf("bundling global assets: %w", err)
	}

	// Load template engine (needs SiteContext + asset pipeline + plugin funcs + i18n for funcMap closures).
	b.tmplEngine.SetSiteContext(siteCtx)
	b.tmplEngine.SetAssetPipeline(assetPipeline.Resolver(), assetPipeline.Manifest())
	b.tmplEngine.SetImageProcessor(assetPipeline.ImageProcessor())
	b.tmplEngine.SetPluginFuncs(b.pluginMgr.TemplateFuncs())
	b.tmplEngine.SetPageIndex(pageIndex)
	if stringTable != nil {
		b.tmplEngine.SetI18nStrings(stringTable)
	}
	resolver := &engine.ThemeResolver{
		ProjectDir: b.projectDir,
		ThemeName:  b.config.Theme.Name,
		EmbeddedFS: b.embeddedFS,
	}
	if err := b.tmplEngine.Load(resolver); err != nil {
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
		if !strings.HasSuffix(base, "/") {
			base += "/"
		}
		for n := 2; n <= total; n++ {
			permalink := fmt.Sprintf("%spage/%d/", base, n)
			stub := &engine.Page{
				Title:        idx.Title,
				Kind:         engine.KindSection,
				Permalink:    permalink,
				RelPermalink: permalink,
				Collection:   col,
				Section:      idx.Section,
				Lang:         idx.Lang,
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
		taxStub := &engine.Page{
			Title:        tax.Name,
			Kind:         engine.KindTaxonomy,
			Permalink:    tax.Permalink,
			RelPermalink: tax.Permalink,
			Params: map[string]any{
				"__taxonomy":     tax,
				"__term_entries": termEntries,
			},
		}
		syntheticPages = append(syntheticPages, taxStub)
		taxonomyPages = append(taxonomyPages, taxStub)

		// Per-term pages (e.g. /tags/go/, /tags/go/page/2/).
		paginateBy := cfg.PaginateBy
		if paginateBy <= 0 {
			paginateBy = 10
		}
		for _, term := range tax.Terms {
			totalTermPages := (len(term.Pages) + paginateBy - 1) / paginateBy
			if totalTermPages < 1 {
				totalTermPages = 1
			}

			// Page 1.
			termStub := &engine.Page{
				Title:        term.Label,
				Kind:         engine.KindTerm,
				Permalink:    term.Permalink,
				RelPermalink: term.Permalink,
				Params: map[string]any{
					"__taxonomy":      tax,
					"__taxonomy_term": term,
				},
			}
			syntheticPages = append(syntheticPages, termStub)
			taxonomyPages = append(taxonomyPages, termStub)

			// Pages 2..N.
			for n := 2; n <= totalTermPages; n++ {
				permalink := fmt.Sprintf("%spage/%d/", term.Permalink, n)
				paginatedStub := &engine.Page{
					Title:        term.Label,
					Kind:         engine.KindTerm,
					Permalink:    permalink,
					RelPermalink: permalink,
					Params: map[string]any{
						"__taxonomy":                tax,
						"__taxonomy_term":           term,
						consts.PaginationCurrentKey: n,
					},
				}
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
		page404 := &engine.Page{Title: "Page Not Found", Kind: engine.KindPage, Lang: lang}
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
			Lang:     lang,
			Dir:      dir,
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

	// Write the embedded CSS bundle as an external file.
	if err := WriteEmbeddedCSS(outputDir, b.tmplEngine.CachedCSS(), tracker); err != nil {
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
	for _, name := range b.config.Plugins.Enabled {
		if name == "sitemap" {
			sitemapCount = 1
			break
		}
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

func enhancePageResources(pages []*engine.Page, pipeline *asset.Pipeline, parallel bool, workerCount int) error {
	if !workers.ShouldParallelize(parallel, len(pages), workerCount) {
		for _, page := range pages {
			if err := pipeline.EnhanceResources(page); err != nil {
				return fmt.Errorf("enhancing resources for %s: %w", page.FilePath, err)
			}
		}
		return nil
	}

	g, _ := errgroup.WithContext(context.Background())
	g.SetLimit(minWorkerLimit(workerCount, len(pages)))
	for _, page := range pages {
		page := page
		g.Go(func() error {
			if err := pipeline.EnhanceResources(page); err != nil {
				return fmt.Errorf("enhancing resources for %s: %w", page.FilePath, err)
			}
			return nil
		})
	}
	return g.Wait()
}

func (b *SiteBuilder) renderPages(pages []*engine.Page, siteCtx *engine.SiteContext, parallel bool, workerCount int) ([]RenderedPage, map[string]string, error) {
	aliases := make(map[string]string)
	if len(pages) == 0 {
		return nil, aliases, nil
	}

	if !workers.ShouldParallelize(parallel, len(pages), workerCount) {
		rendered := make([]RenderedPage, 0, len(pages))
		for _, page := range pages {
			rp, err := b.renderPage(page, siteCtx)
			if err != nil {
				return nil, nil, err
			}
			rendered = append(rendered, rp)
			for _, alias := range page.Aliases {
				aliases[alias] = page.RelPermalink
			}
		}
		return rendered, aliases, nil
	}

	rendered := make([]RenderedPage, len(pages))
	g, _ := errgroup.WithContext(context.Background())
	g.SetLimit(minWorkerLimit(workerCount, len(pages)))
	for i, page := range pages {
		i, page := i, page
		g.Go(func() error {
			rp, err := b.renderPage(page, siteCtx)
			if err != nil {
				return err
			}
			rendered[i] = rp
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}
	for _, page := range pages {
		for _, alias := range page.Aliases {
			aliases[alias] = page.RelPermalink
		}
	}
	return rendered, aliases, nil
}

func (b *SiteBuilder) renderPage(page *engine.Page, siteCtx *engine.SiteContext) (RenderedPage, error) {
	rd := sardetemplate.BuildRouteData(page, siteCtx, b.themeConfig)
	if err := b.pluginMgr.RunBeforeRender(b.config, page, rd, siteCtx); err != nil {
		return RenderedPage{}, err
	}

	var html []byte
	if redirect := tabbedCollectionRedirect(page); redirect != "" {
		html = []byte(buildRedirectHTML(redirect))
	} else {
		var err error
		html, err = b.tmplEngine.Render(rd.Template, rd)
		if err != nil {
			return RenderedPage{}, fmt.Errorf("rendering %s (template %s): %w", page.FilePath, rd.Template, err)
		}
	}

	return RenderedPage{
		Page:    page,
		HTML:    html,
		OutPath: PageOutputPath(page.RelPermalink),
	}, nil
}

func minifyRendered(rendered []RenderedPage, parallel bool, workerCount int) error {
	if !workers.ShouldParallelize(parallel, len(rendered), workerCount) {
		mn := NewMinifier()
		for i := range rendered {
			rendered[i].HTML = mn.MinifyHTML(rendered[i].HTML)
		}
		return nil
	}

	g, _ := errgroup.WithContext(context.Background())
	g.SetLimit(minWorkerLimit(workerCount, len(rendered)))
	for i := range rendered {
		i := i
		g.Go(func() error {
			mn := NewMinifier()
			rendered[i].HTML = mn.MinifyHTML(rendered[i].HTML)
			return nil
		})
	}
	return g.Wait()
}

func minWorkerLimit(workerCount, workItems int) int {
	if workerCount < 1 {
		workerCount = workers.Count()
	}
	if workItems > 0 && workItems < workerCount {
		return workItems
	}
	return workerCount
}

func countMarkdownPages(pages []*engine.Page) int {
	count := 0
	for _, page := range pages {
		if page.RawContent != "" {
			count++
		}
	}
	return count
}

func sortedCollectionNames(collections map[string]*engine.Collection) []string {
	names := make([]string, 0, len(collections))
	for name := range collections {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedTaxonomyNames(taxonomies map[string]*engine.Taxonomy) []string {
	names := make([]string, 0, len(taxonomies))
	for name := range taxonomies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ContentRebuild performs an incremental rebuild for content-only changes.
// It re-parses only the changed files, patches them into the existing collections,
// and renders/writes only the dirty pages. Falls back to full Build() on edge cases.
func (b *SiteBuilder) ContentRebuild(changedPaths []string) (*engine.BuildResult, error) {
	start := time.Now()

	if !b.built || b.lastCollections == nil || b.lastAssetPipeline == nil {
		return b.Build()
	}

	contentDir := filepath.Join(b.projectDir, "content")
	if b.config.Content.Dir != "" {
		if filepath.IsAbs(b.config.Content.Dir) {
			contentDir = b.config.Content.Dir
		} else {
			contentDir = filepath.Join(b.projectDir, b.config.Content.Dir)
		}
	}

	// Skip fallback pages: they share FilePath with their source page and
	// would overwrite the real page entry in the map.
	oldByPath := make(map[string]*engine.Page, len(b.lastAllPages))
	for _, p := range b.lastAllPages {
		if p.FilePath != "" && !p.IsFallback {
			oldByPath[p.FilePath] = p
		}
	}

	type parsedEntry struct {
		filePath string
		newPage  *engine.Page
		old      *engine.Page
	}
	var parsed []parsedEntry
	var warnings []engine.ValidationWarning

	for _, path := range changedPaths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(b.projectDir, path)
		}
		path = filepath.Clean(path)

		base := filepath.Base(path)
		if base == "_index.md" || base == "404.md" || (strings.HasPrefix(base, "404.") && filepath.Ext(base) == ".md") {
			log.Printf("ContentRebuild: structural content changed (%s), falling back to full rebuild", base)
			return b.Build()
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Printf("ContentRebuild: file deleted, falling back to full rebuild")
			return b.Build()
		}

		cf, err := b.scanner.ClassifyFile(contentDir, path)
		if err != nil {
			log.Printf("ContentRebuild: ClassifyFile error: %v, falling back", err)
			return b.Build()
		}

		old := oldByPath[path]
		if old == nil {
			log.Printf("ContentRebuild: new file detected, falling back to full rebuild")
			return b.Build()
		}
		if len(old.Resources) > 0 || cf.IsBundle {
			log.Printf("ContentRebuild: bundle content changed, falling back to full rebuild")
			return b.Build()
		}

		// Determine collection config for parsing.
		var collCfg *engine.CollectionConfig
		var schema *engine.FrontmatterSchema
		if cf.CollectionName != "" {
			inferred := collection.InferCollection(cf.CollectionName)
			if b.config.Collections != nil {
				if scfg, ok := b.config.Collections[cf.CollectionName]; ok {
					inferred = collection.MergeCollectionConfig(inferred, scfg)
				}
			}
			collCfg = inferred
			schema, _ = content.LoadSchema(filepath.Join(contentDir, cf.CollectionName))
		}

		newPage, pageWarnings, err := collection.BuildSinglePage(
			cf, contentDir, collCfg, schema,
			b.config.Content.SummaryLength,
			string(b.config.Build.LastUpdated),
		)
		if err != nil {
			log.Printf("ContentRebuild: parse error for %s: %v, falling back", path, err)
			return b.Build()
		}
		warnings = append(warnings, pageWarnings...)

		includeDrafts := config.BoolVal(b.config.Build.Drafts, false)
		includeFuture := config.BoolVal(b.config.Build.Future, false)
		includeExpired := config.BoolVal(b.config.Build.Expired, false)
		now := time.Now()
		wasExcluded := content.ShouldExclude(old.Draft, old.PublishDate, old.ExpiryDate, includeDrafts, includeFuture, includeExpired, now)
		isExcluded := content.ShouldExclude(newPage.Draft, newPage.PublishDate, newPage.ExpiryDate, includeDrafts, includeFuture, includeExpired, now)
		if wasExcluded != isExcluded {
			log.Printf("ContentRebuild: draft/publish status changed, falling back to full rebuild")
			return b.Build()
		}

		if reason := incrementalEligibilityFailure(old, newPage, cf); reason != "" {
			log.Printf("ContentRebuild: %s, falling back to full rebuild", reason)
			return b.Build()
		}

		parsed = append(parsed, parsedEntry{
			filePath: path,
			newPage:  newPage,
			old:      old,
		})
	}

	patchedAllPages := make([]*engine.Page, len(b.lastAllPages))
	copy(patchedAllPages, b.lastAllPages)

	indexByPath := make(map[string]int, len(patchedAllPages))
	for i, p := range patchedAllPages {
		if p.FilePath != "" && !p.IsFallback {
			indexByPath[p.FilePath] = i
		}
	}

	dirtyPermalinks := make(map[string]struct{})
	changedByPath := make(map[string]*engine.Page, len(parsed))

	for _, e := range parsed {
		idx, ok := indexByPath[e.filePath]
		if !ok {
			return b.Build()
		}

		preserveStablePageState(e.newPage, e.old)

		patchedAllPages[idx] = e.newPage
		changedByPath[e.filePath] = e.newPage

		col := e.old.Collection
		if col != nil {
			replacePagePointer(&col.Pages, e.old, e.newPage)
			replacePagePointer(&col.Featured, e.old, e.newPage)
		}
		if e.old.Section != nil {
			replacePagePointer(&e.old.Section.Pages, e.old, e.newPage)
		}

		dirtyPermalinks[e.newPage.RelPermalink] = struct{}{}
		addCollectionListDirty(col, dirtyPermalinks)
	}

	// Fallback pages are regenerated below; taxonomy stubs are rendered
	// separately in the taxonomy section. Keeping them in patchedAllPages
	// would double-render and cause the dirty set to grow across rebuilds.
	{
		n := 0
		for _, p := range patchedAllPages {
			if !p.IsFallback && p.Kind != engine.KindTaxonomy && p.Kind != engine.KindTerm {
				patchedAllPages[n] = p
				n++
			}
		}
		patchedAllPages = patchedAllPages[:n]
	}

	isMultiLang := b.config.I18n.IsMultiLang()
	if isMultiLang {
		defaultLang := b.config.I18n.GetDefaultLanguage()

		langCodes := b.config.I18n.LanguageCodes()
		fallbacks := i18n.GenerateFallbacks(patchedAllPages, langCodes, defaultLang)
		patchedAllPages = append(patchedAllPages, fallbacks...)

		for _, fb := range fallbacks {
			if fb.FilePath != "" {
				if _, ok := changedByPath[fb.FilePath]; ok {
					dirtyPermalinks[fb.RelPermalink] = struct{}{}
				}
			}
		}

		weights := make(map[string]int)
		for code, lc := range b.config.I18n.Languages {
			weights[code] = lc.Weight
		}
		i18n.LinkTranslations(patchedAllPages, weights)
	}

	newTaxonomies := taxonomy.BuildTaxonomies(patchedAllPages, b.config.Taxonomies)
	if taxWarnings, err := taxonomy.EnrichTaxonomies(newTaxonomies, b.config.Taxonomies, b.projectDir); err != nil {
		return b.Build()
	} else {
		emitTaxonomyWarnings(taxWarnings)
	}

	collection.LinkVersions(patchedAllPages)

	newPageIndex := content.BuildPageIndex(patchedAllPages)
	newPageIndex.AddAssets(filepath.Join(b.projectDir, consts.DirStatic))
	populatePageIndexHeadings(newPageIndex, patchedAllPages)

	for _, e := range parsed {
		addTaxonomyDirtyForPage(e.filePath, newTaxonomies, b.config.Taxonomies, dirtyPermalinks)
	}

	b.lastSiteCtx.Collections = b.lastCollections
	b.lastSiteCtx.Taxonomies = newTaxonomies
	b.lastSiteCtx.Pages = patchedAllPages
	b.lastSiteCtx.BuildTime = time.Now()
	b.tmplEngine.SetSiteContext(b.lastSiteCtx)
	b.tmplEngine.SetPageIndex(newPageIndex)

	// Load template engine (skips re-parsing via loaded flag, just clears caches).
	resolver := &engine.ThemeResolver{
		ProjectDir: b.projectDir,
		ThemeName:  b.config.Theme.Name,
		EmbeddedFS: b.embeddedFS,
	}
	if err := b.tmplEngine.Load(resolver); err != nil {
		return b.Build()
	}

	mergedValidation := make(map[string]engine.ValidationEntry, len(b.lastValidationData))
	for k, v := range b.lastValidationData {
		mergedValidation[k] = v
	}

	for i := range parsed {
		e := &parsed[i]
		if e.newPage.RawContent == "" {
			delete(mergedValidation, e.newPage.RelPermalink)
			continue
		}
		processed, _ := b.lastScProcessor.Process(
			e.newPage.RawContent, e.newPage, b.lastSiteCtx, b.mdRenderer,
		)
		hash := ContentHash(processed + b.lastShortcodesHash)
		if b.lastPageCache != nil {
			if entry := b.lastPageCache.Get(hash); entry != nil {
				e.newPage.Content = htmltemplate.HTML(entry.HTML)
				e.newPage.Headings = entry.Headings
				e.newPage.HasCodeBlocks = entry.HasCodeBlocks
				e.newPage.HasImages = entry.HasImages
				updateValidationEntry(mergedValidation, e.newPage, entry.Links)
				continue
			}
		}
		lookup := markdown.ImageLookupForPage(e.newPage, b.lastAssetPipeline.ImageProcessor())
		b.mdRenderer.SetImageLookup(lookup)
		result, err := b.mdRenderer.Render(processed)
		if err != nil {
			log.Printf("ContentRebuild: markdown render error: %v, falling back", err)
			return b.Build()
		}
		e.newPage.Content = htmltemplate.HTML(result.HTML)
		e.newPage.Headings = result.Headings
		e.newPage.HasCodeBlocks = result.HasCodeBlocks
		e.newPage.HasImages = result.HasImages
		if b.lastPageCache != nil {
			b.lastPageCache.Put(hash, &CacheEntry{
				ContentHash:   hash,
				HTML:          result.HTML,
				Headings:      result.Headings,
				HasCodeBlocks: result.HasCodeBlocks,
				HasImages:     result.HasImages,
				Links:         result.Links,
			})
		}
		updateValidationEntry(mergedValidation, e.newPage, result.Links)
		setPageIndexHeadings(newPageIndex, e.newPage)
	}
	copyRenderedContentToFallbacks(patchedAllPages, changedByPath, newPageIndex)

	var dirtyPages []*engine.Page
	for _, page := range patchedAllPages {
		if _, isDirty := dirtyPermalinks[page.RelPermalink]; !isDirty {
			continue
		}
		if page.Params != nil {
			if r, ok := page.Params["render"].(bool); ok && !r {
				continue
			}
		}
		dirtyPages = append(dirtyPages, page)
	}

	dirtyRendered, dirtyAliases, err := b.renderPages(dirtyPages, b.lastSiteCtx, true, workers.Count())
	if err != nil {
		log.Printf("ContentRebuild: template render error: %v, falling back", err)
		return b.Build()
	}

	if err := b.renderDirtyCollectionPagination(b.lastCollections, dirtyPermalinks, b.lastSiteCtx, &dirtyRendered); err != nil {
		log.Printf("ContentRebuild: paginated list render error: %v, falling back", err)
		return b.Build()
	}

	for taxName, tax := range newTaxonomies {
		cfg := b.config.Taxonomies[taxName]
		if !cfg.ShouldRender() {
			continue
		}
		if _, isDirty := dirtyPermalinks[tax.Permalink]; isDirty {
			termEntries := taxonomy.ComputeTermEntries(tax)
			taxStub := &engine.Page{
				Title: tax.Name, Kind: engine.KindTaxonomy,
				Permalink: tax.Permalink, RelPermalink: tax.Permalink,
				Params: map[string]any{
					"__taxonomy": tax, "__term_entries": termEntries,
				},
			}
			rd := sardetemplate.BuildRouteData(taxStub, b.lastSiteCtx, b.themeConfig)
			html, err := b.tmplEngine.Render(rd.Template, rd)
			if err != nil {
				return b.Build()
			}
			dirtyRendered = append(dirtyRendered, RenderedPage{
				Page: taxStub, HTML: html,
				OutPath: PageOutputPath(tax.Permalink),
			})
		}
		paginateBy := cfg.PaginateBy
		if paginateBy <= 0 {
			paginateBy = 10
		}
		for _, term := range tax.Terms {
			if _, isDirty := dirtyPermalinks[term.Permalink]; isDirty {
				termStub := &engine.Page{
					Title: term.Label, Kind: engine.KindTerm,
					Permalink: term.Permalink, RelPermalink: term.Permalink,
					Params: map[string]any{"__taxonomy": tax, "__taxonomy_term": term},
				}
				rd := sardetemplate.BuildRouteData(termStub, b.lastSiteCtx, b.themeConfig)
				html, err := b.tmplEngine.Render(rd.Template, rd)
				if err != nil {
					return b.Build()
				}
				dirtyRendered = append(dirtyRendered, RenderedPage{
					Page: termStub, HTML: html,
					OutPath: PageOutputPath(term.Permalink),
				})
			}
			totalTermPages := (len(term.Pages) + paginateBy - 1) / paginateBy
			for n := 2; n <= totalTermPages; n++ {
				permalink := fmt.Sprintf("%spage/%d/", term.Permalink, n)
				if _, isDirty := dirtyPermalinks[permalink]; !isDirty {
					continue
				}
				paginatedStub := &engine.Page{
					Title:        term.Label,
					Kind:         engine.KindTerm,
					Permalink:    permalink,
					RelPermalink: permalink,
					Params: map[string]any{
						"__taxonomy":                tax,
						"__taxonomy_term":           term,
						consts.PaginationCurrentKey: n,
					},
				}
				rd := sardetemplate.BuildRouteData(paginatedStub, b.lastSiteCtx, b.themeConfig)
				html, err := b.tmplEngine.Render(rd.Template, rd)
				if err != nil {
					return b.Build()
				}
				dirtyRendered = append(dirtyRendered, RenderedPage{
					Page: paginatedStub, HTML: html,
					OutPath: PageOutputPath(permalink),
				})
			}
		}
	}

	if !b.devMode && config.BoolVal(b.config.Build.Minify, true) {
		mn := NewMinifier()
		for i := range dirtyRendered {
			dirtyRendered[i].HTML = mn.MinifyHTML(dirtyRendered[i].HTML)
		}
	}

	for _, rp := range dirtyRendered {
		if _, err := writeOutputFile(b.lastOutputDir, rp.OutPath, rp.HTML); err != nil {
			return nil, fmt.Errorf("incremental write %s: %w", rp.OutPath, err)
		}
	}
	for aliasPath, target := range dirtyAliases {
		if _, err := writeOutputFile(b.lastOutputDir, PageOutputPath(aliasPath), []byte(redirectHTML(target))); err != nil {
			return nil, fmt.Errorf("incremental write alias %s: %w", aliasPath, err)
		}
	}

	patchedAllPages = appendTaxonomyStubs(patchedAllPages, newTaxonomies, b.config.Taxonomies)
	b.lastSiteCtx.Pages = patchedAllPages

	rebuildLogger := engine.NewBuildLogger()
	var pluginWarnings []engine.ValidationWarning
	buildDoneCtx := &plugin.BuildDoneContext{
		Config:         b.config,
		OutputDir:      b.lastOutputDir,
		Pages:          patchedAllPages,
		Collections:    b.lastCollections,
		Site:           b.lastSiteCtx,
		PageIndex:      newPageIndex,
		ValidationData: mergedValidation,
		DevMode:        b.devMode,
	}
	buildDoneCtx.SetWarnings(&pluginWarnings)
	buildDoneCtx.SetLogger(rebuildLogger)
	if err := b.pluginMgr.RunBuildDone(buildDoneCtx); err != nil {
		return nil, err
	}
	warnings = append(warnings, pluginWarnings...)

	b.lastAllPages = patchedAllPages
	b.lastTaxonomies = newTaxonomies
	b.lastPageIndex = newPageIndex
	b.lastValidationData = mergedValidation

	return &engine.BuildResult{
		PageCount:   len(dirtyRendered),
		Duration:    time.Since(start),
		Warnings:    warnings,
		OutputDir:   b.lastOutputDir,
		LogMessages: rebuildLogger.Messages(),
	}, nil
}

func incrementalEligibilityFailure(old, next *engine.Page, cf content.ContentFile) string {
	if old.RelPermalink != next.RelPermalink {
		return "permalink changed"
	}
	if old.Kind != next.Kind {
		return "content kind changed"
	}
	if collectionName(old) != cf.CollectionName {
		return "collection membership changed"
	}
	if old.Lang != next.Lang || old.LangRelPath != next.LangRelPath {
		return "language identity changed"
	}
	if old.Slug != next.Slug {
		return "slug changed"
	}
	if old.Title != next.Title || !old.Date.Equal(next.Date) || old.Weight != next.Weight {
		return "collection sort or navigation fields changed"
	}
	if old.Draft != next.Draft || !old.PublishDate.Equal(next.PublishDate) || !old.ExpiryDate.Equal(next.ExpiryDate) {
		return "publish state changed"
	}
	if !stringSlicesEqual(old.Aliases, next.Aliases) {
		return "aliases changed"
	}
	if !stringSlicesEqual(old.Tags, next.Tags) || !stringSlicesEqual(old.Categories, next.Categories) {
		return "taxonomy fields changed"
	}
	if boolParam(old.Params, "featured") != boolParam(next.Params, "featured") {
		return "featured status changed"
	}
	if old.SidebarLabel != next.SidebarLabel || old.SidebarHidden != next.SidebarHidden || !reflect.DeepEqual(old.Badge, next.Badge) {
		return "sidebar fields changed"
	}
	for _, key := range []string{
		"render", "template", "layout", "type", "prev", "next",
		"sidebar_group", "sidebar_attrs",
	} {
		if !reflect.DeepEqual(paramValue(old.Params, key), paramValue(next.Params, key)) {
			return key + " changed"
		}
	}
	return ""
}

func collectionName(p *engine.Page) string {
	if p != nil && p.Collection != nil {
		return p.Collection.Name
	}
	return ""
}

func paramValue(params map[string]any, key string) any {
	if params == nil {
		return nil
	}
	return params[key]
}

func boolParam(params map[string]any, key string) bool {
	if v, ok := paramValue(params, key).(bool); ok {
		return v
	}
	return false
}

func preserveStablePageState(next, old *engine.Page) {
	next.Collection = old.Collection
	next.Section = old.Section
	next.PrevPage = old.PrevPage
	next.NextPage = old.NextPage
	next.Siblings = old.Siblings
	next.NavNode = old.NavNode
	next.Version = old.Version
	next.VersionRelPath = old.VersionRelPath
	next.VersionPeers = old.VersionPeers
	next.Translations = old.Translations
}

func replacePagePointer(pages *[]*engine.Page, old, next *engine.Page) {
	if pages == nil {
		return
	}
	for i, p := range *pages {
		if p == old || (p != nil && p.FilePath == old.FilePath && !p.IsFallback) {
			(*pages)[i] = next
		}
	}
}

func addCollectionListDirty(col *engine.Collection, dirty map[string]struct{}) {
	if col == nil || col.IndexPage == nil {
		return
	}
	if col.IndexPage.RelPermalink != "" {
		dirty[col.IndexPage.RelPermalink] = struct{}{}
	}
	if col.Config == nil || col.Config.Paginate <= 0 {
		return
	}
	contentPages := 0
	for _, p := range col.Pages {
		if p.Kind != engine.KindSection {
			contentPages++
		}
	}
	total := (contentPages + col.Config.Paginate - 1) / col.Config.Paginate
	base := col.IndexPage.RelPermalink
	if base == "" {
		base = "/" + col.Name + "/"
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	for n := 2; n <= total; n++ {
		dirty[fmt.Sprintf("%spage/%d/", base, n)] = struct{}{}
	}
}

func addTaxonomyDirtyForPage(filePath string, taxonomies map[string]*engine.Taxonomy, cfg map[string]config.TaxonomyConfig, dirty map[string]struct{}) {
	for taxName, tax := range taxonomies {
		if !cfg[taxName].ShouldRender() {
			continue
		}
		paginateBy := tax.PaginateBy
		if paginateBy <= 0 {
			paginateBy = 10
		}
		for _, term := range tax.Terms {
			found := false
			for _, tp := range term.Pages {
				if tp.FilePath == filePath {
					found = true
					break
				}
			}
			if !found {
				continue
			}
			dirty[tax.Permalink] = struct{}{}
			dirty[term.Permalink] = struct{}{}
			total := (len(term.Pages) + paginateBy - 1) / paginateBy
			for n := 2; n <= total; n++ {
				dirty[fmt.Sprintf("%spage/%d/", term.Permalink, n)] = struct{}{}
			}
		}
	}
}

func updateValidationEntry(data map[string]engine.ValidationEntry, page *engine.Page, links []engine.CollectedLink) {
	if len(links) == 0 {
		delete(data, page.RelPermalink)
		return
	}
	data[page.RelPermalink] = engine.ValidationEntry{Links: links, FilePath: page.FilePath}
}

func populatePageIndexHeadings(idx *content.PageIndex, pages []*engine.Page) {
	for _, p := range pages {
		setPageIndexHeadings(idx, p)
	}
}

func setPageIndexHeadings(idx *content.PageIndex, page *engine.Page) {
	if idx == nil || page == nil || len(page.Headings) == 0 {
		return
	}
	ids := make([]string, len(page.Headings))
	for i, h := range page.Headings {
		ids[i] = h.ID
	}
	idx.SetHeadings(page.RelPermalink, ids)
}

func copyRenderedContentToFallbacks(pages []*engine.Page, changed map[string]*engine.Page, idx *content.PageIndex) {
	for _, p := range pages {
		if !p.IsFallback || p.FilePath == "" {
			continue
		}
		src := changed[p.FilePath]
		if src == nil {
			continue
		}
		p.Content = src.Content
		p.Headings = src.Headings
		p.HasCodeBlocks = src.HasCodeBlocks
		p.HasImages = src.HasImages
		setPageIndexHeadings(idx, p)
	}
}

func (b *SiteBuilder) renderDirtyCollectionPagination(collections map[string]*engine.Collection, dirty map[string]struct{}, site *engine.SiteContext, rendered *[]RenderedPage) error {
	for _, col := range collections {
		if col == nil || col.Config == nil || col.Config.Paginate <= 0 || col.IndexPage == nil {
			continue
		}
		var contentPages []*engine.Page
		for _, p := range col.Pages {
			if p.Kind != engine.KindSection {
				contentPages = append(contentPages, p)
			}
		}
		total := (len(contentPages) + col.Config.Paginate - 1) / col.Config.Paginate
		if total <= 1 {
			continue
		}
		base := col.IndexPage.RelPermalink
		if base == "" {
			base = "/" + col.Name + "/"
		}
		if !strings.HasSuffix(base, "/") {
			base += "/"
		}
		for n := 2; n <= total; n++ {
			permalink := fmt.Sprintf("%spage/%d/", base, n)
			if _, ok := dirty[permalink]; !ok {
				continue
			}
			stub := &engine.Page{
				Title:        col.IndexPage.Title,
				Kind:         engine.KindSection,
				Permalink:    permalink,
				RelPermalink: permalink,
				Collection:   col,
				Section:      col.IndexPage.Section,
				Lang:         col.IndexPage.Lang,
				Params: map[string]any{
					consts.PaginationCurrentKey: n,
				},
			}
			rd := sardetemplate.BuildRouteData(stub, site, b.themeConfig)
			if err := b.pluginMgr.RunBeforeRender(b.config, stub, rd, site); err != nil {
				return err
			}
			html, err := b.tmplEngine.Render(rd.Template, rd)
			if err != nil {
				return err
			}
			*rendered = append(*rendered, RenderedPage{
				Page:    stub,
				HTML:    html,
				OutPath: PageOutputPath(permalink),
			})
		}
	}
	return nil
}

func appendTaxonomyStubs(pages []*engine.Page, taxonomies map[string]*engine.Taxonomy, cfg map[string]config.TaxonomyConfig) []*engine.Page {
	for taxName, tax := range taxonomies {
		if !cfg[taxName].ShouldRender() {
			continue
		}
		pages = append(pages, &engine.Page{
			Title: tax.Name, Kind: engine.KindTaxonomy,
			Permalink: tax.Permalink, RelPermalink: tax.Permalink,
		})
		paginateBy := tax.PaginateBy
		if paginateBy <= 0 {
			paginateBy = 10
		}
		for _, term := range tax.Terms {
			pages = append(pages, &engine.Page{
				Title: term.Label, Kind: engine.KindTerm,
				Permalink: term.Permalink, RelPermalink: term.Permalink,
			})
			total := (len(term.Pages) + paginateBy - 1) / paginateBy
			for n := 2; n <= total; n++ {
				permalink := fmt.Sprintf("%spage/%d/", term.Permalink, n)
				pages = append(pages, &engine.Page{
					Title: term.Label, Kind: engine.KindTerm,
					Permalink: permalink, RelPermalink: permalink,
					Params: map[string]any{consts.PaginationCurrentKey: n},
				})
			}
		}
	}
	return pages
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Validate runs phases 1-4 (Initialize, Discover, Parse, Assemble) without rendering or writing.
func (b *SiteBuilder) Validate() (*ValidateResult, error) {
	start := time.Now()

	contentDir := filepath.Join(b.projectDir, "content")
	if b.config.Content.Dir != "" {
		if filepath.IsAbs(b.config.Content.Dir) {
			contentDir = b.config.Content.Dir
		} else {
			contentDir = filepath.Join(b.projectDir, b.config.Content.Dir)
		}
	}

	// i18n: configure scanner for multi-language detection
	if b.config.I18n.IsMultiLang() {
		langCodes := make(map[string]bool)
		for code := range b.config.I18n.Languages {
			langCodes[code] = true
		}
		b.scanner.Languages = langCodes
		b.scanner.DefaultLang = b.config.I18n.GetDefaultLanguage()
	}

	files, err := b.scanner.DiscoverFiles(contentDir)
	if err != nil {
		return nil, fmt.Errorf("discovering content: %w", err)
	}

	collections, warnings, err := collection.BuildCollections(files, b.config, contentDir)
	if err != nil {
		return nil, fmt.Errorf("building collections: %w", err)
	}

	standalones, err := collection.BuildStandalonePages(files, contentDir, b.config.Content.SummaryLength, string(b.config.Build.LastUpdated))
	if err != nil {
		return nil, fmt.Errorf("building standalone pages: %w", err)
	}

	var allPages []*engine.Page
	for _, col := range collections {
		allPages = append(allPages, col.Pages...)
	}
	allPages = append(allPages, standalones...)

	return &ValidateResult{
		PageCount:   len(allPages),
		Collections: len(collections),
		Warnings:    warnings,
		Pages:       allPages,
		Duration:    time.Since(start),
	}, nil
}

// ValidateResult holds the outcome of a validation run (no rendering or writing).
type ValidateResult struct {
	PageCount   int
	Collections int
	Warnings    []engine.ValidationWarning
	Pages       []*engine.Page
	Duration    time.Duration
}

// langGrouping holds pages grouped by language, preserving order.
type langGrouping struct {
	pages map[string][]*engine.Page
	order []string
}

// groupPagesByLang groups pages by Lang field, with stable ordering.
// Pages without a Lang are placed in a "" group.
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

func groupPagesByLang(pages []*engine.Page) langGrouping {
	result := langGrouping{pages: make(map[string][]*engine.Page)}
	seen := make(map[string]bool)
	for _, p := range pages {
		lang := p.Lang
		result.pages[lang] = append(result.pages[lang], p)
		if !seen[lang] {
			seen[lang] = true
			result.order = append(result.order, lang)
		}
	}
	return result
}

// tabbedCollectionRedirect returns the redirect URL if the page is the root
// index of a tabbed collection, or "" otherwise.
func tabbedCollectionRedirect(page *engine.Page) string {
	col := page.Collection
	if col == nil || !col.IsTabbed || col.IndexPage != page || len(col.Tabs) == 0 {
		return ""
	}
	return col.Tabs[0].Permalink
}

func buildRedirectHTML(target string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html><head>`+
		`<meta http-equiv="refresh" content="0;url=%s">`+
		`<link rel="canonical" href="%s">`+
		`</head><body><p><a href="%s">Continue</a></p></body></html>`,
		target, target, target)
}

const warnCollapseThreshold = 3

func emitTaxonomyWarnings(warnings []string) {
	if len(warnings) == 0 {
		return
	}

	type group struct {
		taxName string
		terms   []string
	}
	groups := make(map[string]*group)
	var order []string

	for _, w := range warnings {
		// Warnings follow: taxonomy "<name>": term "<term>" is not defined in ...
		taxPart, termPart, ok := strings.Cut(w, ": term ")
		if !ok {
			devlog.Warn("taxonomy", "%s", w)
			continue
		}
		taxName := strings.TrimPrefix(taxPart, "taxonomy ")
		taxName = strings.Trim(taxName, "\"")

		term := termPart
		if idx := strings.Index(term, "\" "); idx >= 0 {
			term = term[1:idx] // strip surrounding quote and trailing text
		} else {
			term = strings.Trim(term, "\"")
		}

		g, exists := groups[taxName]
		if !exists {
			g = &group{taxName: taxName}
			groups[taxName] = g
			order = append(order, taxName)
		}
		g.terms = append(g.terms, term)
	}

	for _, name := range order {
		g := groups[name]
		if len(g.terms) <= warnCollapseThreshold {
			for _, t := range g.terms {
				devlog.Warn("taxonomy", "%q: term %q is not defined in data/%s.yml", name, t, name)
			}
		} else {
			devlog.Warn("taxonomy", "%q: %d undefined terms (define in data/%s.yml) — %s", name, len(g.terms), name, strings.Join(g.terms, ", "))
		}
	}
}
