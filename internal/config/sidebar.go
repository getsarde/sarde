package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

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
	Hidden      bool              `yaml:"hidden"`
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
// sarde.yaml). Returns (nil, nil) if the file is absent; the file is entirely
// optional. Decoding is always strict (unknown keys are hard errors): unlike
// sarde.yaml there is no legacy lenient mode to preserve for a brand-new file.
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
		return nil, fmt.Errorf("%s: %w", consts.FileSidebarConfig, err)
	}
	return sf, nil
}
