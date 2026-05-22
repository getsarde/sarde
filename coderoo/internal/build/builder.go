package build

import (
	"context"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/coderoo-dev/coderoo/embedded"
	"github.com/coderoo-dev/coderoo/internal/asset"
	"github.com/coderoo-dev/coderoo/internal/collection"
	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/internal/consts"
	"github.com/coderoo-dev/coderoo/internal/content"
	"github.com/coderoo-dev/coderoo/internal/content/markdown"
	"github.com/coderoo-dev/coderoo/internal/engine"
	"github.com/coderoo-dev/coderoo/internal/i18n"
	"github.com/coderoo-dev/coderoo/internal/plugin"
	"github.com/coderoo-dev/coderoo/internal/plugin/announcements"
	"github.com/coderoo-dev/coderoo/internal/plugin/clientplugins"
	"github.com/coderoo-dev/coderoo/internal/plugin/katex"
	"github.com/coderoo-dev/coderoo/internal/plugin/mermaid"
	"github.com/coderoo-dev/coderoo/internal/taxonomy"
	coderootemplate "github.com/coderoo-dev/coderoo/internal/template"
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

	scanner    *content.Scanner
	mdRenderer *markdown.Renderer
	tmplEngine *coderootemplate.Engine
	pluginMgr  *plugin.Manager
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
		tmplEngine:  coderootemplate.NewEngine(),
		pluginMgr:   mgr,
	}
}

