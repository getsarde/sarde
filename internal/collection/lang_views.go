package collection

import (
	"sort"

	"github.com/getsarde/sarde/internal/engine"
)

// BuildLanguageViews fills each collection's per-language views on a
// multi-language site: the pages, featured pages and root index page of
// every language, with i18n fallback copies included. Collections build
// their section tree once from every language's pages, and RelPermalink
// carries no language, so a folder holding an _index.md in two languages
// keeps only one of them; a list page that reads its section's pages would
// show one language's cards, or none.
//
// Docs collections already get a section tree per language when their
// sidebars are built. For every other unversioned collection (blog, labs,
// slides, default lists) this rebuilds the section tree per language, so each
// page's Section holds only its own language, and labs get per-language
// sidebars and step counts. Collections, Sections, Title and IndexPage are
// left as they are.
//
// It must run after fallback copies exist (they are part of each view) and
// may run again on an incremental rebuild: every call starts from scratch.
// On a single-language site it clears the views, and the accessors fall back
// to the shared fields.
func BuildLanguageViews(collections map[string]*engine.Collection, allPages []*engine.Page, langs []string, defaultLang string) {
	fallbacks := make(map[*engine.Collection][]*engine.Page)
	for _, p := range allPages {
		if p.IsFallback && p.Collection != nil {
			fallbacks[p.Collection] = append(fallbacks[p.Collection], p)
		}
	}

	names := make([]string, 0, len(collections))
	for name := range collections {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		col := collections[name]
		col.LangPages, col.LangIndex, col.LangFeatured, col.LabNavTreesByLang = nil, nil, nil, nil
		if len(langs) <= 1 {
			continue
		}

		members := make([]*engine.Page, 0, len(col.Pages)+len(fallbacks[col]))
		members = append(members, col.Pages...)
		members = append(members, fallbacks[col]...)

		cfg := col.Config
		if cfg == nil {
			cfg = &engine.CollectionConfig{}
		}
		restructure := !engine.LayoutHasSidebar(cfg.Layout) && (cfg.Versioning == nil || !cfg.Versioning.Enabled)
		labs := restructure && IsLabsName(name)

		col.LangPages = make(map[string][]*engine.Page, len(langs))
		col.LangIndex = make(map[string]*engine.Page, len(langs))
		col.LangFeatured = make(map[string][]*engine.Page, len(langs))
		if labs {
			col.LabNavTreesByLang = make(map[string]map[string]*engine.NavTree, len(langs))
		}
		multiCourse := false

		for _, lang := range langs {
			pages := filterByLang(members, lang)
			SortPages(pages, cfg.SortBy, cfg.SortOrder)

			if restructure {
				sections := BuildSectionTree(pages, name)
				setCollectionOnSections(sections, col)
				if labs {
					// The shared tree misplaced the pages of the language that
					// lost a folder's slot, so the labs layout default is
					// applied again; it never overrides a set layout.
					applyLabsLayoutDefault(pages)
					trees, mc := buildLabNavTrees(col, findCollectionRootSection(sections))
					col.LabNavTreesByLang[lang] = trees
					multiCourse = multiCourse || mc
				}
			}

			col.LangPages[lang] = pages
			col.LangFeatured[lang] = extractFeatured(pages)
			col.LangIndex[lang] = rootIndexPage(pages, name, cfg)
		}

		if labs {
			col.IsMultiCourse = multiCourse
			if trees, ok := col.LabNavTreesByLang[defaultLang]; ok {
				col.LabNavTrees = trees
			}
		}
	}
}

// rootIndexPage returns the collection's root _index.md among pages, or nil.
// RelPermalink carries no version, so in a versioned collection every
// version's root _index.md looks like the collection root; the latest
// version's is the collection's own.
func rootIndexPage(pages []*engine.Page, name string, cfg *engine.CollectionConfig) *engine.Page {
	latest := ""
	if vc := cfg.Versioning; vc != nil && vc.Enabled {
		latest = vc.LastVersion
	}
	var first *engine.Page
	for _, p := range pages {
		if p.Kind != engine.KindSection || sectionDir(p.RelPermalink, name) != "" {
			continue
		}
		if latest == "" || p.Version == latest {
			return p
		}
		if first == nil {
			first = p
		}
	}
	return first
}
