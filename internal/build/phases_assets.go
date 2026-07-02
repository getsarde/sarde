package build

import (
	"context"
	"fmt"
	htmltemplate "html/template"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/getsarde/sarde/internal/asset"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/content/markdown"
	"github.com/getsarde/sarde/internal/content/markdown/icons"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/links"
	"github.com/getsarde/sarde/internal/shortcode"
	"github.com/getsarde/sarde/internal/syntax"
	sardetemplate "github.com/getsarde/sarde/internal/template"
	"github.com/getsarde/sarde/internal/theme"
	"github.com/getsarde/sarde/internal/workers"
)

func (b *SiteBuilder) phaseAssets(s *buildState) error {
	b.checkPageSyntax(s)
	if err := b.setupAssetPipelineAndShortcodes(s); err != nil {
		return err
	}
	if err := b.renderAllMarkdown(s); err != nil {
		return err
	}
	if done, err := b.validateLinks(s); done || err != nil {
		return err
	}
	if err := b.bundleGlobalAssets(s); err != nil {
		return err
	}
	return b.wireTemplateEngine(s)
}

// checkPageSyntax lints fenced code blocks in all raw page sources when the
// builder was constructed with CheckSyntax enabled.
func (b *SiteBuilder) checkPageSyntax(s *buildState) {
	if !b.checkSyntax {
		return
	}
	checked := make(map[string]bool)
	for _, p := range s.allPages {
		if p.RawContent == "" || checked[p.FilePath] {
			continue
		}
		checked[p.FilePath] = true
		diags := syntax.Check(p.FilePath, []byte(p.RawContent), p.FrontmatterLines)
		for _, d := range diags {
			msg := fmt.Sprintf("line %d: %s", d.Line, d.Message)
			devlog.Warn("syntax", "%s:%d %s", d.File, d.Line, d.Message)
			s.warnings = append(s.warnings, engine.ValidationWarning{
				File:    d.File,
				Field:   "syntax",
				Message: msg,
				Level:   d.Level,
			})
		}
	}
}

// setupAssetPipelineAndShortcodes constructs the asset pipeline, enhances
// page resources, builds the page index, and loads the shortcode registry
// (three-layer overlay: embedded, theme, user). Results land in s
// (assetPipeline, pageIndex, scProcessor, shortcodesHash, iconRenderKey,
// pageCache).
func (b *SiteBuilder) setupAssetPipelineAndShortcodes(s *buildState) error {
	// Asset pipeline: image processing, resource enhancement, CSS/JS bundling.
	s.assetPipeline = asset.NewPipeline(asset.PipelineOptions{
		ProjectDir: b.projectDir,
		OutputDir:  s.outputDir,
		Config:     b.config,
		ThemeName:  b.config.Theme.Name,
		EmbeddedFS: b.embeddedFS,
		DevMode:    b.devMode,
	})
	if err := enhancePageResources(s.allPages, s.assetPipeline, s.parallel, s.workerCount); err != nil {
		return err
	}
	s.recordTiming("Asset preparation")

	// Build page index for link validation and O(1) ref/relref lookups.
	s.pageIndex = content.BuildPageIndex(s.allPages)
	s.pageIndex.AddAssets(filepath.Join(b.projectDir, consts.DirStatic))

	// Build shortcode registry (three-layer overlay: embedded → theme → user).
	scFuncMap := sardetemplate.BuildShortcodeFuncMap(sardetemplate.ShortcodeFuncMapConfig{
		Site: &s.siteCtx,
		Resolver: &engine.ThemeResolver{
			ProjectDir: b.projectDir,
			ThemeName:  b.config.Theme.Name,
			EmbeddedFS: b.embeddedFS,
		},
		AssetResolver:  s.assetPipeline.Resolver(),
		AssetManifest:  s.assetPipeline.Manifest(),
		ImageProcessor: s.assetPipeline.ImageProcessor(),
		PageIndex:      &s.pageIndex,
	})
	scRegistry, err := shortcode.NewRegistry(b.embeddedFS, scFuncMap)
	if err != nil {
		return fmt.Errorf("loading shortcode registry: %w", err)
	}
	if b.config.Theme.Name != "" {
		themeScDir := filepath.Join(b.projectDir, consts.DirThemes, b.config.Theme.Name, consts.DirLayouts, consts.DirShortcodes)
		if err := scRegistry.LoadOverridesFromDir(themeScDir); err != nil {
			return fmt.Errorf("loading theme shortcode overrides: %w", err)
		}
	}
	userScDir := filepath.Join(b.projectDir, consts.DirLayouts, consts.DirShortcodes)
	if err := scRegistry.LoadOverridesFromDir(userScDir); err != nil {
		return fmt.Errorf("loading user shortcode overrides: %w", err)
	}
	s.scProcessor = shortcode.NewProcessor(scRegistry)
	s.shortcodesHash = scRegistry.TemplateHash()

	s.iconRenderKey = "icon-inline"
	if icons.SpriteMode() {
		s.iconRenderKey = "icon-sprite"
	}

	if config.BoolVal(b.config.Build.Cache, true) && !b.checkOnly {
		s.pageCache = NewPageCache(b.projectDir)
	}
	return nil
}

