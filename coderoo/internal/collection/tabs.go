package collection

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coderoo-dev/coderoo/internal/consts"
	"github.com/coderoo-dev/coderoo/internal/engine"
	"github.com/coderoo-dev/coderoo/internal/navigation"
)

// DetectTabs determines whether a collection should use tabbed docs mode.
// Returns true when: explicit config forces it, or auto-detection finds
// 2+ top-level sections all with _index.md and no loose root pages.
func DetectTabs(col *engine.Collection) bool {
	if col == nil || col.Config == nil {
		return false
	}
	// Explicit opt-out
	if col.Config.Tabs != nil && !*col.Config.Tabs {
		return false
	}
	// Explicit opt-in
	if col.Config.Tabs != nil && *col.Config.Tabs {
		return true
	}
	// Auto-detect only for sidebar layouts
	if !engine.LayoutHasSidebar(col.Config.Layout) {
		return false
	}
	// Check frontmatter opt-out on root _index.md
	if col.IndexPage != nil && col.IndexPage.Params != nil {
		if v, ok := col.IndexPage.Params["tabs"]; ok {
			if b, ok := v.(bool); ok && !b {
				return false
			}
		}
	}
	// Structural check: find top-level sections (direct children of root)
	topSections := topLevelSections(col)
	if len(topSections) < 2 {
		return false
	}
	for _, sec := range topSections {
		if sec.IndexPage == nil {
			return false
		}
	}
	// No loose root-level pages (pages not in any section, or in the root section)
	for _, p := range col.Pages {
		if p.Kind == engine.KindSection {
			continue
		}
		if p.Section == nil {
			return false
		}
		// Pages in the root section (the collection's own _index.md section) are loose
		if isRootSection(p.Section, col) {
			return false
		}
	}
	return true
}

// BuildTabs creates DocsTab entries for a tabbed collection (single language).
func BuildTabs(col *engine.Collection, contentDir string) []*engine.DocsTab {
	topSections := topLevelSections(col)
	tabs := make([]*engine.DocsTab, 0, len(topSections))

	for _, sec := range topSections {
		tab := buildTab(sec, col, contentDir)
		tabs = append(tabs, tab)
	}

	sortTabs(tabs)
	return tabs
}

// BuildTabsI18n creates DocsTab entries with per-language nav trees.
func BuildTabsI18n(col *engine.Collection, contentDir string, langs []string) []*engine.DocsTab {
	topSections := topLevelSections(col)
	tabs := make([]*engine.DocsTab, 0, len(topSections))

	for _, sec := range topSections {
		tabPages := pagesForSection(sec, col)
		tab := &engine.DocsTab{
			Title:       sec.IndexPage.Title,
			Description: sec.IndexPage.Description,
			Slug:        sec.Slug,
			Weight:      sec.IndexPage.Weight,
			Permalink:   sec.Permalink,
			Section:     sec,
			IndexPage:   sec.IndexPage,
			Pages:       tabPages,
			NavTrees:    make(map[string]*engine.NavTree),
		}
		if icon, ok := sec.IndexPage.Params["icon"].(string); ok {
			tab.Icon = icon
		}

		for _, lang := range langs {
			langPages := filterTabPagesByLang(tabPages, lang)
			langSections := BuildSectionTree(langPages, col.Name)
			tabCol := &engine.Collection{
				Name:     col.Name,
				Title:    tab.Title,
				Config:   col.Config,
				Pages:    langPages,
				Sections: langSections,
			}
			tree := buildTabNavTree(tabCol, col.Name, sec.Slug, contentDir)
			tab.NavTrees[lang] = tree
			navigation.WirePrevNextFromTree(tree)
		}
		// Default nav tree for backward compat
		if len(langs) > 0 {
			tab.NavTree = tab.NavTrees[langs[0]]
		}

		tabs = append(tabs, tab)
	}

	sortTabs(tabs)
	return tabs
}

// FindTabForPage returns the tab that contains the given page, or nil.
func FindTabForPage(tabs []*engine.DocsTab, page *engine.Page) *engine.DocsTab {
	if page == nil || len(tabs) == 0 {
		return nil
	}
	for _, tab := range tabs {
		if strings.HasPrefix(page.RelPermalink, tab.Permalink) {
			return tab
		}
	}
	// Fallback: walk section parents
	sec := page.Section
	for sec != nil {
		for _, tab := range tabs {
			if sec == tab.Section {
				return tab
			}
		}
		sec = sec.Parent
	}
	return nil
}

