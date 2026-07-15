package collection

import (
	"fmt"
	"sort"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
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
