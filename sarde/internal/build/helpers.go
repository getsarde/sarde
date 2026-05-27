package build

import (
	"fmt"
	htmltemplate "html/template"
	"sort"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/frostybee/sarde/internal/asset"
	"github.com/frostybee/sarde/internal/consts"
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/content/markdown"
	"github.com/frostybee/sarde/internal/devlog"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/shortcode"
	sardetemplate "github.com/frostybee/sarde/internal/template"
	"github.com/frostybee/sarde/internal/workers"
)

// ---------------------------------------------------------------------------
// Page rendering
// ---------------------------------------------------------------------------

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
				aliases[alias] = page.RelPermalink
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

// ---------------------------------------------------------------------------
// Markdown rendering
// ---------------------------------------------------------------------------

type markdownRenderDeps struct {
	scProcessor    *shortcode.Processor
	shortcodesHash string
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
	hash := ContentHash(processed + deps.shortcodesHash)

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

// ---------------------------------------------------------------------------
// Minification
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Taxonomy stub constructors
// ---------------------------------------------------------------------------

func buildTaxonomyIndexStub(tax *engine.Taxonomy, termEntries []*engine.TermEntry) *engine.Page {
	return &engine.Page{
		Title:        tax.Name,
		Kind:         engine.KindTaxonomy,
		Permalink:    tax.Permalink,
		RelPermalink: tax.Permalink,
		Params: map[string]any{
			consts.TaxonomyKey:    tax,
			consts.TermEntriesKey: termEntries,
		},
	}
}

func buildTermStub(tax *engine.Taxonomy, term *engine.TaxonomyTerm) *engine.Page {
	return &engine.Page{
		Title:        term.Label,
		Kind:         engine.KindTerm,
		Permalink:    term.Permalink,
		RelPermalink: term.Permalink,
		Params: map[string]any{
			consts.TaxonomyKey:     tax,
			consts.TaxonomyTermKey: term,
		},
	}
}

func buildTermPaginatedStub(tax *engine.Taxonomy, term *engine.TaxonomyTerm, permalink string, n int) *engine.Page {
	return &engine.Page{
		Title:        term.Label,
		Kind:         engine.KindTerm,
		Permalink:    permalink,
		RelPermalink: permalink,
		Params: map[string]any{
			consts.TaxonomyKey:          tax,
			consts.TaxonomyTermKey:      term,
			consts.PaginationCurrentKey: n,
		},
	}
}

// ---------------------------------------------------------------------------
// Page index helpers
// ---------------------------------------------------------------------------

func populatePageIndexHeadings(idx *content.PageIndex, pages []*engine.Page) {
	for _, page := range pages {
		setPageIndexHeadings(idx, page)
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

func updateValidationEntry(data map[string]engine.ValidationEntry, page *engine.Page, links []engine.CollectedLink) {
	if len(links) == 0 {
		delete(data, page.RelPermalink)
		return
	}
	data[page.RelPermalink] = engine.ValidationEntry{Links: links, FilePath: page.FilePath}
}

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

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
		taxPart, termPart, ok := strings.Cut(w, ": term ")
		if !ok {
			devlog.Warn("taxonomy", "%s", w)
			continue
		}
		taxName := strings.TrimPrefix(taxPart, "taxonomy ")
		taxName = strings.Trim(taxName, "\"")

		term := termPart
		if idx := strings.Index(term, "\" "); idx >= 0 {
			term = term[1:idx]
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
