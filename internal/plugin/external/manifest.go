// Package external loads declarative disk-based plugins from
// {project}/plugins/{slug}/. Each plugin ships a plugin.yaml manifest
// describing conditional asset injection, output copying, and optional
// template contributions (partials, components, shortcodes). No code
// execution is involved: the engine synthesizes hook functions from the
// manifest and registers them with the regular plugin manager.
package external

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
)

// Manifest is the parsed plugin.yaml of one external plugin.
type Manifest struct {
	Name        string       `yaml:"name"`
	Slug        string       `yaml:"slug"`
	Version     string       `yaml:"version"`
	Description string       `yaml:"description"`
	Author      string       `yaml:"author"`
	Homepage    string       `yaml:"homepage"`
	Premium     bool         `yaml:"premium"`
	PurchaseURL string       `yaml:"purchase_url"`
	Inject      InjectConfig `yaml:"inject"`
	Output      OutputConfig `yaml:"output"`
}

// InjectConfig declares conditional per-page asset injection.
type InjectConfig struct {
	When          string   `yaml:"when"`
	Layout        string   `yaml:"layout"`
	Collection    string   `yaml:"collection"`
	Styles        []string `yaml:"styles"`
	Scripts       []string `yaml:"scripts"`
	ModuleScripts []string `yaml:"module_scripts"`
}

// HasAssets reports whether the inject block references any asset files.
func (i *InjectConfig) HasAssets() bool {
	return len(i.Styles)+len(i.Scripts)+len(i.ModuleScripts) > 0
}

// OutputConfig declares which files under assets/ are copied to dist and where.
type OutputConfig struct {
	Prefix  string   `yaml:"prefix"`
	Include []string `yaml:"include"`
}

// injectWhenRules is the set of valid inject.when values. The parameterless
// rules share the vocabulary of plugin.MatchesInjectRule; layout and
// collection additionally require their matching manifest field.
var injectWhenRules = map[string]bool{
	"always": true, "layout": true, "collection": true,
	"has_sidebar": true, "has_toc": true, "has_headings": true,
	"has_code_blocks": true, "has_images": true, "has_prev_next": true,
	"is_content_page": true, "has_updated": true,
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9_-]*[a-z0-9])?$`)

// ValidSlug reports whether s is a safe single-path-element plugin slug.
func ValidSlug(s string) bool {
	return slugPattern.MatchString(s)
}

// LoadManifest reads and parses {dir}/plugin.yaml. Unknown fields are
// rejected so manifest typos surface as errors instead of silent no-ops.
func LoadManifest(dir string) (*Manifest, error) {
	path := filepath.Join(dir, consts.FilePluginManifest)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", consts.FilePluginManifest, err)
	}
	var m Manifest
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", consts.FilePluginManifest, err)
	}
	return &m, nil
}

// Validate checks required fields, the slug/directory-name match, inject
// rule consistency, and rejects path escapes in asset and output paths.
func (m *Manifest) Validate(dirName string) error {
	if m.Name == "" {
		return fmt.Errorf("manifest is missing required field 'name'")
	}
	if m.Slug == "" {
		return fmt.Errorf("manifest is missing required field 'slug'")
	}
	if !ValidSlug(m.Slug) {
		return fmt.Errorf("invalid slug %q: use lowercase letters, digits, '-' and '_'", m.Slug)
	}
	if m.Slug != dirName {
		return fmt.Errorf("slug %q does not match plugin directory name %q", m.Slug, dirName)
	}
	if m.Version == "" {
		return fmt.Errorf("manifest is missing required field 'version'")
	}

	when := m.Inject.When
	if when == "" && m.Inject.HasAssets() {
		when = "always"
		m.Inject.When = when
	}
	if when != "" {
		if !injectWhenRules[when] {
			return fmt.Errorf("unknown inject.when rule %q", when)
		}
		switch when {
		case "layout":
			if m.Inject.Layout == "" {
				return fmt.Errorf("inject.when 'layout' requires inject.layout")
			}
			if !engine.ValidateLayout(engine.LayoutType(m.Inject.Layout)) {
				return fmt.Errorf("unknown inject.layout %q", m.Inject.Layout)
			}
		case "collection":
			if m.Inject.Collection == "" {
				return fmt.Errorf("inject.when 'collection' requires inject.collection")
			}
		}
	}

	for _, group := range [][]string{m.Inject.Styles, m.Inject.Scripts, m.Inject.ModuleScripts} {
		for _, p := range group {
			if err := validateRelPath(p); err != nil {
				return fmt.Errorf("inject asset path %q: %w", p, err)
			}
		}
	}
	if m.Output.Prefix != "" {
		if err := validateRelPath(strings.TrimSuffix(m.Output.Prefix, "/")); err != nil {
			return fmt.Errorf("output.prefix %q: %w", m.Output.Prefix, err)
		}
	}
	for _, p := range m.Output.Include {
		if err := validateRelPath(strings.TrimSuffix(p, "/")); err != nil {
			return fmt.Errorf("output.include entry %q: %w", p, err)
		}
	}
	return nil
}

// EffectivePrefix returns the dist-relative output prefix, always without a
// leading slash and with a trailing slash. Defaults to assets/vendor/{slug}/.
func (m *Manifest) EffectivePrefix() string {
	prefix := m.Output.Prefix
	if prefix == "" {
		prefix = "assets/vendor/" + m.Slug + "/"
	}
	prefix = strings.TrimPrefix(prefix, "/")
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix
}

// IncludeFilter returns a filter for WriteFSTreeFiltered based on
// output.include, or nil when the whole assets/ tree should be copied.
// Entries ending in "/" match directory prefixes; other entries match the
// exact file or a directory of that name.
func (m *Manifest) IncludeFilter() func(rel string) bool {
	if len(m.Output.Include) == 0 {
		return nil
	}
	entries := make([]string, 0, len(m.Output.Include))
	for _, e := range m.Output.Include {
		entries = append(entries, strings.TrimPrefix(e, "./"))
	}
	return func(rel string) bool {
		for _, e := range entries {
			if strings.HasSuffix(e, "/") {
				if strings.HasPrefix(rel, e) {
					return true
				}
				continue
			}
			if rel == e || strings.HasPrefix(rel, e+"/") {
				return true
			}
		}
		return false
	}
}

// validateRelPath rejects absolute paths, backslashes, and any ".." segment.
func validateRelPath(p string) error {
	if p == "" {
		return fmt.Errorf("path is empty")
	}
	if strings.Contains(p, "\\") {
		return fmt.Errorf("path must use forward slashes")
	}
	if strings.HasPrefix(p, "/") {
		return fmt.Errorf("path must be relative")
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return fmt.Errorf("path must not contain '..'")
		}
	}
	return nil
}
