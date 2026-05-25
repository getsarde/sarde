package collection

import (
	"context"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/navigation"
	"github.com/frostybee/sarde/internal/workers"
	"golang.org/x/sync/errgroup"
)

// BuildOptions controls content parsing work inside collection builders.
type BuildOptions struct {
	Parallel    bool
	WorkerCount int
}

// BuildCollections groups discovered files into typed collections, builds Pages,
// applies sorting, section trees, draft filtering, and prev/next wiring.
// Returns the collections map and any validation warnings.
func BuildCollections(
	files []content.ContentFile,
	siteCfg *config.SiteConfig,
	contentDir string,
) (map[string]*engine.Collection, []engine.ValidationWarning, error) {
	return BuildCollectionsWithOptions(files, siteCfg, contentDir, BuildOptions{})
}

// BuildCollectionsWithOptions groups discovered files into typed collections with optional parallel parsing.
func BuildCollectionsWithOptions(
	files []content.ContentFile,
	siteCfg *config.SiteConfig,
	contentDir string,
	opts BuildOptions,
) (map[string]*engine.Collection, []engine.ValidationWarning, error) {
	grouped := groupByCollection(files)
	collections := make(map[string]*engine.Collection)
	var allWarnings []engine.ValidationWarning

	includeDrafts := config.BoolVal(siteCfg.Build.Drafts, false)
	includeFuture := config.BoolVal(siteCfg.Build.Future, false)
	now := time.Now()

	names := make([]string, 0, len(grouped))
	for name := range grouped {
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		colFiles := grouped[name]

		// 1. Infer defaults from directory name
		collCfg := InferCollection(name)

		// 2. Merge with sarde.yaml overrides
		if siteCfg.Collections != nil {
			if scfg, ok := siteCfg.Collections[name]; ok {
				collCfg = MergeCollectionConfig(collCfg, scfg)
			}
		}

		// Apply site-level permalink pattern if collection doesn't have one
		if collCfg.Permalink == "" {
			if siteCfg.Permalinks != nil {
				if pattern, ok := siteCfg.Permalinks[name]; ok {
					collCfg.Permalink = pattern
				}
			}
		}

		// 3. Load schema (optional)
		schema, _ := content.LoadSchema(filepath.Join(contentDir, name))

		// 4. Build pages
		pages, warnings, err := buildPagesWithOptions(colFiles, contentDir, collCfg, schema, siteCfg.Content.SummaryLength, string(siteCfg.Build.LastUpdated), opts)
		if err != nil {
			return nil, nil, err
		}
		allWarnings = append(allWarnings, warnings...)

		// 5. Filter drafts and future content
		pages = filterExcluded(pages, includeDrafts, includeFuture, now)

		// 6. Sort pages
		SortPages(pages, collCfg.SortBy, collCfg.SortOrder)

		// 7. Build section tree
		sections := BuildSectionTree(pages, name)

		// 8. Find index page
		var indexPage *engine.Page
		for _, p := range pages {
			if p.Kind == engine.KindSection && sectionDir(p.RelPermalink, name) == "" {
				indexPage = p
				break
			}
		}

		col := &engine.Collection{
			Name:      name,
			Title:     collectionTitle(name, indexPage),
			Config:    collCfg,
			Pages:     pages,
			Featured:  extractFeatured(pages),
			Sections:  sections,
			IndexPage: indexPage,
		}

		// Set Collection backref on all pages and sections
		for _, p := range pages {
			p.Collection = col
		}
		setCollectionOnSections(sections, col)

		// 9. Build navigation (versioned, tabbed, or standard)
		langs := collectLanguages(pages)

		// Versioning annotation runs first (sets Page.Version, rewrites URLs).
		if collCfg.Versioning != nil && collCfg.Versioning.Enabled {
			col.Versioning = collCfg.Versioning
			AnnotateVersions(col)
			col.VersionNavTrees = BuildVersionedNavTrees(col)
			if lv := collCfg.Versioning.LastVersion; lv != "" {
				col.NavTree = col.VersionNavTrees[lv]
			}
			if col.NavTree == nil {
				col.NavTree = col.VersionNavTrees[""]
			}
		}

		// Tab detection runs after versioning (topLevelSections filters out version dirs).
		if DetectTabs(col) {
			col.IsTabbed = true
			if len(langs) > 1 {
				col.Tabs = BuildTabsI18n(col, contentDir, langs)
			} else {
				col.Tabs = BuildTabs(col, contentDir)
			}
		} else if len(langs) > 1 {
			// Multi-language, non-tabbed
			col.NavTrees = make(map[string]*engine.NavTree)
			for _, lang := range langs {
				langPages := filterByLang(pages, lang)
				if engine.LayoutHasSidebar(collCfg.Layout) {
					langCol := &engine.Collection{
						Name:     col.Name,
						Config:   col.Config,
						Pages:    langPages,
						Sections: BuildSectionTree(langPages, name),
					}
					tree := navigation.BuildNavTree(langCol)
					col.NavTrees[lang] = tree
					navigation.WirePrevNextFromTree(tree)
				} else {
					WirePrevNext(langPages)
				}
			}
			// Set default NavTree for backward compat
			if len(langs) > 0 {
				col.NavTree = col.NavTrees[langs[0]]
			}
		} else {
			if engine.LayoutHasSidebar(collCfg.Layout) {
				col.NavTree = navigation.BuildNavTree(col)
				navigation.WirePrevNextFromTree(col.NavTree)
			} else {
				WirePrevNext(pages)
			}
		}

		collections[name] = col
	}

	return collections, allWarnings, nil
}

