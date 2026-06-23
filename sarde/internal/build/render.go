package build

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"

	"golang.org/x/sync/errgroup"

	"github.com/frostybee/sarde/internal/asset"
	"github.com/frostybee/sarde/internal/consts"
	"github.com/frostybee/sarde/internal/content/markdown"
	"github.com/frostybee/sarde/internal/content/markdown/icons"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/shortcode"
	sardetemplate "github.com/frostybee/sarde/internal/template"
	"github.com/frostybee/sarde/internal/workers"
)

func enhancePageResources(pages []*engine.Page, pipeline *asset.Pipeline, parallel bool, workerCount int) error {
	if !workers.ShouldParallelize(parallel, len(pages), workerCount) {
		for _, page := range pages {
			if err := pipeline.EnhanceResources(page); err != nil {
				return fmt.Errorf("enhancing resources for %s: %w", page.FilePath, err)
			}
		}
		return nil
	}

	g := new(errgroup.Group)
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
				aliases[alias] = page.Permalink
			}
		}
		return rendered, aliases, nil
	}

	rendered := make([]RenderedPage, len(pages))
	g := new(errgroup.Group)
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
	assetPipeline  *asset.Pipeline
}

func (b *SiteBuilder) renderMarkdownPageSerial(
	page *engine.Page,
	deps markdownRenderDeps,
	siteCtx *engine.SiteContext,
) (links []engine.CollectedLink, scWarnings []engine.ValidationWarning, err error) {
	if page.RawContent == "" {
		return nil, nil, nil
	}

	processed, scWarns := deps.scProcessor.Process(page.RawContent, page, siteCtx, b.mdRenderer)
	hash := ContentHash(processed + deps.shortcodesHash + deps.resolutionKey + deps.iconRenderKey + deps.rendererKey)

	if deps.pageCache != nil {
		if entry := deps.pageCache.Get(hash); entry != nil {
			page.Content = htmltemplate.HTML(entry.HTML)
			page.Headings = entry.Headings
			page.HasCodeBlocks = entry.HasCodeBlocks
			page.HasImages = entry.HasImages
			return entry.Links, scWarns, nil
		}
	}

	lookup := markdown.ImageLookupForPage(page, deps.assetPipeline.ImageProcessor())
	b.mdRenderer.SetImageLookup(lookup)
	b.mdRenderer.SetLinkContext(page)

	result, err := b.mdRenderer.Render(processed)
	if err != nil {
		return nil, scWarns, fmt.Errorf("rendering markdown for %s: %w", page.FilePath, err)
	}

	page.Content = htmltemplate.HTML(result.HTML)
	page.Headings = result.Headings
	page.HasCodeBlocks = result.HasCodeBlocks
	page.HasImages = result.HasImages

	if deps.pageCache != nil {
		deps.pageCache.Put(hash, &CacheEntry{
			ContentHash:   hash,
			HTML:          result.HTML,
			Headings:      result.Headings,
			HasCodeBlocks: result.HasCodeBlocks,
			HasImages:     result.HasImages,
			Links:         result.Links,
		})
	}

	return result.Links, scWarns, nil
}

func minifyRendered(rendered []RenderedPage, parallel bool, workerCount int) error {
	if !workers.ShouldParallelize(parallel, len(rendered), workerCount) {
		mn := NewMinifier()
		for i := range rendered {
			rendered[i].HTML = mn.MinifyHTML(rendered[i].HTML)
		}
		return nil
	}

	g := new(errgroup.Group)
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
