package collection

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/internal/content"
	"github.com/coderoo-dev/coderoo/internal/engine"
	"github.com/coderoo-dev/coderoo/internal/navigation"
)

// BuildCollections groups discovered files into typed collections, builds Pages,
// applies sorting, section trees, draft filtering, and prev/next wiring.
// Returns the collections map and any validation warnings.
func BuildCollections(
	files []content.ContentFile,
	siteCfg *config.SiteConfig,
	contentDir string,
) (map[string]*engine.Collection, []engine.ValidationWarning, error) {
	grouped := groupByCollection(files)
	collections := make(map[string]*engine.Collection)
	var allWarnings []engine.ValidationWarning

	includeDrafts := config.BoolVal(siteCfg.Build.Drafts, false)
	includeFuture := config.BoolVal(siteCfg.Build.Future, false)
	now := time.Now()

	for name, colFiles := range grouped {
		if name == "" {
			continue // standalone/home pages handled separately
		}

		// 1. Infer defaults from directory name
		collCfg := InferCollection(name)

		// 2. Merge with site.yaml overrides
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
		pages, warnings, err := buildPages(colFiles, contentDir, collCfg, schema, siteCfg.Content.SummaryLength, string(siteCfg.Build.LastUpdated))
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

		// 9. Build nav tree (docs-layout) and wire prev/next
		// When multi-language, build per-language trees; otherwise single tree.
		langs := collectLanguages(pages)
		if len(langs) > 1 {
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
					wirePrevNext(langPages)
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
				wirePrevNext(pages)
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

	pages, _, err := buildPages(filtered, contentDir, nil, nil, summaryLength, lastUpdatedStrategy)
	return pages, err
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
	parser := &content.Parser{}
	inferrer := &content.Inferrer{LastUpdatedStrategy: lastUpdatedStrategy}
	transformer := &content.Transformer{SummaryLength: summaryLength}
	validator := &content.Validator{}

	var pages []*engine.Page
	var warnings []engine.ValidationWarning

	for _, cf := range files {
		raw, err := os.ReadFile(cf.FilePath)
		if err != nil {
			return nil, nil, err
		}

		// Parse frontmatter
		fmMap, body, err := parser.Parse(raw)
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

		// Parse typed frontmatter
		fm, _, err := content.ParseFrontmatter(raw)
		if err != nil {
			return nil, nil, err
		}

		// Build Page
		page := &engine.Page{
			Title:         fm.Title,
			Slug:          fm.Slug,
			Date:          fm.Date,
			Updated:       fm.Updated,
			Draft:       fm.Draft,
			PublishDate: fm.PublishDate,
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
		if fm.Prev != "" {
			page.Params["prev"] = fm.Prev
		}
		if fm.Next != "" {
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
			page.Params["hero"] = fm.Hero
		}
		if fm.EditURL != nil {
			page.Params["edit_url"] = *fm.EditURL
		}
		if fm.ShowUpdated != nil {
			page.Params["show_updated"] = *fm.ShowUpdated
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

		pages = append(pages, page)
	}

	return pages, warnings, nil
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

func wirePrevNext(pages []*engine.Page) {
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
