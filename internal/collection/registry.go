package collection

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/navigation"
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
	includeExpired := config.BoolVal(siteCfg.Build.Expired, false)
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

				// Validate versioning config
				if scfg.Versioning != nil {
					if err := config.ValidateVersioning(name, scfg.Versioning); err != nil {
						return nil, nil, err
					}
				}
			}
		}

		// 2b. Overlay sidebar.yaml overrides (independent of sarde.yaml
		// entries). Non-sidebar layouts skip the overlay entirely:
		// sidebarFileCollectionWarnings already emits the single "entry
		// ignored" warning for them, and attaching overrides anyway would
		// add one bogus unmatched-key warning per key (their nav trees are
		// never built).
		if siteCfg.SidebarFile != nil && engine.LayoutHasSidebar(collCfg.Layout) {
			entry := siteCfg.SidebarFile[name]
			collCfg = ApplySidebarFile(collCfg, entry)
			if entry != nil && len(entry.Items) > 0 {
				allWarnings = append(allWarnings, engine.ValidationWarning{
					File:    consts.FileSidebarConfig,
					Field:   name + ".items",
					Message: "structural sidebar items are not implemented yet; ignoring",
					Level:   "warning",
				})
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
		pages, warnings, err := buildPagesWithOptions(colFiles, contentDir, collCfg, schema, siteCfg.Content.SummaryLength, string(siteCfg.Build.LastUpdated), opts, siteCfg.Taxonomies)
		if err != nil {
			return nil, nil, err
		}
		allWarnings = append(allWarnings, warnings...)

		// 5. Filter drafts and future content
		pages = filterExcluded(pages, includeDrafts, includeFuture, includeExpired, now)

		// 5b. Normalize root-level pages in versioned collections: assign
		// Version=lastVersion to pages at the collection root (no vN/ directory).
		// The latest version's content lives at the root; older versions live in
		// vN/ subdirectories. This ensures every page in a versioned collection
		// has a concrete Version, eliminating URL collisions between root pages
		// and "latest" pages that previously required ResolveVersionShadowing.
		if collCfg.Versioning != nil && collCfg.Versioning.Enabled && collCfg.Versioning.LastVersion != "" {
			normalizeRootVersionPages(pages, collCfg.Versioning.LastVersion)
		}

		// 6. Sort pages
		SortPages(pages, collCfg.SortBy, collCfg.SortOrder)

		// 7. Build section tree
		sections := BuildSectionTree(pages, name)

		// 7b. Apply frontmatter cascade from section _index.md
		ApplyCascade(pages)

		// 8. Find index page
		var indexPage *engine.Page
		for _, p := range pages {
			if p.Kind == engine.KindSection && sectionDir(p.RelPermalink, name) == "" {
				indexPage = p
				break
			}
		}

		// Auto-create a synthetic index page when no _index.md exists,
		// so every collection root renders a landing page.
		if indexPage == nil && len(pages) > 0 {
			indexPage = &engine.Page{
				PageIdentity: engine.PageIdentity{
					Title:        collectionTitle(name, nil),
					Slug:         name,
					RelPermalink: "/" + name + "/",
					Kind:         engine.KindSection,
				},
				PageI18n: engine.PageI18n{
					Lang: pages[0].Lang,
				},
				Params: map[string]any{},
			}
			for _, sec := range sections {
				if sec.Permalink == "/"+name+"/" {
					indexPage.Section = sec
					sec.IndexPage = indexPage
					break
				}
			}
			pages = append(pages, indexPage)
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

		if collCfg.Versioning != nil && collCfg.Versioning.Enabled {
			col.Versioning = collCfg.Versioning
			if DetectTabs(col) {
				col.IsTabbed = true
				col.CompositeTabSets = BuildCompositeTabSets(col, contentDir, langs)
			} else {
				col.CompositeNavTrees = BuildCompositeNavTrees(col, langs)
				if len(langs) > 0 {
					key := LangVersionKey(langs[0], collCfg.Versioning.LastVersion)
					col.NavTree = col.CompositeNavTrees[key]
				}
			}
		} else if DetectTabs(col) {
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

	allWarnings = append(allWarnings, sidebarFileCollectionWarnings(siteCfg.SidebarFile, collections)...)

	return collections, allWarnings, nil
}

// sidebarFileCollectionWarnings flags sidebar.yaml entries whose collection
// key matches no collection or targets a collection without a sidebar layout.
func sidebarFileCollectionWarnings(sf config.SidebarFile, collections map[string]*engine.Collection) []engine.ValidationWarning {
	if len(sf) == 0 {
		return nil
	}
	names := make([]string, 0, len(sf))
	for name := range sf {
		names = append(names, name)
	}
	sort.Strings(names)

	var warnings []engine.ValidationWarning
	for _, name := range names {
		col, ok := collections[name]
		if !ok {
			warnings = append(warnings, engine.ValidationWarning{
				File:    consts.FileSidebarConfig,
				Field:   name,
				Message: fmt.Sprintf("sidebar.yaml declares collection %q but no such collection exists", name),
				Level:   "warning",
			})
			continue
		}
		if col.Config == nil || !engine.LayoutHasSidebar(col.Config.Layout) {
			warnings = append(warnings, engine.ValidationWarning{
				File:    consts.FileSidebarConfig,
				Field:   name,
				Message: fmt.Sprintf("collection %q does not use a sidebar layout; sidebar.yaml entry ignored", name),
				Level:   "warning",
			})
		}
	}
	return warnings
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

	pages, _, err := buildPagesWithOptions(filtered, contentDir, nil, nil, summaryLength, lastUpdatedStrategy, opts, nil)
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
	taxCfg map[string]config.TaxonomyConfig,
) (*engine.Page, []engine.ValidationWarning, error) {
	pages, warnings, err := buildPages([]content.ContentFile{cf}, contentDir, collCfg, schema, summaryLength, lastUpdatedStrategy, taxCfg)
	if err != nil || len(pages) == 0 {
		return nil, warnings, err
	}
	return pages[0], warnings, nil
}

func filterExcluded(pages []*engine.Page, includeDrafts, includeFuture, includeExpired bool, now time.Time) []*engine.Page {
	var result []*engine.Page
	for _, p := range pages {
		if !content.ShouldExclude(p.Draft, p.PublishDate, p.ExpiryDate, includeDrafts, includeFuture, includeExpired, now) {
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
