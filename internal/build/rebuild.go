package build

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/getsarde/sarde/internal/collection"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/i18n"
	"github.com/getsarde/sarde/internal/plugin"
	"github.com/getsarde/sarde/internal/syntax"
	"github.com/getsarde/sarde/internal/taxonomy"
	sardetemplate "github.com/getsarde/sarde/internal/template"
	"github.com/getsarde/sarde/internal/workers"
)

// errFallBackToFull signals that the incremental path cannot handle the
// change; ContentRebuild responds by running a full Build().
var errFallBackToFull = errors.New("fall back to full rebuild")

// parsedEntry pairs a re-parsed page with the page it replaces.
type parsedEntry struct {
	filePath string
	newPage  *engine.Page
	old      *engine.Page
}

// incrementalRebuildState carries intermediate results between the phases of
// a single ContentRebuild invocation, mirroring buildState for full builds.
type incrementalRebuildState struct {
	start      time.Time
	contentDir string

	// Classify + parse
	parsed           []parsedEntry
	warnings         []engine.ValidationWarning
	dirtyCollections map[string]*engine.Collection

	// Patch
	patchedAllPages []*engine.Page
	dirtyPermalinks map[string]struct{}
	changedByPath   map[string]*engine.Page

	// i18n + taxonomies
	isMultiLang      bool
	newTaxonomies    map[string]*engine.Taxonomy
	newTaxByLang     map[string]map[string]*engine.Taxonomy
	rebuildTaxByLang map[string]map[string]*engine.Taxonomy
	newPageIndex     *content.PageIndex

	// Markdown re-render
	mergedValidation map[string]engine.ValidationEntry

	// Dirty render + write
	dirtyRendered []RenderedPage
	dirtyAliases  map[string]string
}

// ContentRebuild performs an incremental rebuild for content-only changes.
// It re-parses only the changed files, patches them into the existing collections,
// and renders/writes only the dirty pages. Falls back to full Build() on edge cases.
func (b *SiteBuilder) ContentRebuild(changedPaths []string) (*engine.BuildResult, error) {
	start := time.Now()

	if !b.built || b.lastCollections == nil || b.lastAssetPipeline == nil {
		return b.Build()
	}

	s := &incrementalRebuildState{
		start:            start,
		contentDir:       b.resolveContentDir(),
		dirtyCollections: make(map[string]*engine.Collection),
		dirtyPermalinks:  make(map[string]struct{}),
	}

	if err := b.classifyAndParseChanges(changedPaths, s); err != nil {
		return b.rebuildFallback(err)
	}
	if len(s.parsed) == 0 && len(s.dirtyCollections) == 0 {
		return &engine.BuildResult{PageCount: 0, Duration: time.Since(start)}, nil
	}
	if err := b.patchPages(s); err != nil {
		return b.rebuildFallback(err)
	}
	if err := b.rebuildIncrementalI18nAndTaxonomies(s); err != nil {
		return b.rebuildFallback(err)
	}
	if err := b.rerenderDirtyMarkdown(s); err != nil {
		return b.rebuildFallback(err)
	}
	if err := b.renderAndWriteDirtyPages(s); err != nil {
		return b.rebuildFallback(err)
	}
	return b.finishIncrementalRebuild(s)
}

// rebuildFallback runs a full Build() when the incremental path signals
// errFallBackToFull; any other error aborts the rebuild. An aborted rebuild
// may have already patched collections, the site context, or the template
// engine in place, so clear b.built to force the next change onto the
// full-build path instead of incrementally patching inconsistent state.
func (b *SiteBuilder) rebuildFallback(err error) (*engine.BuildResult, error) {
	if errors.Is(err, errFallBackToFull) {
		return b.Build()
	}
	b.built = false
	return nil, err
}

