package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/navigation"
)

// CollectSidebarOverrideWarnings scans every collection's resolved sidebar
// config for sidebar.yaml override and tab keys that no lane's tree-building
// pass consulted, returning one warning per unmatched key.
//
// Call exactly once per build, after every nav tree has been built, including
// the i18n-fallback rebuild (RebuildNavTreesWithFallbacks). Computing this
// earlier would false-positive on keys only matched in fallback-generated lanes.
func CollectSidebarOverrideWarnings(collections map[string]*engine.Collection) []engine.ValidationWarning {
	names := make([]string, 0, len(collections))
	for name := range collections {
		names = append(names, name)
	}
	sort.Strings(names)

	var warnings []engine.ValidationWarning
	for _, name := range names {
		col := collections[name]
		if col == nil || col.Config == nil || col.Config.Sidebar == nil {
			continue
		}
		sb := col.Config.Sidebar
		for _, key := range sb.UnmatchedOverrideKeys() {
			warnings = append(warnings, engine.ValidationWarning{
				File:    consts.FileSidebarConfig,
				Field:   name + "." + key,
				Message: fmt.Sprintf("override key %q matched no section or page in collection %q", key, name),
				Level:   "warning",
			})
		}
		for _, key := range sb.UnmatchedTabKeys() {
			warnings = append(warnings, engine.ValidationWarning{
				File:    consts.FileSidebarConfig,
				Field:   name + ".tabs." + key,
				Message: fmt.Sprintf("tab override key %q matched no tab in collection %q", key, name),
				Level:   "warning",
			})
		}
	}
	return warnings
}

// CollectNavYAMLWarnings reports the problems of every per-tab nav.yaml in a
// tabbed collection: a file that does not parse, which leaves the tab with
// its auto-generated sidebar, and a page reference that matches no page,
// whose entry and nested items are left out of the sidebar. References are
// checked against the collection's pages in every language and version, so
// one that resolves in any lane is not reported.
//
// Like CollectSidebarOverrideWarnings, call it once per build after the nav
// trees exist: the trees are built once per lane, so warning from inside
// tree building would repeat each warning per lane.
func CollectNavYAMLWarnings(collections map[string]*engine.Collection, contentDir string) []engine.ValidationWarning {
	names := make([]string, 0, len(collections))
	for name := range collections {
		names = append(names, name)
	}
	sort.Strings(names)

	var warnings []engine.ValidationWarning
	for _, name := range names {
		col := collections[name]
		if col == nil || !col.IsTabbed {
			continue
		}
		for _, slug := range tabSlugs(col) {
			rel := filepath.ToSlash(filepath.Join(name, slug, consts.FileNavConfig))
			navPath := filepath.Join(contentDir, name, slug, consts.FileNavConfig)
			if _, err := os.Stat(navPath); err != nil {
				if !os.IsNotExist(err) {
					warnings = append(warnings, engine.ValidationWarning{
						File:    rel,
						Message: fmt.Sprintf("reading nav.yaml: %v; tab %q uses the auto-generated sidebar", err, slug),
						Level:   "warning",
					})
				}
				continue
			}
			unresolved, err := navigation.CheckNavYAML(navPath, col)
			if err != nil {
				// YAML errors span several lines; keep the warning on one.
				msg := strings.Join(strings.Fields(err.Error()), " ")
				warnings = append(warnings, engine.ValidationWarning{
					File:    rel,
					Message: fmt.Sprintf("%s; tab %q uses the auto-generated sidebar", msg, slug),
					Level:   "warning",
				})
				continue
			}
			for _, ref := range unresolved {
				warnings = append(warnings, engine.ValidationWarning{
					File:    rel,
					Field:   "page",
					Message: fmt.Sprintf("page %q matches no page in collection %q; the entry and its items are left out of the sidebar", ref.Page, name),
					Level:   "warning",
					Line:    ref.Line,
					Col:     ref.Col,
				})
			}
		}
	}
	return warnings
}

// tabSlugs returns the slugs of every tab of a tabbed collection, across
// its plain tab list, its per-language lists and its per-language,
// per-version tab sets, sorted.
func tabSlugs(col *engine.Collection) []string {
	seen := make(map[string]bool)
	add := func(tabs []*engine.DocsTab) {
		for _, tab := range tabs {
			if tab != nil && tab.Slug != "" {
				seen[tab.Slug] = true
			}
		}
	}
	add(col.Tabs)
	for _, tabs := range col.LangTabs {
		add(tabs)
	}
	for _, tabs := range col.CompositeTabSets {
		add(tabs)
	}
	slugs := make([]string, 0, len(seen))
	for slug := range seen {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	return slugs
}
