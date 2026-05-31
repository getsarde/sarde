package collection

import (
	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/navigation"
)

// LangVersionKey computes the composite map key for a (lang, version) pair.
func LangVersionKey(lang, ver string) string {
	if lang == "" {
		lang = "_default"
	}
	if ver == "" {
		ver = "_latest"
	}
	return lang + "/" + ver
}

// BuildCompositeNavTrees builds one NavTree per (lang, version) pair found in
// the collection's pages. Keyed by LangVersionKey. Used for versioned
// collections that may also be multi-language.
func BuildCompositeNavTrees(col *engine.Collection, langs []string) map[string]*engine.NavTree {
	vc := col.Config.Versioning
	if vc == nil || !vc.Enabled {
		return nil
	}

	// Discover all (lang, version) pairs present in pages.
	type lvPair struct{ lang, ver string }
	pairSet := make(map[lvPair]bool)
	for _, p := range col.Pages {
		pairSet[lvPair{p.Lang, p.Version}] = true
	}

	trees := make(map[string]*engine.NavTree, len(pairSet))
	for pair := range pairSet {
		filtered := filterByLangAndVersion(col.Pages, pair.lang, pair.ver)
		if len(filtered) == 0 {
			continue
		}

		sections := BuildSectionTree(filtered, col.Name)
		vCol := &engine.Collection{
			Name:     col.Name,
			Title:    col.Title,
			Config:   col.Config,
			Pages:    filtered,
			Sections: sections,
		}

		for _, p := range filtered {
			if p.Kind == engine.KindSection {
				relDir := sectionDir(p.RelPermalink, col.Name)
				if relDir == "" {
					vCol.IndexPage = p
					break
				}
			}
		}

		tree := navigation.BuildNavTree(vCol)
		navigation.WirePrevNextFromTree(tree)
		trees[LangVersionKey(pair.lang, pair.ver)] = tree
	}

	return trees
}

// LinkVersions groups pages across versions by their VersionRelPath and
// populates each page's VersionPeers slice, mirroring i18n.LinkTranslations.
func LinkVersions(pages []*engine.Page) {
	groups := make(map[string][]*engine.Page)
	for _, p := range pages {
		if p.Collection == nil || p.Collection.Config == nil ||
			p.Collection.Config.Versioning == nil || !p.Collection.Config.Versioning.Enabled {
			continue
		}
		if p.VersionRelPath == "" && p.Kind == engine.KindSection {
			continue
		}
		key := p.Collection.Name + ":" + p.VersionRelPath + ":" + p.Lang
		groups[key] = append(groups[key], p)
	}

	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		for _, p := range group {
			var peers []*engine.Page
			for _, other := range group {
				if other != p {
					peers = append(peers, other)
				}
			}
			p.VersionPeers = peers
		}
	}
}

func filterByLangAndVersion(pages []*engine.Page, lang, ver string) []*engine.Page {
	var result []*engine.Page
	for _, p := range pages {
		if p.Lang == lang && p.Version == ver {
			result = append(result, p)
		}
	}
	return result
}