// classifyAndParseChanges re-parses each changed file, applying the digest
// gates (content unchanged, body-only vs frontmatter change) and the change
// classification that decides between the incremental path, a
// collection-scoped rebuild, and a full-build fallback.
func (b *SiteBuilder) classifyAndParseChanges(changedPaths []string, s *incrementalRebuildState) error {
	// Skip fallback pages: they share FilePath with their source page and
	// would overwrite the real page entry in the map.
	oldByPath := make(map[string]*engine.Page, len(b.lastAllPages))
	for _, p := range b.lastAllPages {
		if p.FilePath != "" && !p.IsFallback {
			oldByPath[p.FilePath] = p
		}
	}

	for _, path := range changedPaths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(b.projectDir, path)
		}
		path = filepath.Clean(path)

		base := filepath.Base(path)
		if base == "_index.md" || base == "404.md" || (strings.HasPrefix(base, "404.") && filepath.Ext(base) == ".md") {
			devlog.Warn("build", "ContentRebuild: structural content changed (%s), falling back to full rebuild", base)
			return errFallBackToFull
		}

		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			devlog.Warn("build", "ContentRebuild: file deleted, falling back to full rebuild")
			return errFallBackToFull
		}
		if err == nil && info.IsDir() {
			continue
		}

		cf, err := b.scanner.ClassifyFile(s.contentDir, path)
		if err != nil {
			devlog.Warn("build", "ContentRebuild: ClassifyFile error: %v, falling back", err)
			return errFallBackToFull
		}

		old := oldByPath[path]
		if old == nil {
			devlog.Warn("build", "ContentRebuild: new file detected, falling back to full rebuild")
			return errFallBackToFull
		}
		if len(old.Resources) > 0 || cf.IsBundle {
			devlog.Warn("build", "ContentRebuild: bundle content changed, falling back to full rebuild")
			return errFallBackToFull
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
			schema, _ = content.LoadSchema(filepath.Join(s.contentDir, cf.CollectionName))
		}

		newPage, pageWarnings, err := collection.BuildSinglePage(
			cf, s.contentDir, collCfg, schema,
			b.config.Content.SummaryLength,
			string(b.config.Build.LastUpdated),
			b.config.Taxonomies,
		)
		if err != nil {
			devlog.Warn("build", "ContentRebuild: parse error for %s: %v, falling back", path, err)
			return errFallBackToFull
		}
		s.warnings = append(s.warnings, pageWarnings...)

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
				return errFallBackToFull
			}

			switch classifyChange(old, newPage, cf) {
			case changeFullRebuild:
				devlog.Warn("build", "ContentRebuild: site-structural change detected, falling back to full rebuild")
				return errFallBackToFull
			case changeCollectionScoped:
				// rebuildCollectionNav only rewrites the plain NavTree/NavTrees.
				// Tabbed, versioned, and multi-language collections render their
				// sidebar from composite/tab nav structures it doesn't touch (and
				// its per-language rebuild mixes languages), so a scoped rebuild
				// would leave those sidebars stale. Fall back to a full build.
				if b.collectionNeedsFullNavRebuild(old.Collection) {
					devlog.Warn("build", "ContentRebuild: sort/nav change in a tabbed/versioned/multi-lang collection, falling back to full rebuild")
					return errFallBackToFull
				}
				if old.Collection != nil {
					devlog.Log("build", "ContentRebuild: sort/nav change in %s, will rebuild collection", old.Collection.Name)
					s.dirtyCollections[old.Collection.Name] = old.Collection
				}
			case changeIncremental:
				// pass through — incremental path handles it
			}
		}

		s.parsed = append(s.parsed, parsedEntry{
			filePath: path,
			newPage:  newPage,
			old:      old,
		})
	}
	return nil
}