// BuildStandalonePages builds Page objects for root-level files (home, standalone).
func BuildStandalonePages(
	files []content.ContentFile,
	contentDir string,
	summaryLength int,
	lastUpdatedStrategy string,
) ([]*engine.Page, error) {
	return BuildStandalonePagesWithOptions(files, contentDir, summaryLength, lastUpdatedStrategy, BuildOptions{})
}

// BuildStandalonePagesWithOptions builds root-level pages with optional parallel parsing.
func BuildStandalonePagesWithOptions(
	files []content.ContentFile,
	contentDir string,
	summaryLength int,
	lastUpdatedStrategy string,
	opts BuildOptions,
) ([]*engine.Page, error) {
	grouped := groupByCollection(files)
	rootFiles := grouped[""]
	if len(rootFiles) == 0 {
		return nil, nil
	}

	// Exclude 404.md — it has its own render path in the builder.
	filtered := rootFiles[:0]
	for _, f := range rootFiles {
		base := filepath.Base(f.FilePath)
		if base == "404.md" || strings.HasPrefix(base, "404.") && strings.HasSuffix(base, ".md") {
			continue
		}
		filtered = append(filtered, f)
	}

	pages, _, err := buildPagesWithOptions(filtered, contentDir, nil, nil, summaryLength, lastUpdatedStrategy, opts)
	return pages, err
}

// BuildSinglePage parses, infers, and transforms a single ContentFile into a Page.
// Used by incremental rebuild to re-parse a changed file without rebuilding all collections.
func BuildSinglePage(
	cf content.ContentFile,
	contentDir string,
	collCfg *engine.CollectionConfig,
	schema *engine.FrontmatterSchema,
	summaryLength int,
	lastUpdatedStrategy string,
) (*engine.Page, []engine.ValidationWarning, error) {
	pages, warnings, err := buildPages([]content.ContentFile{cf}, contentDir, collCfg, schema, summaryLength, lastUpdatedStrategy)
	if err != nil || len(pages) == 0 {
		return nil, warnings, err
	}
	return pages[0], warnings, nil
}

