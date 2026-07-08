package build

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"os"
	"path/filepath"

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
	"github.com/getsarde/sarde/internal/taxonomy"
	sardetemplate "github.com/getsarde/sarde/internal/template"
	"github.com/getsarde/sarde/internal/workers"
)

func enhancePageResources(pages []*engine.Page, pipeline *asset.Pipeline, parallel bool, workerCount int) error {
	return workers.ParallelFor(pages, parallel, workerCount, func(_ int, page *engine.Page) error {
		if err := pipeline.EnhanceResources(page); err != nil {
			return fmt.Errorf("enhancing resources for %s: %w", page.FilePath, err)
		}
		return nil
	})
}

func (b *SiteBuilder) renderPages(pages []*engine.Page, siteCtx *engine.SiteContext, parallel bool, workerCount int) ([]RenderedPage, map[string]string, error) {
	aliases := make(map[string]string)
	if len(pages) == 0 {
		return nil, aliases, nil
	}

	rendered := make([]RenderedPage, len(pages))
	err := workers.ParallelFor(pages, parallel, workerCount, func(i int, page *engine.Page) error {
		rp, err := b.renderPage(page, siteCtx)
		if err != nil {
			return err
		}
		rendered[i] = rp
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	for _, page := range pages {
		for _, alias := range page.Aliases {
			aliases[alias] = page.Permalink
		}
	}
	return rendered, aliases, nil
}

func (b *SiteBuilder) renderPage(page *engine.Page, siteCtx *engine.SiteContext) (RenderedPage, error) {
	rd := sardetemplate.BuildRouteData(page, siteCtx, b.themeConfig)
	rd.Styles = append(b.globalCSSURLs, rd.Styles...)
	if b.themeJSURL != "" {
		rd.Scripts = append([]string{b.themeJSURL}, rd.Scripts...)
	}
	if err := b.pluginMgr.RunBeforeRender(b.config, page, rd, siteCtx, b.urlResolver); err != nil {
		return RenderedPage{}, err
	}
	resolveRouteAssets(b.urlResolver, rd)

	var html []byte
	if redirect := tabbedCollectionRedirect(page); redirect != "" {
		if b.urlResolver != nil {
			redirect = b.urlResolver.URL(redirect, page.Lang, "")
		}
		html = []byte(buildRedirectHTML(redirect))
	} else {
		var err error
		html, err = b.tmplEngine.Render(rd.Template, rd)
		if err != nil {
			return RenderedPage{}, fmt.Errorf("rendering %s (template %s): %w", page.FilePath, rd.Template, err)
		}
	}

	// Sprite render mode: collect the <use> references this finished page emitted
	// (from both Markdown :icon[] and template {{ icon }}) and inject the hidden
	// <symbol> sprite before </body>. No-op (and zero overhead) in inline mode.
	if icons.SpriteMode() {
		if sprite := icons.SpriteForHTML(html); sprite != nil {
			html = injectBeforeBodyClose(html, sprite)
		}
	}

	return RenderedPage{
		Page:    page,
		HTML:    html,
		OutPath: PageOutputPath(b.urlResolver.OutputRelPath(page.RelPermalink, page.Lang, resolvePageVersion(page))),
	}, nil
}

// injectBeforeBodyClose splices extra into html immediately before the final
// </body>, mirroring the dev server's live-reload injection. The html is
// returned unchanged when no </body> is present (e.g. redirect stubs).
func injectBeforeBodyClose(html, extra []byte) []byte {
	idx := bytes.LastIndex(html, []byte("</body>"))
	if idx == -1 {
		return html
	}
	out := make([]byte, 0, len(html)+len(extra))
	out = append(out, html[:idx]...)
	out = append(out, extra...)
	out = append(out, html[idx:]...)
	return out
}

type markdownRenderDeps struct {
	scProcessor    *shortcode.Processor
	shortcodesHash string
	resolutionKey  string
	iconRenderKey  string
	rendererKey    string
	pageCache      *PageCache
	pageIndex      *content.PageIndex
	assetPipeline  *asset.Pipeline
}

func (b *SiteBuilder) renderMarkdownPageSerial(
	page *engine.Page,
	deps markdownRenderDeps,
	siteCtx *engine.SiteContext,
) (collectedLinks []engine.CollectedLink, pendingAnchors []links.PendingAnchorCheck, scWarnings []engine.ValidationWarning, err error) {
	if page.RawContent == "" {
		return nil, nil, nil, nil
	}

	processed, scWarns := deps.scProcessor.Process(page.RawContent, page, siteCtx, b.mdRenderer)
	hash := pageCacheKey(processed, deps.shortcodesHash, deps.resolutionKey, deps.iconRenderKey, deps.rendererKey, page.Lang)

	if deps.pageCache != nil {
		if entry := deps.pageCache.Get(hash); entry != nil {
			page.Content = htmltemplate.HTML(entry.HTML)
			page.Headings = entry.Headings
			page.HasCodeBlocks = entry.HasCodeBlocks
			page.HasImages = entry.HasImages
			replayed := replayPendingAnchors(entry.PendingAnchors, page, deps.pageIndex)
			return entry.Links, replayed, scWarns, nil
		}
	}

	lookup := markdown.ImageLookupForPage(page, deps.assetPipeline.ImageProcessor())
	b.mdRenderer.SetImageLookup(lookup)
	b.mdRenderer.SetLinkContext(page)

	result, err := b.mdRenderer.Render(processed)
	pageAnchors := b.mdRenderer.LinkRenderer().DrainPendingAnchors()
	if err != nil {
		return nil, nil, scWarns, fmt.Errorf("rendering markdown for %s: %w", page.FilePath, err)
	}

	page.Content = htmltemplate.HTML(result.HTML)
	page.Headings = result.Headings
	page.HasCodeBlocks = result.HasCodeBlocks
	page.HasImages = result.HasImages

	if deps.pageCache != nil {
		deps.pageCache.Put(hash, &CacheEntry{
			ContentHash:    hash,
			HTML:           result.HTML,
			Headings:       result.Headings,
			HasCodeBlocks:  result.HasCodeBlocks,
			HasImages:      result.HasImages,
			Links:          result.Links,
			PendingAnchors: toCachedAnchors(pageAnchors),
		})
	}

	return result.Links, pageAnchors, scWarns, nil
}

func minifyRendered(rendered []RenderedPage, parallel bool, workerCount int) error {
	return workers.ParallelFor(rendered, parallel, workerCount, func(i int, _ RenderedPage) error {
		mn := NewMinifier()
		rendered[i].HTML = mn.MinifyHTML(rendered[i].HTML)
		return nil
	})
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
		for n := 2; n <= total; n++ {
			permalink := sardetemplate.PaginationURL(base, n)
			if _, ok := dirty[permalink]; !ok {
				continue
			}
			stub := &engine.Page{
				PageIdentity: engine.PageIdentity{
					Title:        col.IndexPage.Title,
					Kind:         engine.KindSection,
					Permalink:    permalink,
					RelPermalink: permalink,
				},
				PageRelationships: engine.PageRelationships{
					Collection: col,
					Section:    col.IndexPage.Section,
				},
				PageI18n: engine.PageI18n{Lang: col.IndexPage.Lang},
				Params: map[string]any{
					consts.PaginationCurrentKey: n,
				},
			}
			resolvePermalinks(b.urlResolver, []*engine.Page{stub})
			rd := sardetemplate.BuildRouteData(stub, site, b.themeConfig)
			if err := b.pluginMgr.RunBeforeRender(b.config, stub, rd, site, b.urlResolver); err != nil {
				return err
			}
			resolveRouteAssets(b.urlResolver, rd)
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

// synthesizePaginationPages builds numbered pagination stubs (e.g.
// /blog/page/2/) for every paginated collection. It returns the stubs and
// the number of paginator pages created.
func (b *SiteBuilder) synthesizePaginationPages(s *buildState) ([]*engine.Page, int) {
	var syntheticPages []*engine.Page
	paginatorPages := 0

	for _, col := range s.collections {
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
	return syntheticPages, paginatorPages
}

// synthesizeTaxonomyPages builds taxonomy index, term, and paginated-term
// stubs for every renderable taxonomy in every language, cross-linking
// translations in multilingual builds. It returns the stubs (all of which are
// also taxonomy pages) and the number of paginator pages created.
func (b *SiteBuilder) synthesizeTaxonomyPages(s *buildState) ([]*engine.Page, int) {
	var taxonomyPages []*engine.Page
	paginatorPages := 0

	allTaxByLang := s.taxByLang
	if allTaxByLang == nil {
		allTaxByLang = map[string]map[string]*engine.Taxonomy{"": s.taxonomies}
	}

	var taxLangs []string
	if s.isMultiLang {
		taxLangs = b.config.I18n.LanguageCodes()
	} else {
		taxLangs = []string{""}
	}

	// Pass 1: build all index + term (page 1) stubs, keyed for cross-linking.
	stubsByLang := make(map[string]map[string]*engine.Page, len(taxLangs))
	for _, lang := range taxLangs {
		langTax := allTaxByLang[lang]
		if langTax == nil {
			continue
		}
		stubs := make(map[string]*engine.Page)
		for _, taxName := range sortedKeys(langTax) {
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

	// Pass 2: cross-link, then collect into taxonomyPages.
	for _, lang := range taxLangs {
		langTax := allTaxByLang[lang]
		if langTax == nil {
			continue
		}
		stubs := stubsByLang[lang]
		for _, taxName := range sortedKeys(langTax) {
			tax := langTax[taxName]
			cfg := b.config.Taxonomies[taxName]
			if !cfg.ShouldRender() {
				continue
			}

			taxStub := stubs[taxName]
			if s.isMultiLang {
				taxStub.Translations = crossLinkStubs(stubsByLang, taxName, lang)
				taxStub.AllTranslations = taxStub.Translations
			}
			taxonomyPages = append(taxonomyPages, taxStub)

			paginateBy := cfg.PaginateBy
			if paginateBy <= 0 {
				paginateBy = consts.DefaultPaginateBy
			}
			for _, term := range tax.Terms {
				key := taxName + "/" + term.Slug
				termStub := stubs[key]
				if s.isMultiLang {
					termStub.Translations = crossLinkStubs(stubsByLang, key, lang)
					termStub.AllTranslations = termStub.Translations
				}
				taxonomyPages = append(taxonomyPages, termStub)

				totalTermPages := (len(term.Pages) + paginateBy - 1) / paginateBy
				if totalTermPages < 1 {
					totalTermPages = 1
				}
				for n := 2; n <= totalTermPages; n++ {
					permalink := sardetemplate.PaginationURL(term.Permalink, n)
					paginatedStub := buildTermPaginatedStub(tax, term, permalink, n, lang)
					taxonomyPages = append(taxonomyPages, paginatedStub)
					paginatorPages++
				}
			}
		}
	}
	return taxonomyPages, paginatorPages
}

func (b *SiteBuilder) phaseRender(s *buildState) error {
	var renderablePages []*engine.Page
	for _, page := range s.allPages {
		if page.Params != nil {
			if r, ok := page.Params["render"].(bool); ok && !r {
				continue
			}
		}
		renderablePages = append(renderablePages, page)
	}

	rendered, aliases, err := b.renderPages(renderablePages, s.siteCtx, s.parallel, s.workerCount)
	if err != nil {
		return err
	}
	s.recordTiming("Rendering templates")

	// Synthesize numbered pagination pages, then taxonomy index/term pages.
	syntheticPages, paginatorPages := b.synthesizePaginationPages(s)
	taxonomyPages, taxPaginatorPages := b.synthesizeTaxonomyPages(s)
	syntheticPages = append(syntheticPages, taxonomyPages...)
	paginatorPages += taxPaginatorPages

	// Resolve Permalinks on synthetic pages (pagination + taxonomy stubs).
	resolvePermalinks(b.urlResolver, syntheticPages)

	syntheticRendered, syntheticAliases, err := b.renderPages(syntheticPages, s.siteCtx, s.parallel, s.workerCount)
	if err != nil {
		return err
	}
	rendered = append(rendered, syntheticRendered...)
	for alias, target := range syntheticAliases {
		aliases[alias] = target
	}
	s.recordTiming("Rendering synthetic pages")

	// Include taxonomy pages in allPages for sitemap and search index.
	s.allPages = append(s.allPages, taxonomyPages...)
	s.siteCtx.Pages = s.allPages

	// Render 404 page(s).
	render404 := func(lang, dir, outPath string) {
		page404 := &engine.Page{
			PageIdentity: engine.PageIdentity{Title: "Page Not Found", Kind: engine.KindPage},
			PageI18n:     engine.PageI18n{Lang: lang},
		}
		templateName := consts.DirDefault + "/404"
		layout := engine.LayoutDefault

		candidates := []string{
			filepath.Join(s.contentDir, "404."+lang+".md"),
			filepath.Join(s.contentDir, "404.md"),
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
			if fm != nil && fm.Layout != "" {
				layout = engine.ResolveLayout(fm.Layout)
			}
			if body != "" {
				b.mdRenderer.SetLinkContext(page404)
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
			Layout:   layout,
			Site:     s.siteCtx,
			Theme:    b.themeConfig,
			Page:     page404,
			RouteI18n: engine.RouteI18n{
				Lang: lang,
				Dir:  dir,
			},
		}
		resolveRouteAssets(b.urlResolver, rd404)
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

	if s.isMultiLang {
		for _, code := range b.config.I18n.LanguageCodes() {
			lc := b.config.I18n.Languages[code]
			dir := lc.Dir
			if dir == "" {
				dir = "ltr"
			}
			if code == s.defaultLang {
				render404(code, dir, consts.Template404)
			} else {
				render404(code, dir, code+"/"+consts.Template404)
			}
		}
	} else {
		lang := s.siteCtx.Language
		if lang == "" {
			lang = "en"
		}
		render404(lang, "ltr", consts.Template404)
	}
	s.recordTiming("Rendering 404 pages")

	rendered = appendVersionedLatestPages(rendered, s.collections, b.urlResolver)

	if !b.devMode && config.BoolVal(b.config.Build.Minify, true) {
		if err := minifyRendered(rendered, s.parallel, s.workerCount); err != nil {
			return err
		}
	}
	s.recordTiming("Minifying HTML")

	s.rendered = rendered
	s.aliases = aliases
	s.paginatorPages = paginatorPages
	s.syntheticPages = syntheticPages
	s.taxonomyPages = taxonomyPages
	return nil
}