// buildTab creates a single DocsTab with its nav tree (single language).
func buildTab(sec *engine.Section, col *engine.Collection, contentDir string) *engine.DocsTab {
	tabPages := pagesForSection(sec, col)
	tab := &engine.DocsTab{
		Title:       sec.IndexPage.Title,
		Description: sec.IndexPage.Description,
		Slug:        sec.Slug,
		Weight:      sec.IndexPage.Weight,
		Permalink:   sec.Permalink,
		Section:     sec,
		IndexPage:   sec.IndexPage,
		Pages:       tabPages,
	}
	if icon, ok := sec.IndexPage.Params["icon"].(string); ok {
		tab.Icon = icon
	}

	// Build a temporary collection scoped to this tab's pages
	tabCol := &engine.Collection{
		Name:     col.Name,
		Title:    tab.Title,
		Config:   col.Config,
		Pages:    tabPages,
		Sections: sec.Sections,
	}

	tab.NavTree = buildTabNavTree(tabCol, col.Name, sec.Slug, contentDir)
	navigation.WirePrevNextFromTree(tab.NavTree)

	return tab
}

// buildTabNavTree checks for a per-tab nav.yaml, falling back to auto-generation.
func buildTabNavTree(tabCol *engine.Collection, colName, tabSlug, contentDir string) *engine.NavTree {
	navPath := filepath.Join(contentDir, colName, tabSlug, consts.FileNavConfig)
	if _, err := os.Stat(navPath); err == nil {
		tree, err := navigation.BuildNavTreeFromYAML(navPath, tabCol)
		if err == nil {
			return tree
		}
		// Fall through to auto-generation on error
	}
	return navigation.BuildNavTree(tabCol)
}

// topLevelSections returns sections that are direct children of the root section,
// or have no parent if no root _index.md exists.
// When versioning is enabled, sections matching version IDs are excluded so they
// aren't misdetected as tabs.
func topLevelSections(col *engine.Collection) []*engine.Section {
	// Find root section (the one representing the collection root directory)
	var rootSection *engine.Section
	for _, sec := range col.Sections {
		if sec.Parent == nil && sec.Permalink == "/"+col.Name+"/" {
			rootSection = sec
			break
		}
	}

	var raw []*engine.Section
	if rootSection != nil {
		raw = rootSection.Sections
	} else {
		for _, sec := range col.Sections {
			if sec.Parent == nil {
				raw = append(raw, sec)
			}
		}
	}

	// Filter out version directories when versioning is active.
	if col.Config != nil && col.Config.Versioning != nil && col.Config.Versioning.Enabled {
		versionIDs := make(map[string]bool, len(col.Config.Versioning.Versions))
		for _, v := range col.Config.Versioning.Versions {
			versionIDs[v.ID] = true
		}
		filtered := make([]*engine.Section, 0, len(raw))
		for _, sec := range raw {
			if !versionIDs[sec.Slug] {
				filtered = append(filtered, sec)
			}
		}
		return filtered
	}

	return raw
}

// isRootSection returns true if the section is the collection's root section.
func isRootSection(sec *engine.Section, col *engine.Collection) bool {
	return sec.Permalink == "/"+col.Name+"/"
}

// pagesForSection collects all pages that belong to a section or its descendants.
func pagesForSection(sec *engine.Section, col *engine.Collection) []*engine.Page {
	prefix := sec.Permalink
	var pages []*engine.Page
	for _, p := range col.Pages {
		if strings.HasPrefix(p.RelPermalink, prefix) {
			pages = append(pages, p)
		}
	}
	return pages
}

// filterTabPagesByLang returns pages matching the given language code.
func filterTabPagesByLang(pages []*engine.Page, lang string) []*engine.Page {
	var result []*engine.Page
	for _, p := range pages {
		if p.Lang == lang {
			result = append(result, p)
		}
	}
	return result
}

// sortTabs sorts tabs by weight ascending, then title alphabetically.
func sortTabs(tabs []*engine.DocsTab) {
	sort.SliceStable(tabs, func(i, j int) bool {
		if tabs[i].Weight != tabs[j].Weight {
			return tabs[i].Weight < tabs[j].Weight
		}
		return strings.ToLower(tabs[i].Title) < strings.ToLower(tabs[j].Title)
	})
}