// extractFeatured returns the subset of pages whose frontmatter Params has
// `featured: true`.
func extractFeatured(pages []*engine.Page) []*engine.Page {
	var out []*engine.Page
	for _, p := range pages {
		if p.Params == nil {
			continue
		}
		if v, ok := p.Params["featured"]; ok {
			if b, _ := v.(bool); b {
				out = append(out, p)
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func groupByCollection(files []content.ContentFile) map[string][]content.ContentFile {
	result := make(map[string][]content.ContentFile)
	for _, f := range files {
		result[f.CollectionName] = append(result[f.CollectionName], f)
	}
	return result
}

func buildPages(
	files []content.ContentFile,
	contentDir string,
	collCfg *engine.CollectionConfig,
	schema *engine.FrontmatterSchema,
	summaryLength int,
	lastUpdatedStrategy string,
) ([]*engine.Page, []engine.ValidationWarning, error) {
	return buildPagesWithOptions(files, contentDir, collCfg, schema, summaryLength, lastUpdatedStrategy, BuildOptions{})
}

func buildPagesWithOptions(
	files []content.ContentFile,
	contentDir string,
	collCfg *engine.CollectionConfig,
	schema *engine.FrontmatterSchema,
	summaryLength int,
	lastUpdatedStrategy string,
	opts BuildOptions,
) ([]*engine.Page, []engine.ValidationWarning, error) {
	if !workers.ShouldParallelize(opts.Parallel, len(files), opts.WorkerCount) {
		var pages []*engine.Page
		var warnings []engine.ValidationWarning
		for _, cf := range files {
			page, pageWarnings, err := buildPage(cf, contentDir, collCfg, schema, summaryLength, lastUpdatedStrategy)
			if err != nil {
				return nil, nil, err
			}
			pages = append(pages, page)
			warnings = append(warnings, pageWarnings...)
		}
		return pages, warnings, nil
	}

	type result struct {
		page     *engine.Page
		warnings []engine.ValidationWarning
	}
	results := make([]result, len(files))
	limit := opts.WorkerCount
	if limit <= 0 {
		limit = workers.Limit(len(files))
	}
	g, _ := errgroup.WithContext(context.Background())
	g.SetLimit(limit)
	for i, cf := range files {
		i, cf := i, cf
		g.Go(func() error {
			page, pageWarnings, err := buildPage(cf, contentDir, collCfg, schema, summaryLength, lastUpdatedStrategy)
			if err != nil {
				return err
			}
			results[i] = result{page: page, warnings: pageWarnings}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}

	pages := make([]*engine.Page, 0, len(files))
	var warnings []engine.ValidationWarning
	for _, r := range results {
		pages = append(pages, r.page)
		warnings = append(warnings, r.warnings...)
	}
	return pages, warnings, nil
}

func buildPage(
	cf content.ContentFile,
	contentDir string,
	collCfg *engine.CollectionConfig,
	schema *engine.FrontmatterSchema,
	summaryLength int,
	lastUpdatedStrategy string,
) (*engine.Page, []engine.ValidationWarning, error) {
	inferrer := &content.Inferrer{LastUpdatedStrategy: lastUpdatedStrategy}
	transformer := &content.Transformer{SummaryLength: summaryLength}
	validator := &content.Validator{}

	var warnings []engine.ValidationWarning

	f, err := os.Open(cf.FilePath)
	if err != nil {
		return nil, nil, err
	}
	fi, _ := f.Stat()
	raw, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		return nil, nil, err
	}

	// Parse frontmatter: single pass produces both untyped map and typed struct.
	fmMap, fm, body, err := content.ParseAll(raw)
	if err != nil {
		return nil, nil, err
	}

	// Apply schema defaults
	if schema != nil {
		fmMap = content.ApplyDefaults(fmMap, schema)
	}

	// Validate against schema
	if schema != nil {
		w := validator.Validate(fmMap, schema)
		for i := range w {
			w[i].File = cf.RelPath
		}
		warnings = append(warnings, w...)
	}

	// Build Page
	page := &engine.Page{
		Title:         fm.Title,
		Slug:          fm.Slug,
		Date:          fm.Date,
		Updated:       fm.Updated,
		Draft:         fm.Draft,
		PublishDate:   fm.PublishDate,
		Weight:        fm.Weight,
		Description:   fm.Description,
		Image:         fm.Image,
		Tags:          fm.Tags,
		Categories:    fm.Categories,
		Aliases:       fm.Aliases,
		SidebarLabel:  fm.SidebarLabel,
		SidebarHidden: fm.SidebarHidden,
		Badge:         fm.Badge,
		Kind:          cf.Kind,
		FilePath:      cf.FilePath,
		RawContent:    body,
		Params:        fm.Params,
		Lang:          cf.Lang,
		LangRelPath:   cf.LangRelPath,
	}
	if rel, err := filepath.Rel(contentDir, cf.FilePath); err == nil {
		page.RelPath = filepath.ToSlash(rel)
	}

	// Ensure Params map is initialized for field transfers.
	if page.Params == nil {
		page.Params = make(map[string]any)
	}

	// Transfer section-specific fields to Params for section builder
	if fm.Transparent {
		page.Params["transparent"] = true
	}
	if fm.Render != nil && !*fm.Render {
		page.Params["render"] = false
	}
	// Transfer `featured` (unknown to typed Frontmatter) from the raw map.
	if b, ok := fmMap["featured"].(bool); ok && b {
		page.Params["featured"] = true
	}
	if fm.Template != "" {
		page.Params["template"] = fm.Template
	}
	if fm.Summary != "" {
		page.Summary = template.HTML(fm.Summary)
	}
	if fm.Prev != nil {
		page.Params["prev"] = fm.Prev
	}
	if fm.Next != nil {
		page.Params["next"] = fm.Next
	}
	if len(fm.SidebarAttrs) > 0 {
		page.Params["sidebar_attrs"] = fm.SidebarAttrs
	}
	if fm.Pagefind != nil {
		page.Params["pagefind"] = *fm.Pagefind
	}
	if fm.TOC != nil {
		page.Params["toc"] = *fm.TOC
	}
	if fm.TOCMinLevel > 0 {
		page.Params["toc_min_level"] = fm.TOCMinLevel
	}
	if fm.TOCMaxLevel > 0 {
		page.Params["toc_max_level"] = fm.TOCMaxLevel
	}
	if fm.SidebarGroup != "" {
		page.Params["sidebar_group"] = fm.SidebarGroup
	}
	if fm.Layout != "" {
		page.Params["layout"] = fm.Layout
	}
	if fm.Type != "" {
		page.Params["type"] = fm.Type
	}
	if fm.Hero != nil {
		fm.Hero.SanitizeAttrs()
		page.Params["hero"] = fm.Hero
	}
	if fm.EditURL != nil {
		if fm.EditURL.CustomURL != "" {
			page.Params["edit_url"] = fm.EditURL.CustomURL
		} else if fm.EditURL.Disabled {
			page.Params["edit_url"] = false
		}
	}
	if len(fm.Head) > 0 {
		var filtered []engine.HeadTag
		for _, h := range fm.Head {
			if engine.AllowedHeadTags[h.Tag] {
				filtered = append(filtered, h)
			}
		}
		if len(filtered) > 0 {
			page.Params["head"] = filtered
		}
	}
	if fm.Banner != nil && fm.Banner.Content != "" {
		page.Params["banner"] = fm.Banner
	}
	if fm.ShowUpdated != nil {
		page.Params["show_updated"] = *fm.ShowUpdated
	}
	if fm.Icon != "" {
		page.Params["icon"] = fm.Icon
	}

	// Pre-populate Date from the already-opened FileInfo to avoid a
	// redundant os.Stat inside Infer.
	if page.Date.IsZero() && fi != nil {
		page.Date = fi.ModTime()
	}

	// Infer defaults (title, date, slug, weight)
	if err := inferrer.Infer(page, cf.FilePath); err != nil {
		return nil, nil, err
	}

	// Compute permalink: use pattern if configured, else directory-based
	isIndex := filepath.Base(cf.FilePath) == "_index.md" || filepath.Base(cf.FilePath) == "index.md"
	if collCfg != nil && collCfg.Permalink != "" && !isIndex {
		vars := content.PermalinkVars{
			Slug:       page.Slug,
			Year:       page.Date.Format("2006"),
			Month:      page.Date.Format("01"),
			Day:        page.Date.Format("02"),
			Section:    extractSection(cf.FilePath, contentDir),
			Collection: cf.CollectionName,
			Title:      content.Slugify(page.Title),
		}
		page.RelPermalink = content.ComputePatternPermalink(collCfg.Permalink, vars)
	} else {
		page.RelPermalink = content.ComputePermalink(contentDir, cf.FilePath)
	}
	page.Permalink = page.RelPermalink

	// Transform (word count, reading time, summary)
	if err := transformer.Transform(page); err != nil {
		return nil, nil, err
	}

	// Copy bundle assets
	if cf.IsBundle {
		for _, asset := range cf.BundleAssets {
			page.Resources = append(page.Resources, engine.Resource{
				Name: filepath.Base(asset),
			})
		}
	}

	return page, warnings, nil
}

// extractSection returns the immediate parent directory name relative to the collection root.
func extractSection(filePath, contentDir string) string {
	rel, _ := filepath.Rel(contentDir, filePath)
	rel = filepath.ToSlash(rel)
	parts := strings.Split(rel, "/")
	// parts[0] is collection, parts[1..n-1] are sections, parts[n-1] is filename
	if len(parts) > 2 {
		return parts[len(parts)-2]
	}
	return ""
}

func filterExcluded(pages []*engine.Page, includeDrafts, includeFuture bool, now time.Time) []*engine.Page {
	var result []*engine.Page
	for _, p := range pages {
		if !content.ShouldExclude(p.Draft, p.PublishDate, includeDrafts, includeFuture, now) {
			result = append(result, p)
		}
	}
	return result
}

func WirePrevNext(pages []*engine.Page) {
	// Only wire non-section pages
	var contentPages []*engine.Page
	for _, p := range pages {
		if p.Kind != engine.KindSection {
			contentPages = append(contentPages, p)
		}
	}
	for i, p := range contentPages {
		if i > 0 {
			p.PrevPage = contentPages[i-1]
		}
		if i < len(contentPages)-1 {
			p.NextPage = contentPages[i+1]
		}
	}
}

func collectionTitle(name string, indexPage *engine.Page) string {
	if indexPage != nil && indexPage.Title != "" {
		return indexPage.Title
	}
	// Title-case the directory name
	return content.FilenameToTitle(name + ".md")
}

func setCollectionOnSections(sections []*engine.Section, col *engine.Collection) {
	for _, sec := range sections {
		sec.Collection = col
		setCollectionOnSections(sec.Sections, col)
	}
}

// collectLanguages returns the unique language codes found in the pages, in order of first appearance.
func collectLanguages(pages []*engine.Page) []string {
	seen := make(map[string]bool)
	var langs []string
	for _, p := range pages {
		if p.Lang != "" && !seen[p.Lang] {
			seen[p.Lang] = true
			langs = append(langs, p.Lang)
		}
	}
	return langs
}

// filterByLang returns only pages with the given language code.
func filterByLang(pages []*engine.Page, lang string) []*engine.Page {
	var result []*engine.Page
	for _, p := range pages {
		if p.Lang == lang {
			result = append(result, p)
		}
	}
	return result
}
