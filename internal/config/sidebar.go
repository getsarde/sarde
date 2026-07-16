package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
)

// SidebarFile is the parsed sidebar.yaml: a map from collection name to that
// collection's navigation overrides. nil means no sidebar.yaml was found and
// must behave identically to "file absent" (byte-identical-build invariant).
type SidebarFile map[string]*SidebarCollectionEntry

// SidebarCollectionEntry is one collection's block within sidebar.yaml.
type SidebarCollectionEntry struct {
	// CollapseLevel, when set, expands sidebar groups at depth <= N by
	// default and collapses deeper groups. Wins over the sarde.yaml
	// collections.{name}.sidebar.collapse_level value.
	CollapseLevel *int `yaml:"collapse_level"`

	// Tabs holds tab-bar property overrides keyed by tab slug
	// (tabbed docs collections only).
	Tabs map[string]*SidebarTabOverride `yaml:"tabs"`

	// Overrides holds node property overrides keyed by collection-relative
	// path (for example "guide/advanced"). Keys address sections and pages.
	Overrides map[string]*SidebarNodeOverride `yaml:"overrides"`

	// Items is the structural sidebar skeleton (Phase 2). Declared so strict
	// decoding accepts the key; Phase 1 warns and ignores it when non-empty.
	Items []SidebarItemEntry `yaml:"items"`
}

// SidebarTabOverride overrides tab-bar properties for one docs tab.
type SidebarTabOverride struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description"`
	Icon        string `yaml:"icon"`
	Order       *int   `yaml:"order"`
}

// SidebarNodeOverride overrides sidebar properties for one section or page.
type SidebarNodeOverride struct {
	Label       string            `yaml:"label"`
	Description string            `yaml:"description"`
	Order       *int              `yaml:"order"`
	Collapsed   *bool             `yaml:"collapsed"`
	Icon        string            `yaml:"icon"`
	Badge       engine.Badge      `yaml:"badge"` // scalar or {text, variant} form
	Hidden      *bool             `yaml:"hidden"` // nil = unset; false un-hides a frontmatter-hidden page
	Attrs       map[string]string `yaml:"attrs"`
}

// SidebarItemEntry is one entry of the Phase 2 structural skeleton. Stub in
// Phase 1: the full schema (page/url/autogenerate/items recursion) lands with
// the structural feature.
type SidebarItemEntry struct {
	Label        string             `yaml:"label"`
	Page         string             `yaml:"page"`
	URL          string             `yaml:"url"`
	External     bool               `yaml:"external"`
	Badge        engine.Badge       `yaml:"badge"`
	Collapsed    *bool              `yaml:"collapsed"`
	Attrs        map[string]string  `yaml:"attrs"`
	Autogenerate string             `yaml:"autogenerate"`
	Items        []SidebarItemEntry `yaml:"items"`
}

// LoadSidebarFile reads sidebar.yaml from dir (the directory containing
// sarde.yaml). Returns (nil, nil) if the file is absent or empty; the file is
// entirely optional. Decoding is always strict (unknown keys are hard errors):
// unlike sarde.yaml there is no legacy lenient mode to preserve for a
// brand-new file. Override and tab keys are canonicalized here; two raw keys
// that normalize to the same canonical key are a hard error, because letting
// map iteration order pick a winner would make builds nondeterministic.
func LoadSidebarFile(dir string) (SidebarFile, error) {
	path := filepath.Join(dir, consts.FileSidebarConfig)
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var sf SidebarFile
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(&sf); err != nil {
		// An empty or comments-only file has no YAML document: valid, no config.
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, fmt.Errorf("%s: %w", consts.FileSidebarConfig, err)
	}

	for collection, entry := range sf {
		if entry == nil {
			continue
		}
		if entry.Overrides, err = normalizeKeyedMap(collection, "overrides", entry.Overrides); err != nil {
			return nil, err
		}
		if entry.Tabs, err = normalizeKeyedMap(collection, "tabs", entry.Tabs); err != nil {
			return nil, err
		}
	}
	return sf, nil
}

// normalizeKeyedMap rebuilds m with canonical keys, erroring when two raw
// keys collapse into the same canonical key. Keys are visited in sorted order
// so the reported collision pair is deterministic.
func normalizeKeyedMap[V any](collection, section string, m map[string]V) (map[string]V, error) {
	if len(m) == 0 {
		return m, nil
	}
	raw := make([]string, 0, len(m))
	for key := range m {
		raw = append(raw, key)
	}
	sort.Strings(raw)

	out := make(map[string]V, len(m))
	seen := make(map[string]string, len(m))
	for _, key := range raw {
		canon := normalizeSidebarKey(key)
		if prev, ok := seen[canon]; ok {
			return nil, fmt.Errorf("%s: %s.%s: keys %q and %q both normalize to %q",
				consts.FileSidebarConfig, collection, section, prev, key, canon)
		}
		seen[canon] = key
		out[canon] = m[key]
	}
	return out, nil
}

// normalizeSidebarKey canonicalizes a sidebar.yaml path key: slashes are
// forward, no leading or trailing slash.
func normalizeSidebarKey(key string) string {
	key = strings.ReplaceAll(key, "\\", "/")
	return strings.Trim(key, "/")
}
