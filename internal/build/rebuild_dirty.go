package build

import (
	"reflect"

	"github.com/getsarde/sarde/internal/collection"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/navigation"
	sardetemplate "github.com/getsarde/sarde/internal/template"
)

type changeClass int

const (
	changeIncremental      changeClass = iota
	changeCollectionScoped
	changeFullRebuild
)

func classifyChange(old, next *engine.Page, cf content.ContentFile) changeClass {
	if old.RelPermalink != next.RelPermalink {
		return changeFullRebuild
	}
	if old.Kind != next.Kind {
		return changeFullRebuild
	}
	if collectionName(old) != cf.CollectionName {
		return changeFullRebuild
	}
	if old.Lang != next.Lang || old.LangRelPath != next.LangRelPath {
		return changeFullRebuild
	}
	if old.Slug != next.Slug {
		return changeFullRebuild
	}

	sortBy := ""
	if old.Collection != nil && old.Collection.Config != nil {
		sortBy = old.Collection.Config.SortBy
	}
	switch sortBy {
	case "date":
		if !old.Date.Equal(next.Date) {
			return changeCollectionScoped
		}
	case "order":
		if old.Sidebar.Order != next.Sidebar.Order {
			return changeCollectionScoped
		}
	case "title":
		if old.Title != next.Title {
			return changeCollectionScoped
		}
	}

	if old.Title != next.Title {
		return changeCollectionScoped
	}

	if old.Sidebar.Label != next.Sidebar.Label || old.Sidebar.Hidden != next.Sidebar.Hidden ||
		old.Sidebar.Group != next.Sidebar.Group ||
		!reflect.DeepEqual(old.Sidebar.Badge, next.Sidebar.Badge) ||
		!reflect.DeepEqual(old.Sidebar.Attrs, next.Sidebar.Attrs) {
		return changeCollectionScoped
	}

	return changeIncremental
}

// collectionNeedsFullNavRebuild reports whether a collection-scoped nav change
// cannot be handled by the incremental rebuildCollectionNav and must fall back
// to a full build. rebuildCollectionNav only rewrites NavTree/NavTrees; it does
// not rebuild the tab or composite (versioned) nav structures the template
// renders from, and its per-language path mixes languages into each tree.
func (b *SiteBuilder) collectionNeedsFullNavRebuild(col *engine.Collection) bool {
	if b.config.I18n.IsMultiLang() {
		return true
	}
	if col == nil {
		return false
	}
	if col.IsTabbed || col.CompositeNavTrees != nil || col.CompositeTabSets != nil {
		return true
	}
	if col.Versioning != nil && col.Versioning.Enabled {
		return true
	}
	if col.Config != nil && col.Config.Versioning != nil && col.Config.Versioning.Enabled {
		return true
	}
	return false
}

func rebuildCollectionNav(col *engine.Collection) {
	if col == nil || col.Config == nil {
		return
	}
	collection.SortPages(col.Pages, col.Config.SortBy, col.Config.SortOrder)

	if engine.LayoutHasSidebar(col.Config.Layout) {
		col.NavTree = navigation.BuildNavTree(col)
		navigation.WirePrevNextFromTree(col.NavTree)
		for lang := range col.NavTrees {
			langCol := &engine.Collection{
				Name:     col.Name,
				Config:   col.Config,
				Sections: col.Sections,
			}
			for _, p := range col.Pages {
				if p.Lang == lang {
					langCol.Pages = append(langCol.Pages, p)
				}
			}
			col.NavTrees[lang] = navigation.BuildNavTree(langCol)
			navigation.WirePrevNextFromTree(col.NavTrees[lang])
		}
	} else {
		collection.WirePrevNext(col.Pages)
	}
}

// pageTerms returns the page's terms for the named taxonomy, matching the
// extraction taxonomy.BuildTaxonomies uses (built-in tags/categories fields,
// Extra for custom taxonomies such as authors or series).
func pageTerms(p *engine.Page, taxName string) []string {
	switch taxName {
	case "tags":
		return p.Tags
	case "categories":
		return p.Categories
	default:
		return p.Extra[taxName]
	}
}

// addRemovedTermsDirty marks the term pages (and their taxonomy index) of
// every term the change removed from the page, so those pages stop listing
// it. oldTaxonomies must be the last build's taxonomy set for the page's own
// language; terms are matched by slug, the key BuildTaxonomies stores under.
func addRemovedTermsDirty(old, next *engine.Page, oldTaxonomies map[string]*engine.Taxonomy, cfg map[string]config.TaxonomyConfig, dirty map[string]struct{}) {
	for taxName, tax := range oldTaxonomies {
		if !cfg[taxName].ShouldRender() {
			continue
		}
		kept := make(map[string]bool)
		for _, t := range pageTerms(next, taxName) {
			kept[content.Slugify(t)] = true
		}
		removedAny := false
		for _, t := range pageTerms(old, taxName) {
			slug := content.Slugify(t)
			if kept[slug] {
				continue
			}
			if term, ok := tax.Terms[slug]; ok {
				dirty[term.Permalink] = struct{}{}
				removedAny = true
			}
		}
		if removedAny {
			dirty[tax.Permalink] = struct{}{}
		}
	}
}

func addCollectionListDirty(col *engine.Collection, dirty map[string]struct{}) {
	if col == nil || col.IndexPage == nil {
		return
	}
	if col.IndexPage.RelPermalink != "" {
		dirty[col.IndexPage.RelPermalink] = struct{}{}
	}
	if col.Config == nil || col.Config.Paginate <= 0 {
		return
	}
	contentPages := 0
	for _, p := range col.Pages {
		if p.Kind != engine.KindSection {
			contentPages++
		}
	}
	total := (contentPages + col.Config.Paginate - 1) / col.Config.Paginate
	base := col.IndexPage.RelPermalink
	if base == "" {
		base = "/" + col.Name + "/"
	}
	for n := 2; n <= total; n++ {
		dirty[sardetemplate.PaginationURL(base, n)] = struct{}{}
	}
}

func addTaxonomyDirtyForPage(filePath string, taxonomies map[string]*engine.Taxonomy, cfg map[string]config.TaxonomyConfig, dirty map[string]struct{}) {
	for taxName, tax := range taxonomies {
		if !cfg[taxName].ShouldRender() {
			continue
		}
		paginateBy := tax.PaginateBy
		if paginateBy <= 0 {
			paginateBy = consts.DefaultPaginateBy
		}
		for _, term := range tax.Terms {
			found := false
			for _, tp := range term.Pages {
				if tp.FilePath == filePath {
					found = true
					break
				}
			}
			if !found {
				continue
			}
			dirty[tax.Permalink] = struct{}{}
			dirty[term.Permalink] = struct{}{}
			total := (len(term.Pages) + paginateBy - 1) / paginateBy
			for n := 2; n <= total; n++ {
				dirty[sardetemplate.PaginationURL(term.Permalink, n)] = struct{}{}
			}
		}
	}
}