// patchPages swaps the re-parsed pages into copies of the last build's page
// list and collection/section membership, rebuilds navigation for
// collection-scoped changes, and dirty-marks every permalink whose output
// could differ.
func (b *SiteBuilder) patchPages(s *incrementalRebuildState) error {
	patchedAllPages := make([]*engine.Page, len(b.lastAllPages))
	copy(patchedAllPages, b.lastAllPages)

	indexByPath := make(map[string]int, len(patchedAllPages))
	for i, p := range patchedAllPages {
		if p.FilePath != "" && !p.IsFallback {
			indexByPath[p.FilePath] = i
		}
	}

	s.changedByPath = make(map[string]*engine.Page, len(s.parsed))

	for _, e := range s.parsed {
		idx, ok := indexByPath[e.filePath]
		if !ok {
			return errFallBackToFull
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
		s.changedByPath[e.filePath] = e.newPage

		col := e.old.Collection
		if col != nil {
			replacePagePointer(&col.Pages, e.old, e.newPage)
			replacePagePointer(&col.Featured, e.old, e.newPage)
		}
		if e.old.Section != nil {
			replacePagePointer(&e.old.Section.Pages, e.old, e.newPage)
		}

		s.dirtyPermalinks[e.newPage.RelPermalink] = struct{}{}
		addCollectionListDirty(col, s.dirtyPermalinks)
	}

	// Collection-scoped rebuild: re-sort and rebuild navigation for any
	// collection whose sort or sidebar fields changed.
	for colName, col := range s.dirtyCollections {
		rebuildCollectionNav(col)
		for _, p := range col.Pages {
			s.dirtyPermalinks[p.RelPermalink] = struct{}{}
		}
		addCollectionListDirty(col, s.dirtyPermalinks)
		devlog.Log("build", "ContentRebuild: rebuilt navigation for collection %q (%d pages)", colName, len(col.Pages))
	}

	// Dirty-mark old taxonomy terms for pages whose terms changed, so
	// removed-term pages are re-rendered without the page. Use the taxonomy
	// set of the page's own language; the default set only covers the
	// default language on multi-language sites.
	if b.lastSiteCtx != nil {
		for _, e := range s.parsed {
			oldTax := b.lastSiteCtx.Taxonomies
			if langTax, ok := b.lastSiteCtx.TaxonomiesByLang[e.old.Lang]; ok {
				oldTax = langTax
			}
			if oldTax == nil {
				continue
			}
			addRemovedTermsDirty(e.old, e.newPage, oldTax, b.config.Taxonomies, s.dirtyPermalinks)
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

	s.patchedAllPages = patchedAllPages
	return nil
}

// rebuildIncrementalI18nAndTaxonomies regenerates i18n fallback pages and
// translation links, rebuilds taxonomies per language, rebuilds the page
// index, and refreshes the site context handed to the template engine.
func (b *SiteBuilder) rebuildIncrementalI18nAndTaxonomies(s *incrementalRebuildState) error {
	s.isMultiLang = b.config.I18n.IsMultiLang()
	if s.isMultiLang {
		defaultLang := b.config.I18n.GetDefaultLanguage()
		langCodes := b.config.I18n.LanguageCodes()
		weights := make(map[string]int)
		for code, lc := range b.config.I18n.Languages {
			weights[code] = lc.Weight
		}

		i18n.LinkTranslations(s.patchedAllPages, weights)

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

		fallbacks := i18n.GenerateFallbacks(s.patchedAllPages, langCodes, defaultLang, fbOpts)
		s.patchedAllPages = append(s.patchedAllPages, fallbacks...)

		if b.urlResolver != nil {
			resolvePermalinks(b.urlResolver, fallbacks)
		}

		for _, fb := range fallbacks {
			if fb.FilePath != "" {
				if _, ok := s.changedByPath[fb.FilePath]; ok {
					s.dirtyPermalinks[fb.Permalink] = struct{}{}
				}
			}
		}

		i18n.LinkAllTranslations(s.patchedAllPages, weights)
	}

	// Home layouts render recentEntries (title, date, summary) from the site
	// context, so any patched page can change the home output. Re-render every
	// home page (one per language, including regenerated fallbacks).
	for _, p := range s.patchedAllPages {
		if p.Kind == engine.KindHome {
			s.dirtyPermalinks[p.RelPermalink] = struct{}{}
		}
	}

	if s.isMultiLang {
		defaultLang := b.config.I18n.GetDefaultLanguage()
		langCodes := b.config.I18n.LanguageCodes()
		s.newTaxByLang = make(map[string]map[string]*engine.Taxonomy, len(langCodes))
		for _, code := range langCodes {
			langTax := taxonomy.BuildTaxonomies(s.patchedAllPages, b.config.Taxonomies, code)
			if _, err := taxonomy.EnrichTaxonomies(langTax, b.config.Taxonomies, b.projectDir, code); err != nil {
				return errFallBackToFull
			}
			if b.urlResolver != nil {
				for _, tax := range langTax {
					tax.Permalink = b.urlResolver.URL(tax.Permalink, code, "")
					for _, term := range tax.Terms {
						term.Permalink = b.urlResolver.URL(term.Permalink, code, "")
					}
				}
			}
			s.newTaxByLang[code] = langTax
		}
		s.newTaxonomies = s.newTaxByLang[defaultLang]
	} else {
		s.newTaxonomies = taxonomy.BuildTaxonomies(s.patchedAllPages, b.config.Taxonomies, "")
		if _, err := taxonomy.EnrichTaxonomies(s.newTaxonomies, b.config.Taxonomies, b.projectDir, ""); err != nil {
			return errFallBackToFull
		}
	}

	s.rebuildTaxByLang = s.newTaxByLang
	if s.rebuildTaxByLang == nil {
		s.rebuildTaxByLang = map[string]map[string]*engine.Taxonomy{"": s.newTaxonomies}
	}

	collection.LinkVersions(s.patchedAllPages)

	s.newPageIndex = content.BuildPageIndex(s.patchedAllPages)
	s.newPageIndex.AddAssets(filepath.Join(b.projectDir, consts.DirStatic))
	populatePageIndexHeadings(s.newPageIndex, s.patchedAllPages)
	emitCollisionWarnings(s.newPageIndex.Collisions())

	if s.newTaxByLang != nil {
		for _, langTax := range s.newTaxByLang {
			for _, e := range s.parsed {
				addTaxonomyDirtyForPage(e.filePath, langTax, b.config.Taxonomies, s.dirtyPermalinks)
			}
		}
	} else {
		for _, e := range s.parsed {
			addTaxonomyDirtyForPage(e.filePath, s.newTaxonomies, b.config.Taxonomies, s.dirtyPermalinks)
		}
	}

	b.lastSiteCtx.Collections = b.lastCollections
	b.lastSiteCtx.Taxonomies = s.newTaxonomies
	b.lastSiteCtx.TaxonomiesByLang = s.newTaxByLang
	b.lastSiteCtx.Pages = s.patchedAllPages
	b.lastSiteCtx.BuildTime = time.Now()
	b.tmplEngine.SetSiteContext(b.lastSiteCtx)
	b.tmplEngine.SetPageIndex(s.newPageIndex)

	// Inject Kazari CSS into the template engine (reuse existing engine on incremental rebuilds).
	if b.kazariEngine != nil {
		b.tmplEngine.SetCodeBlockCSS(b.kazariEngine.CSS())
	}
	return nil
}

// rerenderDirtyMarkdown re-renders markdown for every re-parsed page,
// merging the new link-validation entries over the last build's data and
// copying rendered content onto regenerated fallback pages.
func (b *SiteBuilder) rerenderDirtyMarkdown(s *incrementalRebuildState) error {
	s.mergedValidation = make(map[string]engine.ValidationEntry, len(b.lastValidationData))
	for k, v := range b.lastValidationData {
		s.mergedValidation[k] = v
	}

	// Configure link resolver for incremental re-render. Collections and the
	// escape prefix must match the full build so hrefs render identically; the
	// LinkGraph is deliberately left unset — incremental rebuilds don't run link
	// validation, and b.linkGraph is only reset by a full Build().
	lr := b.mdRenderer.LinkRenderer()
	lr.PageIndex = s.newPageIndex
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
	for i := range s.parsed {
		e := &s.parsed[i]
		if e.newPage.RawContent == "" {
			delete(s.mergedValidation, e.newPage.Permalink)
			continue
		}
		if b.checkSyntax {
			diags := syntax.Check(e.newPage.FilePath, []byte(e.newPage.RawContent), e.newPage.FrontmatterLines)
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
		links, _, err := b.renderMarkdownPageSerial(e.newPage, deps, b.lastSiteCtx)
		if err != nil {
			devlog.Warn("build", "ContentRebuild: markdown render error: %v, falling back", err)
			return errFallBackToFull
		}
		updateValidationEntry(s.mergedValidation, e.newPage, links)
		setPageIndexHeadings(s.newPageIndex, e.newPage)
	}
	copyRenderedContentToFallbacks(s.patchedAllPages, s.changedByPath, s.newPageIndex)
	return nil
}

// renderAndWriteDirtyPages template-renders every dirty page, regenerates
// dirty collection pagination and taxonomy stubs, minifies, and writes the
// results to the output directory.
func (b *SiteBuilder) renderAndWriteDirtyPages(s *incrementalRebuildState) error {
	var dirtyPages []*engine.Page
	for _, page := range s.patchedAllPages {
		if _, isDirty := s.dirtyPermalinks[page.RelPermalink]; !isDirty {
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
		return errFallBackToFull
	}

	if err := b.renderDirtyCollectionPagination(b.lastCollections, s.dirtyPermalinks, b.lastSiteCtx, &dirtyRendered); err != nil {
		devlog.Warn("build", "ContentRebuild: paginated list render error: %v, falling back", err)
		return errFallBackToFull
	}

	for lang, langTax := range s.rebuildTaxByLang {
		for taxName, tax := range langTax {
			cfg := b.config.Taxonomies[taxName]
			if !cfg.ShouldRender() {
				continue
			}
			if _, isDirty := s.dirtyPermalinks[tax.Permalink]; isDirty {
				termEntries := taxonomy.ComputeTermEntries(tax)
				taxStub := buildTaxonomyIndexStub(tax, termEntries, lang)
				resolvePermalinks(b.urlResolver, []*engine.Page{taxStub})
				rd := sardetemplate.BuildRouteData(taxStub, b.lastSiteCtx, b.themeConfig)
				resolveRouteAssets(b.urlResolver, rd)
				html, err := b.tmplEngine.Render(rd.Template, rd)
				if err != nil {
					return errFallBackToFull
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
				if _, isDirty := s.dirtyPermalinks[term.Permalink]; isDirty {
					termStub := buildTermStub(tax, term, lang)
					resolvePermalinks(b.urlResolver, []*engine.Page{termStub})
					rd := sardetemplate.BuildRouteData(termStub, b.lastSiteCtx, b.themeConfig)
					resolveRouteAssets(b.urlResolver, rd)
					html, err := b.tmplEngine.Render(rd.Template, rd)
					if err != nil {
						return errFallBackToFull
					}
					dirtyRendered = append(dirtyRendered, RenderedPage{
						Page: termStub, HTML: html,
						OutPath: PageOutputPath(b.urlResolver.OutputRelPath(termStub.RelPermalink, termStub.Lang, "")),
					})
				}
				totalTermPages := (len(term.Pages) + paginateBy - 1) / paginateBy
				for n := 2; n <= totalTermPages; n++ {
					permalink := sardetemplate.PaginationURL(term.Permalink, n)
					if _, isDirty := s.dirtyPermalinks[permalink]; !isDirty {
						continue
					}
					paginatedStub := buildTermPaginatedStub(tax, term, permalink, n, lang)
					resolvePermalinks(b.urlResolver, []*engine.Page{paginatedStub})
					rd := sardetemplate.BuildRouteData(paginatedStub, b.lastSiteCtx, b.themeConfig)
					resolveRouteAssets(b.urlResolver, rd)
					html, err := b.tmplEngine.Render(rd.Template, rd)
					if err != nil {
						return errFallBackToFull
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
			return fmt.Errorf("incremental write %s: %w", rp.OutPath, err)
		}
	}
	for aliasPath, target := range dirtyAliases {
		if _, err := writeOutputFile(b.lastOutputDir, PageOutputPath(aliasPath), []byte(redirectHTML(target))); err != nil {
			return fmt.Errorf("incremental write alias %s: %w", aliasPath, err)
		}
	}

	s.dirtyRendered = dirtyRendered
	s.dirtyAliases = dirtyAliases
	return nil
}

// finishIncrementalRebuild re-adds taxonomy stubs to the page list, updates
// the builder's last-build snapshot, runs the BuildDone plugin hook, and
// assembles the incremental BuildResult.
func (b *SiteBuilder) finishIncrementalRebuild(s *incrementalRebuildState) (*engine.BuildResult, error) {
	for lang, langTax := range s.rebuildTaxByLang {
		s.patchedAllPages = appendTaxonomyStubsForLang(s.patchedAllPages, langTax, b.config.Taxonomies, lang)
	}
	b.lastSiteCtx.Pages = s.patchedAllPages

	b.lastAllPages = s.patchedAllPages
	b.lastTaxByLang = s.newTaxByLang
	b.lastPageIndex = s.newPageIndex
	b.lastValidationData = s.mergedValidation

	buildLogger := engine.NewBuildLogger()
	var pluginWarnings []engine.ValidationWarning
	buildDoneCtx := &plugin.BuildDoneContext{
		Config:         b.config,
		OutputDir:      b.lastOutputDir,
		Pages:          s.patchedAllPages,
		Collections:    b.lastCollections,
		Site:           b.lastSiteCtx,
		Resolver:       b.urlResolver,
		PageIndex:      s.newPageIndex,
		ValidationData: s.mergedValidation,
		DevMode:        b.devMode,
	}
	buildDoneCtx.SetWarnings(&pluginWarnings)
	buildDoneCtx.SetLogger(buildLogger)
	if err := b.pluginMgr.RunBuildDone(buildDoneCtx); err != nil {
		devlog.Warn("build", "BuildDone plugin error during incremental rebuild: %v, falling back to full build", err)
		return b.Build()
	}
	warnings := append(s.warnings, pluginWarnings...)

	return &engine.BuildResult{
		PageCount:   len(s.dirtyRendered),
		Duration:    time.Since(s.start),
		Warnings:    warnings,
		OutputDir:   b.lastOutputDir,
		LogMessages: nil,
	}, nil
}
