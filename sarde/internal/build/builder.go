package build

import (
	"context"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
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
	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/i18n"
	"github.com/frostybee/sarde/internal/navigation"
	"github.com/frostybee/sarde/internal/plugin"
	"github.com/frostybee/sarde/internal/shortcode"
	"github.com/frostybee/sarde/internal/plugin/announcements"
	"github.com/frostybee/sarde/internal/plugin/clientplugins"
	"github.com/frostybee/sarde/internal/plugin/katex"
	"github.com/frostybee/sarde/internal/plugin/mermaid"
	"github.com/frostybee/sarde/internal/plugin/socialcards"
	"github.com/frostybee/sarde/internal/taxonomy"
	sardetemplate "github.com/frostybee/sarde/internal/template"
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
		tmplEngine:  sardetemplate.NewEngine(),
		pluginMgr:   mgr,
	}
}

// Build executes the full six-phase pipeline and writes output to disk.
func (b *SiteBuilder) Build() (*engine.BuildResult, error) {
	start := time.Now()
	var timings []engine.PhaseTiming
	phaseStart := time.Now()

	// Phase 1: INITIALIZE
	contentDir := filepath.Join(b.projectDir, "content")
	if b.config.Content.Dir != "" {
		if filepath.IsAbs(b.config.Content.Dir) {
			contentDir = b.config.Content.Dir
		} else {
			contentDir = filepath.Join(b.projectDir, b.config.Content.Dir)
		}
	}

	outputDir := filepath.Join(b.projectDir, "dist")
	if b.config.Build.Output != "" {
		outputDir = filepath.Join(b.projectDir, b.config.Build.Output)
	}

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
	files, err := b.scanner.DiscoverFiles(contentDir)
	if err != nil {
		return nil, fmt.Errorf("discovering content: %w", err)
	}
	timings = append(timings, engine.PhaseTiming{Phase: "Discovering content", Duration: time.Since(phaseStart)})
	phaseStart = time.Now()

	// Phase 3: PARSE (collections + standalone)
	collections, warnings, err := collection.BuildCollections(files, b.config, contentDir)
	if err != nil {
		return nil, fmt.Errorf("building collections: %w", err)
	}

	standalones, err := collection.BuildStandalonePages(files, contentDir, b.config.Content.SummaryLength, string(b.config.Build.LastUpdated))
	if err != nil {
		return nil, fmt.Errorf("building standalone pages: %w", err)
	}
	timings = append(timings, engine.PhaseTiming{Phase: "Parsing markdown", Duration: time.Since(phaseStart)})
	phaseStart = time.Now()

	// Phase 4: ASSEMBLE

	// Collect all pages.
	var allPages []*engine.Page
	for _, col := range collections {
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
	for _, w := range taxWarnings {
		fmt.Fprintf(os.Stderr, "WARNING: %s\n", w)
	}

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

	// Phase 4.5: ASSETS
	assetPipeline := asset.NewPipeline(asset.PipelineOptions{
		ProjectDir: b.projectDir,
		OutputDir:  outputDir,
		Config:     b.config,
		ThemeName:  b.config.Theme.Name,
		EmbeddedFS: b.embeddedFS,
		DevMode:    b.devMode,
	})
	for _, page := range allPages {
		if err := assetPipeline.EnhanceResources(page); err != nil {
			return nil, fmt.Errorf("enhancing resources for %s: %w", page.FilePath, err)
		}
	}

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
	parallel := config.BoolVal(b.config.Build.Parallel, true)
	if parallel {
		// Lazily initialize the renderer pool (Goldmark construction is expensive).
		poolSize := runtime.NumCPU()
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

	timings = append(timings, engine.PhaseTiming{Phase: "Assembling site", Duration: time.Since(phaseStart)})
	phaseStart = time.Now()

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

	if parallel {
		// Group pages by language for thread-safe t() function.
		langGroups := groupPagesByLang(renderablePages)
		var mu sync.Mutex

		for _, lang := range langGroups.order {
			pages := langGroups.pages[lang]
			b.tmplEngine.SetCurrentLang(lang)

			g, _ := errgroup.WithContext(context.Background())
			g.SetLimit(runtime.NumCPU())

			for _, page := range pages {
				g.Go(func() error {
					rd := sardetemplate.BuildRouteData(page, siteCtx, b.themeConfig)

					if err := b.pluginMgr.RunBeforeRender(b.config, page, rd, siteCtx); err != nil {
						return err
					}

					var html []byte
					if redirect := tabbedCollectionRedirect(page); redirect != "" {
						html = []byte(buildRedirectHTML(redirect))
					} else {
						var err error
						html, err = b.tmplEngine.Render(rd.Template, rd)
						if err != nil {
							return fmt.Errorf("rendering %s (template %s): %w", page.FilePath, rd.Template, err)
						}
					}

					rp := RenderedPage{
						Page:    page,
						HTML:    html,
						OutPath: PageOutputPath(page.RelPermalink),
					}

					mu.Lock()
					rendered = append(rendered, rp)
					for _, alias := range page.Aliases {
						aliases[alias] = page.RelPermalink
					}
					mu.Unlock()
					return nil
				})
			}
			if err := g.Wait(); err != nil {
				return nil, err
			}
		}
	} else {
		for _, page := range renderablePages {
			rd := sardetemplate.BuildRouteData(page, siteCtx, b.themeConfig)
			b.tmplEngine.SetCurrentLang(rd.Lang)

			if err := b.pluginMgr.RunBeforeRender(b.config, page, rd, siteCtx); err != nil {
				return nil, err
			}

			var html []byte
			if redirect := tabbedCollectionRedirect(page); redirect != "" {
				html = []byte(buildRedirectHTML(redirect))
			} else {
				var err error
				html, err = b.tmplEngine.Render(rd.Template, rd)
				if err != nil {
					return nil, fmt.Errorf("rendering %s (template %s): %w", page.FilePath, rd.Template, err)
				}
			}

			rendered = append(rendered, RenderedPage{
				Page:    page,
				HTML:    html,
				OutPath: PageOutputPath(page.RelPermalink),
			})

			for _, alias := range page.Aliases {
				aliases[alias] = page.RelPermalink
			}
		}
	}

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
			rd := sardetemplate.BuildRouteData(stub, siteCtx, b.themeConfig)
			b.tmplEngine.SetCurrentLang(rd.Lang)
			if err := b.pluginMgr.RunBeforeRender(b.config, stub, rd, siteCtx); err != nil {
				return nil, err
			}
			html, err := b.tmplEngine.Render(rd.Template, rd)
			if err != nil {
				return nil, fmt.Errorf("rendering paginated list %s: %w", permalink, err)
			}
			rendered = append(rendered, RenderedPage{
				Page:    stub,
				HTML:    html,
				OutPath: PageOutputPath(permalink),
			})
			paginatorPages++
		}
	}

	// Synthesize taxonomy index and term pages.
	var taxonomyPages []*engine.Page
	for taxName, tax := range taxonomies {
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
		rd := sardetemplate.BuildRouteData(taxStub, siteCtx, b.themeConfig)
		b.tmplEngine.SetCurrentLang(rd.Lang)
		if err := b.pluginMgr.RunBeforeRender(b.config, taxStub, rd, siteCtx); err != nil {
			return nil, err
		}
		html, err := b.tmplEngine.Render(rd.Template, rd)
		if err != nil {
			return nil, fmt.Errorf("rendering taxonomy index %s: %w", tax.Permalink, err)
		}
		rendered = append(rendered, RenderedPage{
			Page:    taxStub,
			HTML:    html,
			OutPath: PageOutputPath(tax.Permalink),
		})
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
			rd := sardetemplate.BuildRouteData(termStub, siteCtx, b.themeConfig)
			b.tmplEngine.SetCurrentLang(rd.Lang)
			if err := b.pluginMgr.RunBeforeRender(b.config, termStub, rd, siteCtx); err != nil {
				return nil, err
			}
			html, err := b.tmplEngine.Render(rd.Template, rd)
			if err != nil {
				return nil, fmt.Errorf("rendering term page %s: %w", term.Permalink, err)
			}
			rendered = append(rendered, RenderedPage{
				Page:    termStub,
				HTML:    html,
				OutPath: PageOutputPath(term.Permalink),
			})
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
						"__taxonomy":              tax,
						"__taxonomy_term":          term,
						consts.PaginationCurrentKey: n,
					},
				}
				rd := sardetemplate.BuildRouteData(paginatedStub, siteCtx, b.themeConfig)
				b.tmplEngine.SetCurrentLang(rd.Lang)
				if err := b.pluginMgr.RunBeforeRender(b.config, paginatedStub, rd, siteCtx); err != nil {
					return nil, err
				}
				html, err := b.tmplEngine.Render(rd.Template, rd)
				if err != nil {
					return nil, fmt.Errorf("rendering paginated term %s: %w", permalink, err)
				}
				rendered = append(rendered, RenderedPage{
					Page:    paginatedStub,
					HTML:    html,
					OutPath: PageOutputPath(permalink),
				})
				taxonomyPages = append(taxonomyPages, paginatedStub)
				paginatorPages++
			}
		}
	}

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
		b.tmplEngine.SetCurrentLang(lang)
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

	// HTML minification (production builds only).
	if !b.devMode && config.BoolVal(b.config.Build.Minify, true) {
		mn := NewMinifier()
		for i := range rendered {
			rendered[i].HTML = mn.MinifyHTML(rendered[i].HTML)
		}
	}

	timings = append(timings, engine.PhaseTiming{Phase: "Rendering templates", Duration: time.Since(phaseStart)})
	phaseStart = time.Now()

	// Phase 6: WRITE
	clean := config.BoolVal(b.config.Build.Clean, true)
	var tracker *OutputTracker
	if clean && !b.devMode {
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

	// Write bundle assets (images, PDFs, etc. co-located with content).
	if err := assetPipeline.WriteBundleAssets(allPages, outputDir, trackFn); err != nil {
		return nil, fmt.Errorf("writing bundle assets: %w", err)
	}

	// Copy processed image variants from cache to output.
	processedImages, err := assetPipeline.WriteProcessedImages(outputDir, trackFn)
	if err != nil {
		return nil, fmt.Errorf("writing processed images: %w", err)
	}

	// Write bundled CSS/JS files to output.
	if err := assetPipeline.WriteBundledFiles(outputDir, trackFn); err != nil {
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

	timings = append(timings, engine.PhaseTiming{Phase: "Writing output", Duration: time.Since(phaseStart)})

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

	// Prune orphaned files from previous builds.
	if tracker != nil {
		tracker.Prune(outputDir)
	}
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

	// ── Edge case detection ────────────────────────────────────────────
	for _, p := range changedPaths {
		base := filepath.Base(p)
		if base == "_index.md" {
			log.Printf("ContentRebuild: section index changed, falling back to full rebuild")
			return b.Build()
		}
	}
	// ── Build lookup for old pages by file path ────────────────────────
	// Skip fallback pages: they share FilePath with their source page and
	// would overwrite the real page entry in the map.
	oldByPath := make(map[string]*engine.Page, len(b.lastAllPages))
	for _, p := range b.lastAllPages {
		if p.FilePath != "" && !p.IsFallback {
			oldByPath[p.FilePath] = p
		}
	}

	// ── Classify and re-parse changed files ────────────────────────────
	type parsedEntry struct {
		filePath string
		newPage  *engine.Page
		old      *engine.Page
	}
	var parsed []parsedEntry

	for _, path := range changedPaths {
		// Check if file still exists (deletion → fallback).
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

		newPage, _, err := collection.BuildSinglePage(
			cf, contentDir, collCfg, schema,
			b.config.Content.SummaryLength,
			string(b.config.Build.LastUpdated),
		)
		if err != nil {
			log.Printf("ContentRebuild: parse error for %s: %v, falling back", path, err)
			return b.Build()
		}

		// Check draft/publish status change → fallback.
		includeDrafts := config.BoolVal(b.config.Build.Drafts, false)
		includeFuture := config.BoolVal(b.config.Build.Future, false)
		now := time.Now()
		wasExcluded := content.ShouldExclude(old.Draft, old.PublishDate, includeDrafts, includeFuture, now)
		isExcluded := content.ShouldExclude(newPage.Draft, newPage.PublishDate, includeDrafts, includeFuture, now)
		if wasExcluded != isExcluded {
			log.Printf("ContentRebuild: draft/publish status changed, falling back to full rebuild")
			return b.Build()
		}

		parsed = append(parsed, parsedEntry{filePath: path, newPage: newPage, old: old})
	}

	// ── Patch allPages (shallow copy) ──────────────────────────────────
	patchedAllPages := make([]*engine.Page, len(b.lastAllPages))
	copy(patchedAllPages, b.lastAllPages)

	indexByPath := make(map[string]int, len(patchedAllPages))
	for i, p := range patchedAllPages {
		if p.FilePath != "" && !p.IsFallback {
			indexByPath[p.FilePath] = i
		}
	}

	dirtyPermalinks := make(map[string]struct{})

	for _, e := range parsed {
		idx, ok := indexByPath[e.filePath]
		if !ok {
			return b.Build()
		}

		// Preserve backrefs from the old page.
		e.newPage.Collection = e.old.Collection
		e.newPage.Section = e.old.Section

		// Detect what changed for dirty set computation.
		titleChanged := e.old.Title != e.newPage.Title
		weightChanged := e.old.Weight != e.newPage.Weight
		sidebarChanged := e.old.SidebarLabel != e.newPage.SidebarLabel ||
			e.old.SidebarHidden != e.newPage.SidebarHidden

		// Replace in allPages.
		patchedAllPages[idx] = e.newPage

		// Patch collection.Pages.
		col := e.old.Collection
		if col != nil {
			for i, cp := range col.Pages {
				if cp.FilePath == e.filePath {
					col.Pages[i] = e.newPage
					break
				}
			}

			// Re-sort and re-wire prev/next if sort-affecting fields changed.
			if weightChanged || titleChanged {
				sortBy := "weight"
				sortOrder := ""
				if col.Config != nil {
					if col.Config.SortBy != "" {
						sortBy = col.Config.SortBy
					}
					sortOrder = col.Config.SortOrder
				}
				collection.SortPages(col.Pages, sortBy, sortOrder)
				if col.Config != nil && engine.LayoutHasSidebar(col.Config.Layout) {
					navigation.WirePrevNextFromTree(col.NavTree)
				} else {
					collection.WirePrevNext(col.Pages)
				}
			}
		}

		// ── Dirty set computation ──────────────────────────────────────
		dirtyPermalinks[e.newPage.RelPermalink] = struct{}{}

		if titleChanged {
			if e.newPage.PrevPage != nil {
				dirtyPermalinks[e.newPage.PrevPage.RelPermalink] = struct{}{}
			}
			if e.newPage.NextPage != nil {
				dirtyPermalinks[e.newPage.NextPage.RelPermalink] = struct{}{}
			}
		}

		if sidebarChanged || weightChanged {
			if col != nil {
				for _, cp := range col.Pages {
					dirtyPermalinks[cp.RelPermalink] = struct{}{}
				}
			}
		}
	}

	// ── Strip stale synthetic pages ────────────────────────────────────
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

	// ── i18n: regenerate fallbacks and re-link translations ────────────
	isMultiLang := b.config.I18n.IsMultiLang()
	if isMultiLang {
		defaultLang := b.config.I18n.GetDefaultLanguage()

		// Regenerate fallbacks from the updated real pages.
		langCodes := b.config.I18n.LanguageCodes()
		fallbacks := i18n.GenerateFallbacks(patchedAllPages, langCodes, defaultLang)
		patchedAllPages = append(patchedAllPages, fallbacks...)

		// Mark new fallback pages of changed files as dirty.
		changedSet := make(map[string]struct{}, len(parsed))
		for _, e := range parsed {
			changedSet[e.filePath] = struct{}{}
		}
		for _, fb := range fallbacks {
			if fb.FilePath != "" {
				if _, ok := changedSet[fb.FilePath]; ok {
					dirtyPermalinks[fb.RelPermalink] = struct{}{}
				}
			}
		}

		// Re-link translations.
		weights := make(map[string]int)
		for code, lc := range b.config.I18n.Languages {
			weights[code] = lc.Weight
		}
		i18n.LinkTranslations(patchedAllPages, weights)
	}

	log.Printf("[diag] after strip+fallback: patchedAllPages=%d, dirtyPermalinks=%d", len(patchedAllPages), len(dirtyPermalinks))

	// ── Rebuild lightweight global state (all O(n), fast) ──────────────
	newTaxonomies := taxonomy.BuildTaxonomies(patchedAllPages, b.config.Taxonomies)
	if _, err := taxonomy.EnrichTaxonomies(newTaxonomies, b.config.Taxonomies, b.projectDir); err != nil {
		return b.Build()
	}

	collection.LinkVersions(patchedAllPages)

	newPageIndex := content.BuildPageIndex(patchedAllPages)
	newPageIndex.AddAssets(filepath.Join(b.projectDir, consts.DirStatic))

	// Mark dirty taxonomy permalinks for changed pages.
	for _, e := range parsed {
		for taxName, tax := range newTaxonomies {
			cfg := b.config.Taxonomies[taxName]
			if !cfg.ShouldRender() {
				continue
			}
			for _, term := range tax.Terms {
				for _, tp := range term.Pages {
					if tp.FilePath == e.filePath {
						dirtyPermalinks[term.Permalink] = struct{}{}
						dirtyPermalinks[tax.Permalink] = struct{}{}
						break
					}
				}
			}
		}
	}

	log.Printf("[diag] after taxonomy dirty: dirtyPermalinks=%d", len(dirtyPermalinks))

	// ── Update SiteContext in-place ────────────────────────────────────
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

	// ── Markdown render for changed pages only ─────────────────────────
	for _, e := range parsed {
		if e.newPage.RawContent == "" {
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
			})
		}
		if len(e.newPage.Headings) > 0 {
			ids := make([]string, len(e.newPage.Headings))
			for i, h := range e.newPage.Headings {
				ids[i] = h.ID
			}
			newPageIndex.SetHeadings(e.newPage.RelPermalink, ids)
		}
	}

	// ── Template render for dirty set only ─────────────────────────────
	var dirtyRendered []RenderedPage
	dirtyAliases := make(map[string]string)

	for _, page := range patchedAllPages {
		if _, isDirty := dirtyPermalinks[page.RelPermalink]; !isDirty {
			continue
		}
		if page.Params != nil {
			if r, ok := page.Params["render"].(bool); ok && !r {
				continue
			}
		}
		rd := sardetemplate.BuildRouteData(page, b.lastSiteCtx, b.themeConfig)
		b.tmplEngine.SetCurrentLang(rd.Lang)
		if err := b.pluginMgr.RunBeforeRender(b.config, page, rd, b.lastSiteCtx); err != nil {
			return b.Build()
		}
		var html []byte
		if redirect := tabbedCollectionRedirect(page); redirect != "" {
			html = []byte(buildRedirectHTML(redirect))
		} else {
			var err error
			html, err = b.tmplEngine.Render(rd.Template, rd)
			if err != nil {
				log.Printf("ContentRebuild: template render error: %v, falling back", err)
				return b.Build()
			}
		}
		dirtyRendered = append(dirtyRendered, RenderedPage{
			Page: page, HTML: html,
			OutPath: PageOutputPath(page.RelPermalink),
		})
		for _, alias := range page.Aliases {
			dirtyAliases[alias] = page.RelPermalink
		}
	}

	// Render dirty taxonomy pages.
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
			b.tmplEngine.SetCurrentLang(rd.Lang)
			html, err := b.tmplEngine.Render(rd.Template, rd)
			if err != nil {
				return b.Build()
			}
			dirtyRendered = append(dirtyRendered, RenderedPage{
				Page: taxStub, HTML: html,
				OutPath: PageOutputPath(tax.Permalink),
			})
		}
		for _, term := range tax.Terms {
			if _, isDirty := dirtyPermalinks[term.Permalink]; !isDirty {
				continue
			}
			termStub := &engine.Page{
				Title: term.Label, Kind: engine.KindTerm,
				Permalink: term.Permalink, RelPermalink: term.Permalink,
				Params: map[string]any{"__taxonomy": tax, "__taxonomy_term": term},
			}
			rd := sardetemplate.BuildRouteData(termStub, b.lastSiteCtx, b.themeConfig)
			b.tmplEngine.SetCurrentLang(rd.Lang)
			html, err := b.tmplEngine.Render(rd.Template, rd)
			if err != nil {
				return b.Build()
			}
			dirtyRendered = append(dirtyRendered, RenderedPage{
				Page: termStub, HTML: html,
				OutPath: PageOutputPath(term.Permalink),
			})
		}
	}

	// ── Write dirty pages only ─────────────────────────────────────────
	for _, rp := range dirtyRendered {
		outPath := filepath.Join(b.lastOutputDir, filepath.FromSlash(rp.OutPath))
		if err := writeFile(outPath, rp.HTML); err != nil {
			return nil, fmt.Errorf("incremental write %s: %w", rp.OutPath, err)
		}
	}
	for aliasPath, target := range dirtyAliases {
		outPath := filepath.Join(b.lastOutputDir, filepath.FromSlash(PageOutputPath(aliasPath)))
		if err := writeFile(outPath, []byte(redirectHTML(target))); err != nil {
			return nil, fmt.Errorf("incremental write alias %s: %w", aliasPath, err)
		}
	}

	// Re-add taxonomy stubs to patchedAllPages so BuildDone plugins
	// (sitemap, search, RSS) see all pages including taxonomy pages.
	for _, tax := range newTaxonomies {
		patchedAllPages = append(patchedAllPages, &engine.Page{
			Title: tax.Name, Kind: engine.KindTaxonomy,
			Permalink: tax.Permalink, RelPermalink: tax.Permalink,
		})
		for _, term := range tax.Terms {
			patchedAllPages = append(patchedAllPages, &engine.Page{
				Title: term.Label, Kind: engine.KindTerm,
				Permalink: term.Permalink, RelPermalink: term.Permalink,
			})
		}
	}
	b.lastSiteCtx.Pages = patchedAllPages

	// ── BuildDone plugins — always over full page set ──────────────────
	mergedValidation := make(map[string]engine.ValidationEntry, len(b.lastValidationData))
	for k, v := range b.lastValidationData {
		mergedValidation[k] = v
	}
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

	// ── Persist updated state ──────────────────────────────────────────
	b.lastAllPages = patchedAllPages
	b.lastTaxonomies = newTaxonomies
	b.lastPageIndex = newPageIndex
	b.lastValidationData = mergedValidation

	return &engine.BuildResult{
		PageCount:   len(dirtyRendered),
		Duration:    time.Since(start),
		Warnings:    pluginWarnings,
		OutputDir:   b.lastOutputDir,
		LogMessages: rebuildLogger.Messages(),
	}, nil
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