// renderAllMarkdown renders every page's markdown to HTML, in parallel when
// the page count warrants it, collecting link-validation data and pending
// anchor checks into s.
func (b *SiteBuilder) renderAllMarkdown(s *buildState) error {
	assetPipeline := s.assetPipeline
	pageIndex := s.pageIndex
	scProcessor := s.scProcessor
	shortcodesHash := s.shortcodesHash
	iconRenderKey := s.iconRenderKey
	pageCache := s.pageCache

	b.linkGraph = links.NewLinkGraph()

	var validationMu sync.Mutex
	validationData := make(map[string]engine.ValidationEntry)
	var pendingAnchors []links.PendingAnchorCheck

	// Initialize Kazari and the markdown renderer on first build only.
	if b.kazariEngine == nil {
		ke, err := markdown.BuildKazariEngine(context.Background(), &b.config.Markdown.Codeblocks, b.projectDir)
		if err != nil {
			return fmt.Errorf("initializing code block engine: %w", err)
		}
		b.kazariEngine = ke
	}
	if b.mdRenderer == nil {
		b.mdRenderer = markdown.NewRendererFromConfig(markdown.RendererConfig{
			BlockedHrefSchemes: b.config.Security.BlockedHrefSchemes,
			HeadingLinks:       config.BoolVal(b.config.Site.HeadingLinks, true),
			KazariEngine:       b.kazariEngine,
		})
	}
	if b.rendererKey == "" {
		b.rendererKey = b.mdRenderer.Fingerprint()
	}

	// Render markdown, parallel when page count warrants it.
	markdownPages := countMarkdownPages(s.allPages)
	if workers.ShouldParallelize(s.parallel, markdownPages, s.workerCount) {
		poolSize := s.workerCount
		if b.rendererPool == nil {
			b.rendererPool = make(chan *markdown.Renderer, poolSize)
			for i := 0; i < poolSize; i++ {
				b.rendererPool <- markdown.NewRendererFromConfig(markdown.RendererConfig{
					BlockedHrefSchemes: b.config.Security.BlockedHrefSchemes,
					HeadingLinks:       config.BoolVal(b.config.Site.HeadingLinks, true),
					KazariEngine:       b.kazariEngine,
				})
			}
		}

		g := new(errgroup.Group)
		g.SetLimit(cap(b.rendererPool))
		for _, page := range s.allPages {
			if page.RawContent == "" {
				continue
			}
			g.Go(func() error {
				renderer := <-b.rendererPool
				lr := renderer.LinkRenderer()
				lr.PageIndex = pageIndex
				lr.URLResolver = b.urlResolver
				lr.LinkGraph = b.linkGraph
				lr.Collections = s.collections
				lr.SiteRootEscapePrefix = b.config.LinkValidation.SiteRootEscapePrefix

				processed, scWarns := scProcessor.Process(page.RawContent, page, s.siteCtx, renderer)
				if len(scWarns) > 0 {
					validationMu.Lock()
					s.warnings = append(s.warnings, scWarns...)
					validationMu.Unlock()
				}

				hash := ContentHash(processed + shortcodesHash + b.resolutionKey + iconRenderKey + b.rendererKey + "\x00lang=" + page.Lang)
				if pageCache != nil {
					if entry := pageCache.Get(hash); entry != nil {
						page.Content = htmltemplate.HTML(entry.HTML)
						page.Headings = entry.Headings
						page.HasCodeBlocks = entry.HasCodeBlocks
						page.HasImages = entry.HasImages
						if len(entry.Links) > 0 {
							validationMu.Lock()
							validationData[page.Permalink] = engine.ValidationEntry{Links: entry.Links, FilePath: page.FilePath, Lang: page.Lang}
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
				pagePendingAnchors := renderer.LinkRenderer().DrainPendingAnchors()
				b.rendererPool <- renderer
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
						validationData[page.Permalink] = engine.ValidationEntry{Links: result.Links, FilePath: page.FilePath, Lang: page.Lang}
					}
					pendingAnchors = append(pendingAnchors, pagePendingAnchors...)
					validationMu.Unlock()
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
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return err
		}
	} else {
		lr := b.mdRenderer.LinkRenderer()
		lr.PageIndex = pageIndex
		lr.URLResolver = b.urlResolver
		lr.LinkGraph = b.linkGraph
		lr.Collections = s.collections
		lr.SiteRootEscapePrefix = b.config.LinkValidation.SiteRootEscapePrefix

		deps := markdownRenderDeps{
			scProcessor:    scProcessor,
			shortcodesHash: shortcodesHash,
			resolutionKey:  b.resolutionKey,
			iconRenderKey:  iconRenderKey,
			rendererKey:    b.rendererKey,
			pageCache:      pageCache,
			assetPipeline:  assetPipeline,
		}
		for _, page := range s.allPages {
			collectedLinks, scWarns, err := b.renderMarkdownPageSerial(page, deps, s.siteCtx)
			if err != nil {
				return err
			}
			s.warnings = append(s.warnings, scWarns...)
			if len(collectedLinks) > 0 {
				validationData[page.Permalink] = engine.ValidationEntry{Links: collectedLinks, FilePath: page.FilePath, Lang: page.Lang}
			}
			pendingAnchors = append(pendingAnchors, lr.DrainPendingAnchors()...)
		}
	}
	s.recordTiming("Rendering markdown")

	s.pendingAnchors = pendingAnchors
	s.validationData = validationData
	return nil
}

// validateLinks validates collected anchors, computes link coverage, runs the
// optional external link check, and generates the structured link report. It
// returns true when the build should stop early (check-only mode).
func (b *SiteBuilder) validateLinks(s *buildState) (bool, error) {
	populatePageIndexHeadings(s.pageIndex, s.allPages)
	emitCollisionWarnings(s.pageIndex.Collisions())

	links.ValidateAnchors(b.linkGraph, s.pendingAnchors, s.pageIndex)

	var langCodes []string
	if s.isMultiLang {
		langCodes = b.config.I18n.LanguageCodes()
	}
	expectedLanes := links.EnumerateLanes(s.collections, langCodes)
	b.lastCoverage = links.ComputeCoverage(b.linkGraph, s.allPages, expectedLanes)

	// Generate structured link validation report.
	if config.BoolVal(b.config.LinkValidation.Enabled, true) {
		lvc := b.config.LinkValidation

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
				s.warnings = append(s.warnings, engine.ValidationWarning{
					File:    f.Ref.FromFile,
					Message: fmt.Sprintf("%s: %s", f.Type.Label(), f.Ref.RawDest),
					Level:   "warn",
				})
			}
		}
		if reportResult.HasErrors {
			return false, fmt.Errorf("build failed: link validation errors found")
		}
	}

	// Check-only: return after validation without rendering or writing.
	if b.checkOnly {
		s.recordTiming("Link validation")
		s.checkResult = &engine.BuildResult{
			PageCount: len(s.allPages),
			Warnings:  s.warnings,
		}
		return true, nil
	}
	return false, nil
}

// bundleGlobalAssets bundles global CSS/JS, theme JS, Kazari JS, the favicon,
// and the externalized theme token CSS.
func (b *SiteBuilder) bundleGlobalAssets(s *buildState) error {
	if err := s.assetPipeline.BundleGlobalAssets(); err != nil {
		return fmt.Errorf("bundling global assets: %w", err)
	}
	b.globalCSSURLs = s.assetPipeline.GlobalCSSURLs()

	// Bundle theme JS: prefer external theme over embedded.
	var jsContent []byte
	var jsFilename string
	var err error
	if b.config.Theme.Name != "" {
		jsContent, jsFilename, err = BundleThemeJS(b.projectDir, b.config.Theme.Name, b.devMode)
		if err != nil {
			return fmt.Errorf("bundling theme JS: %w", err)
		}
	}
	if jsFilename == "" {
		jsContent, jsFilename, err = BundleEmbeddedJS(b.embeddedFS, b.devMode)
		if err != nil {
			return fmt.Errorf("bundling embedded theme JS: %w", err)
		}
	}
	if jsFilename != "" {
		b.themeJSURL = "/assets/js/" + jsFilename
		b.themeJSContent = jsContent
		b.themeJSFilename = jsFilename
	}

	// Bundle Kazari interaction JS.
	{
		kazariJS := []byte(b.kazariEngine.JS())
		hash := asset.Fingerprint(kazariJS)
		if b.devMode {
			b.kazariJSFilename = "kazari.js"
		} else {
			b.kazariJSFilename = asset.FingerprintedName("kazari.js", hash)
		}
		b.kazariJSURL = "/assets/js/" + b.kazariJSFilename
		b.kazariJSContent = kazariJS
		s.siteCtx.KazariScriptURL = b.urlResolver.URL(b.kazariJSURL, "", "")
	}

	// Auto-detect or use configured favicon.
	if fav := b.config.Site.Favicon; fav != "" {
		s.siteCtx.Favicon = b.urlResolver.URL(fav, "", "")
		s.siteCtx.FaviconType = faviconMIME(fav)
	} else if detected := detectFavicon(b.projectDir); detected != "" {
		s.siteCtx.Favicon = b.urlResolver.URL(detected, "", "")
		s.siteCtx.FaviconType = faviconMIME(detected)
	}

	// Externalize theme token CSS as a fingerprinted file.
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
	return nil
}

// wireTemplateEngine hands the assembled build context to the template
// engine and loads all templates.
func (b *SiteBuilder) wireTemplateEngine(s *buildState) error {
	b.tmplEngine.SetTokenCSSURL(b.tokenCSSURL)
	b.tmplEngine.SetSiteContext(s.siteCtx)
	b.tmplEngine.SetURLResolver(b.urlResolver)
	b.tmplEngine.SetAssetPipeline(s.assetPipeline.Resolver(), s.assetPipeline.Manifest())
	b.tmplEngine.SetImageProcessor(s.assetPipeline.ImageProcessor())
	b.tmplEngine.SetPluginFuncs(b.pluginMgr.TemplateFuncs())
	b.tmplEngine.SetPageIndex(s.pageIndex)
	if s.stringTable != nil {
		b.tmplEngine.SetI18nStrings(s.stringTable)
	}
	b.tmplEngine.SetCodeBlockCSS(b.kazariEngine.CSS())

	resolver := &engine.ThemeResolver{
		ProjectDir: b.projectDir,
		ThemeName:  b.config.Theme.Name,
		EmbeddedFS: b.embeddedFS,
	}
	if err := b.tmplEngine.Load(resolver, b.devMode); err != nil {
		return fmt.Errorf("loading templates: %w", err)
	}
	s.recordTiming("Template setup")
	return nil
}
