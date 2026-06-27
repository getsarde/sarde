package build

import (
	"reflect"
	"strings"

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

func addRemovedTermsDirty(old, next *engine.Page, oldTaxonomies map[string]*engine.Taxonomy, cfg map[string]config.TaxonomyConfig, dirty map[string]struct{}) {
	oldTerms := make(map[string]bool)
	newTerms := make(map[string]bool)
	for _, t := range old.Tags {
		oldTerms["tags:"+strings.ToLower(t)] = true
	}
	for _, c := range old.Categories {
		oldTerms["categories:"+strings.ToLower(c)] = true
	}
	for _, t := range next.Tags {
		newTerms["tags:"+strings.ToLower(t)] = true
	}
	for _, c := range next.Categories {
		newTerms["categories:"+strings.ToLower(c)] = true
	}
	for key := range oldTerms {
		if newTerms[key] {
			continue
		}
		parts := strings.SplitN(key, ":", 2)
		taxName, termSlug := parts[0], parts[1]
		tax, ok := oldTaxonomies[taxName]
		if !ok || !cfg[taxName].ShouldRender() {
			continue
		}
		dirty[tax.Permalink] = struct{}{}
		for _, term := range tax.Terms {
			if strings.ToLower(term.Name) == termSlug || strings.ToLower(term.Slug) == termSlug {
				dirty[term.Permalink] = struct{}{}
				break
			}
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
