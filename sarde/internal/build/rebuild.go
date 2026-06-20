package build

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/frostybee/sarde/internal/collection"
	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/consts"
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/devlog"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/i18n"
	"github.com/frostybee/sarde/internal/taxonomy"
	sardetemplate "github.com/frostybee/sarde/internal/template"
	"github.com/frostybee/sarde/internal/workers"
)

// ContentRebuild performs an incremental rebuild for content-only changes.
// It re-parses only the changed files, patches them into the existing collections,
// and renders/writes only the dirty pages. Falls back to full Build() on edge cases.
func (b *SiteBuilder) ContentRebuild(changedPaths []string) (*engine.BuildResult, error) {
	start := time.Now()

	if !b.built || b.lastCollections == nil || b.lastAssetPipeline == nil {
		return b.Build()
	}

	contentDir := b.resolveContentDir()

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
	dirtyCollections := make(map[string]*engine.Collection)

	for _, path := range changedPaths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(b.projectDir, path)
		}
		path = filepath.Clean(path)

		base := filepath.Base(path)
		if base == "_index.md" || base == "404.md" || (strings.HasPrefix(base, "404.") && filepath.Ext(base) == ".md") {
			devlog.Warn("build", "ContentRebuild: structural content changed (%s), falling back to full rebuild", base)
			return b.Build()
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			devlog.Warn("build", "ContentRebuild: file deleted, falling back to full rebuild")
			return b.Build()
		}

		cf, err := b.scanner.ClassifyFile(contentDir, path)
		if err != nil {
			devlog.Warn("build", "ContentRebuild: ClassifyFile error: %v, falling back", err)
			return b.Build()
		}

		old := oldByPath[path]
		if old == nil {
			devlog.Warn("build", "ContentRebuild: new file detected, falling back to full rebuild")
			return b.Build()
		}
		if len(old.Resources) > 0 || cf.IsBundle {
			devlog.Warn("build", "ContentRebuild: bundle content changed, falling back to full rebuild")
			return b.Build()
		}

		// Layer 1: content digest gate — skip if raw bytes unchanged.
		if old.ContentDigest != "" {
			rawBytes, readErr := os.ReadFile(path)
			if readErr == nil {
				h := sha256.Sum256(rawBytes)
				if old.ContentDigest == fmt.Sprintf("%x", h[:8]) {
					devlog.Log("build", "ContentRebuild: content unchanged for %s, skipping", filepath.Base(path))
					continue
				}
			}
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
			b.config.Taxonomies,
		)
		if err != nil {
			devlog.Warn("build", "ContentRebuild: parse error for %s: %v, falling back", path, err)
			return b.Build()
		}
		warnings = append(warnings, pageWarnings...)

		// Layer 2: frontmatter digest gate — if frontmatter unchanged, this is
		// a body-only change. Skip all eligibility checks and go straight to
		// the incremental path.
		if old.FrontmatterDigest != "" && old.FrontmatterDigest == newPage.FrontmatterDigest {
			// Body-only change — always eligible for incremental rebuild.
		} else {
			// Frontmatter changed — classify the change.
			includeDrafts := config.BoolVal(b.config.Build.Drafts, false)
			includeFuture := config.BoolVal(b.config.Build.Future, false)
			includeExpired := config.BoolVal(b.config.Build.Expired, false)
			now := time.Now()
			wasExcluded := content.ShouldExclude(old.Draft, old.PublishDate, old.ExpiryDate, includeDrafts, includeFuture, includeExpired, now)
			isExcluded := content.ShouldExclude(newPage.Draft, newPage.PublishDate, newPage.ExpiryDate, includeDrafts, includeFuture, includeExpired, now)
			if wasExcluded != isExcluded {
				devlog.Warn("build", "ContentRebuild: draft/publish status changed, falling back to full rebuild")
				return b.Build()
			}

			switch classifyChange(old, newPage, cf) {
			case changeFullRebuild:
				devlog.Warn("build", "ContentRebuild: site-structural change detected, falling back to full rebuild")
				return b.Build()
			case changeCollectionScoped:
				if old.Collection != nil {
					devlog.Log("build", "ContentRebuild: sort/nav change in %s, will rebuild collection", old.Collection.Name)
					dirtyCollections[old.Collection.Name] = old.Collection
				}
			case changeIncremental:
				// pass through — incremental path handles it
			}
		}

		parsed = append(parsed, parsedEntry{
			filePath: path,
			newPage:  newPage,
			old:      old,
		})
	}

	if len(parsed) == 0 && len(dirtyCollections) == 0 {
		return &engine.BuildResult{PageCount: 0, Duration: time.Since(start)}, nil
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

		// Normalize version for root-level pages in versioned collections.
		// BuildSinglePage leaves Version="" for root pages (no vN/ directory);
		// the full build normalizes them in BuildCollectionsWithOptions, but the
		// incremental path skips that step. preserveStablePageState below copies
		// old.Version, but we normalize first as a safety net.
		if e.newPage.Version == "" {
			if col := e.old.Collection; col != nil {
				if vc := col.Config.Versioning; vc != nil && vc.Enabled && vc.LastVersion != "" {
					e.newPage.Version = vc.LastVersion
					if parts := strings.SplitN(e.newPage.LangRelPath, "/", 2); len(parts) == 2 {
						e.newPage.VersionRelPath = parts[1]
					}
				}
			}
		}

		preserveStablePageState(e.newPage, e.old)

		if b.urlResolver != nil {
			e.newPage.Permalink = b.urlResolver.URL(
				e.newPage.RelPermalink, e.newPage.Lang, resolvePageVersion(e.newPage))
		}

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

	// Collection-scoped rebuild: re-sort and rebuild navigation for any
	// collection whose sort or sidebar fields changed.
	for colName, col := range dirtyCollections {
		rebuildCollectionNav(col)
		for _, p := range col.Pages {
			dirtyPermalinks[p.RelPermalink] = struct{}{}
		}
		addCollectionListDirty(col, dirtyPermalinks)
		devlog.Log("build", "ContentRebuild: rebuilt navigation for collection %q (%d pages)", colName, len(col.Pages))
	}

	// Dirty-mark old taxonomy terms for pages whose tags/categories changed,
	// so removed-tag term pages are re-rendered.
	if b.lastSiteCtx != nil && b.lastSiteCtx.Taxonomies != nil {
		for _, e := range parsed {
			addRemovedTermsDirty(e.old, e.newPage, b.lastSiteCtx.Taxonomies, b.config.Taxonomies, dirtyPermalinks)
		}
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
		weights := make(map[string]int)
		for code, lc := range b.config.I18n.Languages {
			weights[code] = lc.Weight
		}

		i18n.LinkTranslations(patchedAllPages, weights)

		collFallback := make(map[string]string)
		for colName, colCfg := range b.config.Collections {
			if colCfg != nil && colCfg.I18nFallback != "" {
				collFallback[colName] = colCfg.I18nFallback
			}
		}
		fbOpts := i18n.FallbackOptions{
			SiteFallback:       b.config.I18n.Fallback,
			CollectionFallback: collFallback,
		}

		fallbacks := i18n.GenerateFallbacks(patchedAllPages, langCodes, defaultLang, fbOpts)
		patchedAllPages = append(patchedAllPages, fallbacks...)

		if b.urlResolver != nil {
			resolvePermalinks(b.urlResolver, fallbacks)
		}

		for _, fb := range fallbacks {
			if fb.FilePath != "" {
				if _, ok := changedByPath[fb.FilePath]; ok {
					dirtyPermalinks[fb.Permalink] = struct{}{}
				}
			}
		}

		i18n.LinkAllTranslations(patchedAllPages, weights)
	}

	var newTaxonomies map[string]*engine.Taxonomy
	var newTaxByLang map[string]map[string]*engine.Taxonomy

	if isMultiLang {
		defaultLang := b.config.I18n.GetDefaultLanguage()
		langCodes := b.config.I18n.LanguageCodes()
		newTaxByLang = make(map[string]map[string]*engine.Taxonomy, len(langCodes))
		for _, code := range langCodes {
			langTax := taxonomy.BuildTaxonomies(patchedAllPages, b.config.Taxonomies, code)
			if _, err := taxonomy.EnrichTaxonomies(langTax, b.config.Taxonomies, b.projectDir, code); err != nil {
				return b.Build()
			}
			if b.urlResolver != nil {
				for _, tax := range langTax {
					tax.Permalink = b.urlResolver.URL(tax.Permalink, code, "")
					for _, term := range tax.Terms {
						term.Permalink = b.urlResolver.URL(term.Permalink, code, "")
					}
				}
			}
			newTaxByLang[code] = langTax
		}
		newTaxonomies = newTaxByLang[defaultLang]
	} else {
		newTaxonomies = taxonomy.BuildTaxonomies(patchedAllPages, b.config.Taxonomies, "")
		if _, err := taxonomy.EnrichTaxonomies(newTaxonomies, b.config.Taxonomies, b.projectDir, ""); err != nil {
			return b.Build()
		}
	}

	collection.LinkVersions(patchedAllPages)

	newPageIndex := content.BuildPageIndex(patchedAllPages)
	newPageIndex.AddAssets(filepath.Join(b.projectDir, consts.DirStatic))
	populatePageIndexHeadings(newPageIndex, patchedAllPages)
	emitCollisionWarnings(newPageIndex.Collisions())

	if newTaxByLang != nil {
		for _, langTax := range newTaxByLang {
			for _, e := range parsed {
				addTaxonomyDirtyForPage(e.filePath, langTax, b.config.Taxonomies, dirtyPermalinks)
			}
		}
	} else {
		for _, e := range parsed {
			addTaxonomyDirtyForPage(e.filePath, newTaxonomies, b.config.Taxonomies, dirtyPermalinks)
		}
	}

	b.lastSiteCtx.Collections = b.lastCollections
	b.lastSiteCtx.Taxonomies = newTaxonomies
	b.lastSiteCtx.TaxonomiesByLang = newTaxByLang
	b.lastSiteCtx.Pages = patchedAllPages
	b.lastSiteCtx.BuildTime = time.Now()
	b.tmplEngine.SetSiteContext(b.lastSiteCtx)
	b.tmplEngine.SetPageIndex(newPageIndex)

	// Inject Kazari CSS into the template engine (reuse existing engine on incremental rebuilds).
	if b.kazariEngine != nil {
		b.tmplEngine.SetCodeBlockCSS(b.kazariEngine.CSS())
	}

	mergedValidation := make(map[string]engine.ValidationEntry, len(b.lastValidationData))
	for k, v := range b.lastValidationData {
		mergedValidation[k] = v
	}

	// Configure link resolver for incremental re-render. Collections and the
	// escape prefix must match the full build so hrefs render identically; the
	// LinkGraph is deliberately left unset — incremental rebuilds don't run link
	// validation, and b.linkGraph is only reset by a full Build().
	lr := b.mdRenderer.LinkRenderer()
	lr.PageIndex = newPageIndex
	lr.URLResolver = b.urlResolver
	lr.Collections = b.lastCollections
	lr.SiteRootEscapePrefix = b.config.LinkValidation.SiteRootEscapePrefix

	deps := markdownRenderDeps{
		scProcessor:    b.lastScProcessor,
		shortcodesHash: b.lastShortcodesHash,
		resolutionKey:  b.resolutionKey,
		iconRenderKey:  b.lastIconRenderKey,
		rendererKey:    b.rendererKey,
		pageCache:      b.lastPageCache,
		assetPipeline:  b.lastAssetPipeline,
	}
	for i := range parsed {
		e := &parsed[i]
		if e.newPage.RawContent == "" {
			delete(mergedValidation, e.newPage.Permalink)
			continue
		}
		links, _, err := b.renderMarkdownPageSerial(e.newPage, deps, b.lastSiteCtx)
		if err != nil {
			devlog.Warn("build", "ContentRebuild: markdown render error: %v, falling back", err)
			return b.Build()
		}
		updateValidationEntry(mergedValidation, e.newPage, links)
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
		devlog.Warn("build", "ContentRebuild: template render error: %v, falling back", err)
		return b.Build()
	}

	if err := b.renderDirtyCollectionPagination(b.lastCollections, dirtyPermalinks, b.lastSiteCtx, &dirtyRendered); err != nil {
		devlog.Warn("build", "ContentRebuild: paginated list render error: %v, falling back", err)
		return b.Build()
	}

	rebuildTaxByLang := newTaxByLang
	if rebuildTaxByLang == nil {
		rebuildTaxByLang = map[string]map[string]*engine.Taxonomy{"": newTaxonomies}
	}
	for lang, langTax := range rebuildTaxByLang {
		for taxName, tax := range langTax {
			cfg := b.config.Taxonomies[taxName]
			if !cfg.ShouldRender() {
				continue
			}
			if _, isDirty := dirtyPermalinks[tax.Permalink]; isDirty {
				termEntries := taxonomy.ComputeTermEntries(tax)
				taxStub := buildTaxonomyIndexStub(tax, termEntries, lang)
				resolvePermalinks(b.urlResolver, []*engine.Page{taxStub})
				rd := sardetemplate.BuildRouteData(taxStub, b.lastSiteCtx, b.themeConfig)
				resolveRouteAssets(b.urlResolver, rd)
				html, err := b.tmplEngine.Render(rd.Template, rd)
				if err != nil {
					return b.Build()
				}
				dirtyRendered = append(dirtyRendered, RenderedPage{
					Page: taxStub, HTML: html,
					OutPath: PageOutputPath(b.urlResolver.OutputRelPath(taxStub.RelPermalink, taxStub.Lang, "")),
				})
			}
			paginateBy := cfg.PaginateBy
			if paginateBy <= 0 {
				paginateBy = consts.DefaultPaginateBy
			}
			for _, term := range tax.Terms {
				if _, isDirty := dirtyPermalinks[term.Permalink]; isDirty {
					termStub := buildTermStub(tax, term, lang)
					resolvePermalinks(b.urlResolver, []*engine.Page{termStub})
					rd := sardetemplate.BuildRouteData(termStub, b.lastSiteCtx, b.themeConfig)
					resolveRouteAssets(b.urlResolver, rd)
					html, err := b.tmplEngine.Render(rd.Template, rd)
					if err != nil {
						return b.Build()
					}
					dirtyRendered = append(dirtyRendered, RenderedPage{
						Page: termStub, HTML: html,
						OutPath: PageOutputPath(b.urlResolver.OutputRelPath(termStub.RelPermalink, termStub.Lang, "")),
					})
				}
				totalTermPages := (len(term.Pages) + paginateBy - 1) / paginateBy
				for n := 2; n <= totalTermPages; n++ {
					permalink := sardetemplate.PaginationURL(term.Permalink, n)
					if _, isDirty := dirtyPermalinks[permalink]; !isDirty {
						continue
					}
					paginatedStub := buildTermPaginatedStub(tax, term, permalink, n, lang)
					resolvePermalinks(b.urlResolver, []*engine.Page{paginatedStub})
					rd := sardetemplate.BuildRouteData(paginatedStub, b.lastSiteCtx, b.themeConfig)
					resolveRouteAssets(b.urlResolver, rd)
					html, err := b.tmplEngine.Render(rd.Template, rd)
					if err != nil {
						return b.Build()
					}
					dirtyRendered = append(dirtyRendered, RenderedPage{
						Page: paginatedStub, HTML: html,
						OutPath: PageOutputPath(b.urlResolver.OutputRelPath(paginatedStub.RelPermalink, paginatedStub.Lang, "")),
					})
				}
			}
		}
	}

	if !b.devMode && config.BoolVal(b.config.Build.Minify, true) {
		minifyRendered(dirtyRendered, false, 0)
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

	for lang, langTax := range rebuildTaxByLang {
		patchedAllPages = appendTaxonomyStubsForLang(patchedAllPages, langTax, b.config.Taxonomies, lang)
	}
	b.lastSiteCtx.Pages = patchedAllPages

	b.lastAllPages = patchedAllPages
	b.lastTaxByLang = newTaxByLang
	b.lastPageIndex = newPageIndex
	b.lastValidationData = mergedValidation

	return &engine.BuildResult{
		PageCount:   len(dirtyRendered),
		Duration:    time.Since(start),
		Warnings:    warnings,
		OutputDir:   b.lastOutputDir,
		LogMessages: nil,
	}, nil
}

