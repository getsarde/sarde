package collection

import (
	"strings"

	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/navigation"
)

// RebuildNavTreesWithFallbacks rebuilds per-language nav trees after fallback
// pages have been generated, so every language gets a complete sidebar
// (real translations + fallback pages from the default language).
func RebuildNavTreesWithFallbacks(collections map[string]*engine.Collection, allPages []*engine.Page, langs []string) {
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
				col.CompositeTabSets = BuildCompositeTabSets(col, "", langs)
			} else {
				col.CompositeNavTrees = BuildCompositeNavTrees(col, langs)
				if len(langs) > 0 {
					key := LangVersionKey(langs[0], col.Config.Versioning.LastVersion)
					col.NavTree = col.CompositeNavTrees[key]
				}
			}
			continue
		}

		if col.IsTabbed && len(col.Tabs) > 0 {
			for _, tab := range col.Tabs {
				prefix := tab.Permalink
				var tabPages []*engine.Page
				for _, p := range colPages {
					if strings.HasPrefix(p.RelPermalink, prefix) {
						tabPages = append(tabPages, p)
					}
				}
				tab.NavTrees = make(map[string]*engine.NavTree)
				for _, lang := range langs {
					langPages := filterByLang(tabPages, lang)
					langCol := &engine.Collection{
						Name:     col.Name,
						Config:   col.Config,
						Pages:    langPages,
						Sections: BuildSectionTree(langPages, col.Name),
					}
					tree := buildTabNavTree(langCol, col.Name, tab.Slug, "")
					tab.NavTrees[lang] = tree
					navigation.WirePrevNextFromTree(tree)
				}
				if len(langs) > 0 {
					tab.NavTree = tab.NavTrees[langs[0]]
				}
			}
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
			if len(langs) > 0 {
				col.NavTree = col.NavTrees[langs[0]]
			}
		}
	}
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
