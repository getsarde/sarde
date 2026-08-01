package directive

import (
	"crypto/sha256"
	"fmt"
	htmltemplate "html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getsarde/sarde/internal/engine"
)

// Registry holds all loaded generic directive definitions. Load order gives
// overlay precedence: a directory loaded later overwrites same-named
// directives from earlier directories (plugins first, then theme, then
// site). The registry
// is immutable after loading and safe for concurrent read access.
type Registry struct {
	funcMap htmltemplate.FuncMap
	defs    map[string]*Def
}

// NewRegistry creates an empty Registry. funcMap is applied to every
// directive template (the shortcode FuncMap: icon, i18n, URL helpers).
func NewRegistry(funcMap htmltemplate.FuncMap) *Registry {
	return &Registry{funcMap: funcMap, defs: make(map[string]*Def)}
}

// LoadDir loads every <name>.yaml + <name>.html pair in dir, stamping source
// ("site" or "theme") on each definition. A missing dir is not an error.
// Invalid definitions are skipped with a warning; valid ones still load.
func (r *Registry) LoadDir(dir, source string) []engine.ValidationWarning {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var warnings []engine.ValidationWarning
	warn := func(name, msg string) {
		warnings = append(warnings, engine.ValidationWarning{
			File:    "directives/" + name + ".yaml",
			Field:   "directive",
			Message: msg,
			Level:   "warning",
		})
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		stem := strings.TrimSuffix(e.Name(), ".yaml")

		yamlRaw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			warn(stem, fmt.Sprintf("reading %s: %v", e.Name(), err))
			continue
		}
		htmlRaw, err := os.ReadFile(filepath.Join(dir, stem+".html"))
		if err != nil {
			warn(stem, fmt.Sprintf("missing sibling template %s.html (every directive needs one)", stem))
			continue
		}

		def, err := parseDef(stem, yamlRaw, htmlRaw, r.funcMap)
		if err != nil {
			warn(stem, err.Error())
			continue
		}
		def.Source = source

		// Optional CSS sidecar.
		if css, err := os.ReadFile(filepath.Join(dir, stem+".css")); err == nil {
			def.CSS = css
		}

		r.defs[def.Name] = def
	}
	return warnings
}

// Lookup returns the definition for name, or nil.
func (r *Registry) Lookup(name string) *Def { return r.defs[name] }

// Names returns all registered directive names, sorted.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.defs))
	for name := range r.defs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Empty reports whether the registry has no definitions.
func (r *Registry) Empty() bool { return len(r.defs) == 0 }

// CSS returns all directive CSS sidecars concatenated in name order, each
// prefixed with a header comment. Empty string when no directive ships CSS.
func (r *Registry) CSS() string {
	var b strings.Builder
	for _, name := range r.Names() {
		css := r.defs[name].CSS
		if len(css) == 0 {
			continue
		}
		fmt.Fprintf(&b, "/* directive: %s */\n", name)
		b.Write(css)
		b.WriteString("\n")
	}
	return b.String()
}

// Hash returns a sha256 hex digest over every definition's name, YAML,
// template, and CSS bytes, sorted by name. Folded into the markdown
// renderer's fingerprint so the page cache busts on any directive change.
// Empty string for an empty registry.
func (r *Registry) Hash() string {
	if len(r.defs) == 0 {
		return ""
	}
	h := sha256.New()
	for _, name := range r.Names() {
		def := r.defs[name]
		h.Write([]byte(name))
		h.Write(def.yamlRaw)
		h.Write(def.htmlRaw)
		h.Write(def.CSS)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ValidateAgainstBuiltins removes any directive whose name collides with a
// built-in catalog entry, returning one warning per removal. Built-ins always
// win: their bespoke parsers register ahead of the generic extension, so a
// colliding definition would silently never run.
func (r *Registry) ValidateAgainstBuiltins(cat *engine.DirectiveCatalog) []engine.ValidationWarning {
	if cat == nil {
		return nil
	}
	builtin := make(map[string]bool)
	for _, c := range cat.Categories {
		for _, d := range c.Directives {
			builtin[d.Name] = true
		}
	}

	var warnings []engine.ValidationWarning
	for _, name := range r.Names() {
		if !builtin[name] {
			continue
		}
		delete(r.defs, name)
		warnings = append(warnings, engine.ValidationWarning{
			File:    "directives/" + name + ".yaml",
			Field:   "directive",
			Message: fmt.Sprintf("directive %q collides with a built-in directive; the built-in wins and this definition is ignored", name),
			Level:   "warning",
		})
	}
	return warnings
}

// CatalogEntries converts every definition into its catalog form, sorted by
// name. Kind maps to the catalog's "block"; every field placement is "attr".
func (r *Registry) CatalogEntries() []engine.CatalogDirective {
	entries := make([]engine.CatalogDirective, 0, len(r.defs))
	for _, name := range r.Names() {
		def := r.defs[name]
		entries = append(entries, engine.CatalogDirective{
			Name:        def.Name,
			Label:       def.Label,
			Description: def.Description,
			Kind:        "block",
			Source:      def.Source,
			Bracket:     def.Bracket,
			Fields:      def.Fields,
		})
	}
	return entries
}
