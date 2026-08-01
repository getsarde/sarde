package directive

import (
	"sort"

	"github.com/getsarde/sarde/internal/engine"
)

// MergeCatalog returns a new catalog combining the embedded built-in catalog
// with every registry's generic directives. Built-in entries are stamped
// Source "builtin"; generic directives land in one appended "custom"
// category. base is not mutated. Later registries win name conflicts among
// themselves; built-in name conflicts cannot occur because
// ValidateAgainstBuiltins removes them at load time.
func MergeCatalog(base *engine.DirectiveCatalog, regs ...*Registry) *engine.DirectiveCatalog {
	merged := &engine.DirectiveCatalog{}
	if base != nil {
		merged.Categories = make([]engine.DirectiveCategory, len(base.Categories))
		for i, c := range base.Categories {
			directives := make([]engine.CatalogDirective, len(c.Directives))
			for j, d := range c.Directives {
				d.Source = "builtin"
				directives[j] = d
			}
			merged.Categories[i] = engine.DirectiveCategory{
				Name:       c.Name,
				Label:      c.Label,
				Directives: directives,
			}
		}
	}

	byName := make(map[string]engine.CatalogDirective)
	for _, reg := range regs {
		if reg == nil {
			continue
		}
		for _, entry := range reg.CatalogEntries() {
			byName[entry.Name] = entry
		}
	}
	if len(byName) == 0 {
		return merged
	}

	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)

	custom := engine.DirectiveCategory{Name: "custom", Label: "Custom"}
	for _, name := range names {
		custom.Directives = append(custom.Directives, byName[name])
	}
	merged.Categories = append(merged.Categories, custom)
	return merged
}
