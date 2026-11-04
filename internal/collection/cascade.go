package collection

import (
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
)

// ApplyCascade resolves frontmatter inheritance from section _index.md pages.
// Each section's `cascade` map is applied to all descendant pages that don't
// explicitly set those fields. Must be called after BuildSectionTree.
func ApplyCascade(pages []*engine.Page) {
	for _, page := range pages {
		if page.Section == nil {
			continue
		}
		merged := collectCascade(page)
		if len(merged) == 0 {
			continue
		}
		if page.Params == nil {
			page.Params = make(map[string]any)
		}
		for key, value := range merged {
			applyCascadeField(page, key, value)
		}
	}
}

// ApplyCascadeByLane applies cascade to targets from the section indexes of
// each target's own lane, its (language, version) pair: a French page inherits
// from content/fr/... _index.md files, a v1 page from the v1 ones. The shared
// section tree keys folders by the language- and version-free path, so the
// _index.md files of every lane fight for one slot there; each lane gets its
// own tree instead. A lane in another language than defaultLang borrows the
// default language's section index (same version) for a folder it has no
// _index.md of its own, as the i18n fallback does. Lanes never borrow across
// versions.
//
// pages are the collection's pages and targets the pages to fill, which must
// be among pages. Every page's Section is restored afterwards, so the shared
// tree and the pages' links into it are untouched. Pages without a FilePath
// (synthetic indexes, plugin pages) take no part. defaultLang is "" on a
// single-language site.
func ApplyCascadeByLane(pages, targets []*engine.Page, collectionName, defaultLang string) {
	type lane struct{ lang, version string }
	members := make(map[lane][]*engine.Page)
	for _, p := range pages {
		if p.FilePath == "" {
			continue
		}
		k := lane{p.Lang, p.Version}
		members[k] = append(members[k], p)
	}
	var order []lane
	laneTargets := make(map[lane][]*engine.Page)
	for _, p := range targets {
		if p.FilePath == "" {
			continue
		}
		k := lane{p.Lang, p.Version}
		if _, seen := laneTargets[k]; !seen {
			order = append(order, k)
		}
		laneTargets[k] = append(laneTargets[k], p)
	}
	if len(order) == 0 {
		return
	}

	saved := make(map[*engine.Page]*engine.Section, len(pages)+len(targets))
	for _, p := range pages {
		saved[p] = p.Section
	}
	for _, p := range targets {
		saved[p] = p.Section
	}
	defer func() {
		for p, sec := range saved {
			p.Section = sec
		}
	}()

	for _, k := range order {
		tree := members[k]
		if defaultLang != "" && k.lang != defaultLang {
			tree = withBorrowedSectionIndexes(tree, members[lane{defaultLang, k.version}], collectionName)
		}
		BuildSectionTree(tree, collectionName)
		ApplyCascade(laneTargets[k])
	}
}

// withBorrowedSectionIndexes returns lanePages plus the section index pages of
// defaultPages whose folder has no section index in lanePages.
func withBorrowedSectionIndexes(lanePages, defaultPages []*engine.Page, collectionName string) []*engine.Page {
	own := make(map[string]bool)
	for _, p := range lanePages {
		if p.Kind == engine.KindSection {
			own[sectionDir(p.RelPermalink, collectionName)] = true
		}
	}
	result := lanePages
	for _, p := range defaultPages {
		if p.Kind == engine.KindSection && !own[sectionDir(p.RelPermalink, collectionName)] {
			if len(result) == len(lanePages) {
				result = append([]*engine.Page(nil), lanePages...)
			}
			result = append(result, p)
		}
	}
	return result
}

