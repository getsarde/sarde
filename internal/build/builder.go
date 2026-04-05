package build

import (
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/coderoo-dev/coderoo/internal/asset"
	"github.com/coderoo-dev/coderoo/internal/collection"
	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/internal/content"
	"github.com/coderoo-dev/coderoo/internal/content/markdown"
	"github.com/coderoo-dev/coderoo/internal/engine"
	"github.com/coderoo-dev/coderoo/internal/plugin"
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

	return &SiteBuilder{
		projectDir:  opts.ProjectDir,
		config:      opts.Config,
		themeConfig: opts.ThemeConfig,
		embeddedFS:  opts.EmbeddedFS,
		devMode:     opts.DevMode,
		scanner:     &content.Scanner{},
		mdRenderer:  markdown.NewRenderer(),
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
		contentDir = filepath.Join(b.projectDir, b.config.Content.Dir)
	}

	outputDir := filepath.Join(b.projectDir, "dist")
	if b.config.Build.Output != "" {
		outputDir = filepath.Join(b.projectDir, b.config.Build.Output)
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

	standalones, err := collection.BuildStandalonePages(files, contentDir, b.config.Content.SummaryLength)
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

	// Build taxonomies.
	taxonomies := taxonomy.BuildTaxonomies(allPages)

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

	// Render markdown for all pages (after asset enhancement so image
	// renderer can access processed resource data for <picture> generation).
	for _, page := range allPages {
		if page.RawContent == "" {
			continue
		}

		// Set per-page image lookup for bundle-relative images.
		lookup := markdown.ImageLookupForPage(page, assetPipeline.ImageProcessor(), outputDir)
		b.mdRenderer.SetImageLookup(lookup)

		html, headings, err := b.mdRenderer.Render(page.RawContent)
		if err != nil {
			return nil, fmt.Errorf("rendering markdown for %s: %w", page.FilePath, err)
		}
		page.Content = htmltemplate.HTML(html)
		page.Headings = headings
	}

	// Bundle global CSS/JS assets (processes head.custom_css/custom_js from config).
	if err := assetPipeline.BundleGlobalAssets(); err != nil {
		return nil, fmt.Errorf("bundling global assets: %w", err)
	}

	// Load template engine (needs SiteContext + asset pipeline + plugin funcs for funcMap closures).
	b.tmplEngine.SetSiteContext(siteCtx)
	b.tmplEngine.SetAssetPipeline(assetPipeline.Resolver(), assetPipeline.Manifest())
	b.tmplEngine.SetPluginFuncs(b.pluginMgr.TemplateFuncs())
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

	for _, page := range allPages {
		// Skip non-rendering pages.
		if page.Params != nil {
			if r, ok := page.Params["render"].(bool); ok && !r {
				continue
			}
		}

		rd := coderootemplate.BuildRouteData(page, siteCtx, b.themeConfig)

		// Plugin hook: BeforeRender (serial per page).
		if err := b.pluginMgr.RunBeforeRender(b.config, page, rd, siteCtx); err != nil {
			return nil, err
		}

		html, err := b.tmplEngine.Render(rd.Template, rd)
		if err != nil {
			return nil, fmt.Errorf("rendering %s (template %s): %w", page.FilePath, rd.Template, err)
		}

		rendered = append(rendered, RenderedPage{
			Page:    page,
			HTML:    html,
			OutPath: PageOutputPath(page.RelPermalink),
		})

		// Collect aliases.
		for _, alias := range page.Aliases {
			aliases[alias] = page.RelPermalink
		}
	}

	// Render 404 page.
	page404 := &engine.Page{Title: "Page Not Found", Kind: engine.KindPage}
	rd404 := &engine.RouteData{
		Template: "_default/404",
		Layout:   engine.LayoutDefault,
		Site:     siteCtx,
		Theme:    b.themeConfig,
		Page:     page404,
		Lang:     siteCtx.Language,
		Dir:      "ltr",
	}
	if siteCtx.Language == "" {
		rd404.Lang = "en"
	}
	html404, err := b.tmplEngine.Render(rd404.Template, rd404)
	if err == nil {
		rendered = append(rendered, RenderedPage{
			Page:    page404,
			HTML:    html404,
			OutPath: "404.html",
		})
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
		contentDir = filepath.Join(b.projectDir, b.config.Content.Dir)
	}

	files, err := b.scanner.DiscoverFiles(contentDir)
	if err != nil {
		return nil, fmt.Errorf("discovering content: %w", err)
	}

	collections, warnings, err := collection.BuildCollections(files, b.config, contentDir)
	if err != nil {
		return nil, fmt.Errorf("building collections: %w", err)
	}

	standalones, err := collection.BuildStandalonePages(files, contentDir, b.config.Content.SummaryLength)
	if err != nil {
		return nil, fmt.Errorf("building standalone pages: %w", err)
	}

	pageCount := 0
	for _, col := range collections {
		pageCount += len(col.Pages)
	}
	pageCount += len(standalones)

	return &ValidateResult{
		PageCount:   pageCount,
		Collections: len(collections),
		Warnings:    warnings,
		Duration:    time.Since(start),
	}, nil
}

// ValidateResult holds the outcome of a validation run (no rendering or writing).
type ValidateResult struct {
	PageCount   int
	Collections int
	Warnings    []engine.ValidationWarning
	Duration    time.Duration
}
