package collection

import (
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/navigation"
)

// RebuildNavTreesWithFallbacks rebuilds per-language nav trees after fallback
// pages have been generated, so every language gets a complete sidebar
// (real translations + fallback pages from the default language). contentDir
// must be the real content directory so per-tab nav.yaml files are honored on
// the rebuild exactly as in the initial build. The language-free NavTree
// fields hold defaultLang's tree.
func RebuildNavTreesWithFallbacks(collections map[string]*engine.Collection, allPages []*engine.Page, langs []string, contentDir, defaultLang string) {
	if len(langs) <= 1 {
		return
	}
	for _, col := range collections {
		if col.Config == nil || !engine.LayoutHasSidebar(col.Config.Layout) {
			continue
		}

		var colPages []*engine.Page
		for _, p := range allPages {
			if p.Collection != nil && p.Collection.Name == col.Name {
				colPages = append(colPages, p)
			}
		}

		// Versioned collections: rebuild composite nav trees or tab sets.
		if col.Config.Versioning != nil && col.Config.Versioning.Enabled {
			if col.IsTabbed {
				col.CompositeTabSets = BuildCompositeTabSets(col, contentDir, langs)
			} else {
				col.CompositeNavTrees = BuildCompositeNavTrees(col, langs)
				key := LangVersionKey(primaryLang(langs, defaultLang), col.Config.Versioning.LastVersion)
				col.NavTree = col.CompositeNavTrees[key]
			}
			continue
		}

		if col.IsTabbed && len(col.Tabs) > 0 {
			// Fallback copies of untranslated _index.md files give each
			// language's tabs their labels.
			col.Tabs, col.LangTabs = BuildTabsI18n(col, colPages, contentDir, langs, defaultLang)
		} else {
			col.NavTrees = make(map[string]*engine.NavTree)
			for _, lang := range langs {
				langPages := filterByLang(colPages, lang)
				langCol := &engine.Collection{
					Name:     col.Name,
					Config:   col.Config,
					Pages:    langPages,
					Sections: BuildSectionTree(langPages, col.Name),
				}
				tree := navigation.BuildNavTree(langCol)
				col.NavTrees[lang] = tree
				navigation.WirePrevNextFromTree(tree)
			}
			col.NavTree = col.NavTrees[primaryLang(langs, defaultLang)]
		}
	}
}

// primaryLang returns the language whose tree fills a collection's
// language-free NavTree: defaultLang when langs has it, else the first of
// langs, else "".
func primaryLang(langs []string, defaultLang string) string {
	for _, lang := range langs {
		if lang == defaultLang {
			return lang
		}
	}
	if len(langs) > 0 {
		return langs[0]
	}
	return ""
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