// Build executes the full six-phase pipeline and writes output to disk.
func (b *SiteBuilder) Build() (*engine.BuildResult, error) {
	start := time.Now()

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

	// Plugin hook: ConfigSetup (serial, after config resolved).
	if err := b.pluginMgr.RunConfigSetup(b.config); err != nil {
		return nil, err
	}

	// Phase 2: DISCOVER
	files, err := b.scanner.DiscoverFiles(contentDir)
	if err != nil {
		return nil, fmt.Errorf("discovering content: %w", err)
	}

	// Phase 3: PARSE (collections + standalone)
	collections, warnings, err := collection.BuildCollections(files, b.config, contentDir)
	if err != nil {
		return nil, fmt.Errorf("building collections: %w", err)
	}

	standalones, err := collection.BuildStandalonePages(files, contentDir, b.config.Content.SummaryLength, string(b.config.Build.LastUpdated))
	if err != nil {
		return nil, fmt.Errorf("building standalone pages: %w", err)
	}

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

	// Markdown render cache (dev mode only — production builds have image processing side effects).
	var pageCache *PageCache
	if b.devMode && config.BoolVal(b.config.Build.Cache, true) {
		pageCache = NewPageCache(b.projectDir)
	}

	// Render markdown for all pages (after asset enhancement so image
	// renderer can access processed resource data for <picture> generation).
	parallel := config.BoolVal(b.config.Build.Parallel, true)
	if parallel {
		// Create a pool of renderers (Goldmark construction is expensive).
		poolSize := runtime.NumCPU()
		rendererPool := make(chan *markdown.Renderer, poolSize)
		for i := 0; i < poolSize; i++ {
			rendererPool <- markdown.NewRendererFromConfig(markdown.RendererConfig{
				BlockedHrefSchemes: b.config.Security.BlockedHrefSchemes,
				HeadingLinks:       config.BoolVal(b.config.Site.HeadingLinks, true),
			})
		}

		g, _ := errgroup.WithContext(context.Background())
		g.SetLimit(poolSize)
		for _, page := range allPages {
			if page.RawContent == "" {
				continue
			}
			g.Go(func() error {
				// Check cache first.
				hash := ContentHash(page.RawContent)
				if pageCache != nil {
					if entry := pageCache.Get(hash); entry != nil {
						page.Content = htmltemplate.HTML(entry.HTML)
						page.Headings = entry.Headings
						page.HasCodeBlocks = entry.HasCodeBlocks
						page.HasImages = entry.HasImages
						return nil
					}
				}

				// Borrow a renderer from the pool.
				renderer := <-rendererPool
				lookup := markdown.ImageLookupForPage(page, assetPipeline.ImageProcessor())
				renderer.SetImageLookup(lookup)

				result, err := renderer.Render(page.RawContent)
				rendererPool <- renderer // return to pool
				if err != nil {
					return fmt.Errorf("rendering markdown for %s: %w", page.FilePath, err)
				}
				page.Content = htmltemplate.HTML(result.HTML)
				page.Headings = result.Headings
				page.HasCodeBlocks = result.HasCodeBlocks
				page.HasImages = result.HasImages

				// Store in cache.
				if pageCache != nil {
					pageCache.Put(hash, &CacheEntry{
						ContentHash:   hash,
						HTML:          result.HTML,
						Headings:      result.Headings,
						HasCodeBlocks: result.HasCodeBlocks,
						HasImages:     result.HasImages,
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

			hash := ContentHash(page.RawContent)
			if pageCache != nil {
				if entry := pageCache.Get(hash); entry != nil {
					page.Content = htmltemplate.HTML(entry.HTML)
					page.Headings = entry.Headings
					page.HasCodeBlocks = entry.HasCodeBlocks
					page.HasImages = entry.HasImages
					continue
				}
			}

			lookup := markdown.ImageLookupForPage(page, assetPipeline.ImageProcessor())
			b.mdRenderer.SetImageLookup(lookup)

			result, err := b.mdRenderer.Render(page.RawContent)
			if err != nil {
				return nil, fmt.Errorf("rendering markdown for %s: %w", page.FilePath, err)
			}
			page.Content = htmltemplate.HTML(result.HTML)
			page.Headings = result.Headings
			page.HasCodeBlocks = result.HasCodeBlocks
			page.HasImages = result.HasImages

			if pageCache != nil {
				pageCache.Put(hash, &CacheEntry{
					ContentHash:   hash,
					HTML:          result.HTML,
					Headings:      result.Headings,
					HasCodeBlocks: result.HasCodeBlocks,
					HasImages:     result.HasImages,
				})
			}
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

	// Phase 5: RENDER
	var rendered []RenderedPage
	aliases := make(map[string]string)

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
					rd := coderootemplate.BuildRouteData(page, siteCtx, b.themeConfig)

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
			rd := coderootemplate.BuildRouteData(page, siteCtx, b.themeConfig)
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
			rd := coderootemplate.BuildRouteData(stub, siteCtx, b.themeConfig)
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
		rd := coderootemplate.BuildRouteData(taxStub, siteCtx, b.themeConfig)
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
			rd := coderootemplate.BuildRouteData(termStub, siteCtx, b.themeConfig)
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
				rd := coderootemplate.BuildRouteData(paginatedStub, siteCtx, b.themeConfig)
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

	// Phase 6: WRITE
	writer := &Writer{
		OutputDir:  outputDir,
		ProjectDir: b.projectDir,
		Clean:      config.BoolVal(b.config.Build.Clean, true),
	}
	if err := writer.Write(rendered, aliases); err != nil {
		return nil, fmt.Errorf("writing output: %w", err)
	}

	// Write bundle assets (images, PDFs, etc. co-located with content).
	if err := assetPipeline.WriteBundleAssets(allPages, outputDir); err != nil {
		return nil, fmt.Errorf("writing bundle assets: %w", err)
	}

	// Copy processed image variants from cache to output.
	if err := assetPipeline.WriteProcessedImages(outputDir); err != nil {
		return nil, fmt.Errorf("writing processed images: %w", err)
	}

	// Write bundled CSS/JS files to output.
	if err := assetPipeline.WriteBundledFiles(outputDir); err != nil {
		return nil, fmt.Errorf("writing bundled files: %w", err)
	}

	// Write embedded theme-level static assets (JS helpers like copy-code, spoiler, tabs, prefetch).
	if err := WriteEmbeddedAssets(b.embeddedFS, outputDir); err != nil {
		return nil, fmt.Errorf("writing embedded theme assets: %w", err)
	}

	// Write the embedded CSS bundle as an external file.
	if err := WriteEmbeddedCSS(outputDir, b.tmplEngine.CachedCSS()); err != nil {
		return nil, fmt.Errorf("writing embedded CSS bundle: %w", err)
	}

	// Plugin hook: BuildDone (parallel, after all files written).
	var pluginWarnings []engine.ValidationWarning
	buildDoneCtx := &plugin.BuildDoneContext{
		Config:      b.config,
		OutputDir:   outputDir,
		Pages:       allPages,
		Collections: collections,
		Site:        siteCtx,
	}
	buildDoneCtx.SetWarnings(&pluginWarnings)
	if err := b.pluginMgr.RunBuildDone(buildDoneCtx); err != nil {
		return nil, err
	}
	warnings = append(warnings, pluginWarnings...)

	return &engine.BuildResult{
		PageCount: len(rendered),
		Duration:  time.Since(start),
		Warnings:  warnings,
		OutputDir: outputDir,
	}, nil
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
			mgr.Register(announcements.New(configs[name]))
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