// ApplyPageDefaults fills the values a build gives a page beyond its own
// frontmatter: the cascade from its lane's section indexes, then the slides
// and labs layout defaults, which run after cascade so explicit frontmatter
// and cascade both win over them. pages are the collection's pages and
// targets the pages to fill, a subset of pages: the full build passes every
// page, an incremental rebuild the one page it re-parsed, so the two cannot
// drift apart.
func ApplyPageDefaults(collectionName string, pages, targets []*engine.Page, defaultLang string) {
	ApplyCascadeByLane(pages, targets, collectionName, defaultLang)
	if IsSlidesName(collectionName) {
		applySlidesLayoutDefault(targets)
	}
	if IsLabsName(collectionName) {
		// Multi-course detection reads every section page: on a one-page
		// slice a multi-course collection reads as a two-level one.
		applyLabsLayoutDefaultWith(targets, detectMultiCourseFromPages(pages))
	}
}

// collectCascade walks the section ancestry from root to the page's direct
// parent, merging cascade maps. Inner sections override outer ones.
func collectCascade(page *engine.Page) map[string]any {
	var ancestors []*engine.Section
	sec := page.Section
	// For section _index.md pages, start from their parent (don't cascade to yourself).
	if page.Kind == engine.KindSection && sec.Parent != nil {
		sec = sec.Parent
	}
	for sec != nil {
		ancestors = append(ancestors, sec)
		sec = sec.Parent
	}
	if len(ancestors) == 0 {
		return nil
	}

	merged := make(map[string]any)
	// Apply outermost first so inner sections override outer.
	for i := len(ancestors) - 1; i >= 0; i-- {
		idx := ancestors[i].IndexPage
		if idx == nil || idx.Params == nil {
			continue
		}
		cascade, ok := idx.Params[consts.CascadeKey].(map[string]any)
		if !ok || len(cascade) == 0 {
			continue
		}
		for k, v := range cascade {
			merged[k] = v
		}
	}
	return merged
}

// applyCascadeField sets a single cascade value on a page, only if the page
// hasn't explicitly set that field itself.
func applyCascadeField(page *engine.Page, key string, value any) {
	switch key {
	case "layout", "template", "type",
		"pagefind", "icon", "edit_url", "show_updated":
		if _, exists := page.Params[key]; !exists {
			page.Params[key] = value
		}

	case "sidebar":
		if subMap, ok := value.(map[string]any); ok {
			if n, ok := subMap["order"].(int); ok && page.Sidebar.Order == 0 {
				page.Sidebar.Order = n
			}
			if s, ok := subMap["label"].(string); ok && page.Sidebar.Label == "" {
				page.Sidebar.Label = s
			}
			if b, ok := subMap["hidden"].(bool); ok && !page.Sidebar.Hidden {
				page.Sidebar.Hidden = b
			}
		}

	case "toc":
		if subMap, ok := value.(map[string]any); ok {
			if b, ok := subMap["enabled"].(bool); ok && page.TOC.Enabled == nil {
				page.TOC.Enabled = &b
			}
			if n, ok := subMap["min_level"].(int); ok && page.TOC.MinLevel == 0 {
				page.TOC.MinLevel = n
			}
			if n, ok := subMap["max_level"].(int); ok && page.TOC.MaxLevel == 0 {
				page.TOC.MaxLevel = n
			}
		}

	case "banner":
		if _, exists := page.Params["banner"]; !exists {
			page.Params["banner"] = mapToPageBanner(value)
		}

	case "head":
		if _, exists := page.Params["head"]; !exists {
			page.Params["head"] = value
		}

	case "params":
		if subMap, ok := value.(map[string]any); ok {
			for k, v := range subMap {
				if _, exists := page.Params[k]; !exists {
					page.Params[k] = v
				}
			}
		}

	case "draft", "cascade":
		// draft: filtering already happened before cascade runs
		// cascade: don't cascade the cascade key itself

	default:
		if _, exists := page.Params[key]; !exists {
			page.Params[key] = value
		}
	}
}

func mapToPageBanner(v any) *engine.PageBanner {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	b := &engine.PageBanner{}
	if s, ok := m["content"].(string); ok {
		b.Content = s
	}
	if s, ok := m["variant"].(string); ok {
		b.Variant = s
	}
	if s, ok := m["icon"].(string); ok {
		b.Icon = s
	}
	if b.Content == "" {
		return nil
	}
	return b
}
