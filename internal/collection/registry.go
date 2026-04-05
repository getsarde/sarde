package collection

import (
	"os"
	"path/filepath"

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

		// 3. Load schema (optional)
		schema, _ := content.LoadSchema(filepath.Join(contentDir, name))

		// 4. Build pages
		pages, warnings, err := buildPages(colFiles, contentDir, collCfg, schema, siteCfg.Content.SummaryLength)
		if err != nil {
			return nil, nil, err
		}
		allWarnings = append(allWarnings, warnings...)

		// 5. Filter drafts
		if !includeDrafts {
			pages = filterDrafts(pages)
		}

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
				if collCfg.Layout == engine.LayoutDocs {
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
			if collCfg.Layout == engine.LayoutDocs {
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
) ([]*engine.Page, error) {
	grouped := groupByCollection(files)
	rootFiles := grouped[""]
	if len(rootFiles) == 0 {
		return nil, nil
	}

	pages, _, err := buildPages(rootFiles, contentDir, nil, nil, summaryLength)
	return pages, err
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
) ([]*engine.Page, []engine.ValidationWarning, error) {
	parser := &content.Parser{}
	inferrer := &content.Inferrer{}
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
			Draft:         fm.Draft,
			Weight:        fm.Weight,
			Description:   fm.Description,
			Image:         fm.Image,
			Tags:          fm.Tags,
			Categories:    fm.Categories,
			Aliases:       fm.Aliases,
			SidebarLabel:  fm.SidebarLabel,
			SidebarHidden: fm.SidebarHidden,
			Badge:         fm.Badge,
			BadgeColor:    fm.BadgeColor,
			Kind:          cf.Kind,
			FilePath:      cf.FilePath,
			RawContent:    body,
			Params:        fm.Params,
			Lang:          cf.Lang,
			LangRelPath:   cf.LangRelPath,
		}

		// Transfer section-specific fields to Params for section builder
		if fm.Transparent {
			if page.Params == nil {
				page.Params = make(map[string]any)
			}
			page.Params["transparent"] = true
		}
		if fm.Render != nil && !*fm.Render {
			if page.Params == nil {
				page.Params = make(map[string]any)
			}
			page.Params["render"] = false
		}

		// Infer defaults (title, date, slug, weight)
		if err := inferrer.Infer(page, cf.FilePath); err != nil {
			return nil, nil, err
		}

		// Compute permalink
		page.RelPermalink = content.ComputePermalink(contentDir, cf.FilePath)
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

func filterDrafts(pages []*engine.Page) []*engine.Page {
	var result []*engine.Page
	for _, p := range pages {
		if !p.Draft {
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
