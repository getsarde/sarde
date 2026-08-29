// Package catalog assembles the machine-readable plugin catalog behind
// `sarde plugins --format json`: every plugin name the engine accepts in
// plugins.enabled, with metadata and configurable fields.
//
// It lives apart from internal/build so that the CLI can describe plugins
// without pulling in the builder, and apart from internal/plugin because the
// subpackage plugins cannot be imported from there (see plugin/registry.go).
package catalog

import (
	"path/filepath"
	"sort"

	"github.com/getsarde/sarde/internal/plugin/cfgutil"
	"github.com/getsarde/sarde/internal/plugin/clientplugins"
	"github.com/getsarde/sarde/internal/plugin/external"
	"github.com/getsarde/sarde/internal/plugin/serverplugins"
)

// Version is bumped when the JSON shape changes incompatibly.
const Version = 1

// Kind classifies where a plugin is implemented.
const (
	KindServer   = "server"
	KindClient   = "client"
	KindExternal = "external"
)

// Entry describes one plugin. Keep field names stable: Sarde Studio decodes
// them.
type Entry struct {
	ID             string          `json:"id"`
	Label          string          `json:"label"`
	Kind           string          `json:"kind"`
	Group          string          `json:"group"`
	Description    string          `json:"description"`
	DefaultEnabled bool            `json:"defaultEnabled"`
	ConfigKey      string          `json:"configKey,omitempty"`
	Fields         []cfgutil.Field `json:"fields"`
}

// Catalog is the full listing.
type Catalog struct {
	Version int     `json:"version"`
	Plugins []Entry `json:"plugins"`
}

// Build lists built-in server and client plugins plus, when projectDir is
// non-empty, external plugins discovered under {projectDir}/plugins.
// External plugins are enabled by presence, so they report defaultEnabled.
func Build(projectDir string) Catalog {
	var entries []Entry

	for _, slug := range serverplugins.Slugs() {
		e, _ := serverplugins.Entry(slug)
		entries = append(entries, Entry{
			ID: slug, Label: e.Label, Kind: KindServer, Group: e.Group, Description: e.Description,
			DefaultEnabled: e.DefaultEnabled, ConfigKey: e.ConfigKey,
			Fields: nonNil(serverplugins.Fields(slug)),
		})
	}

	_ = clientplugins.Initialize()
	for _, slug := range clientplugins.PluginSlugs() {
		entries = append(entries, Entry{
			ID: slug, Label: clientplugins.Label(slug), Kind: KindClient, Group: KindClient,
			Description: clientplugins.Description(slug),
			Fields:      nonNil(clientplugins.Fields(slug)),
		})
	}

	if projectDir != "" {
		for _, dir := range external.DiscoverDirs(projectDir) {
			slug := filepath.Base(dir)
			entry := Entry{ID: slug, Label: slug, Kind: KindExternal, Group: KindExternal, DefaultEnabled: true, Fields: []cfgutil.Field{}}
			if m, err := external.LoadManifest(dir); err == nil {
				entry.Description = m.Description
				entry.Label = m.Name
			}
			if fields, err := external.LoadBlueprint(dir); err == nil {
				entry.Fields = nonNil(fields)
			}
			entries = append(entries, entry)
		}
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return kindRank(entries[i].Kind) < kindRank(entries[j].Kind)
		}
		return entries[i].ID < entries[j].ID
	})
	return Catalog{Version: Version, Plugins: entries}
}

// IDs returns every plugin name in the catalog.
func (c Catalog) IDs() []string {
	ids := make([]string, 0, len(c.Plugins))
	for _, e := range c.Plugins {
		ids = append(ids, e.ID)
	}
	return ids
}

func kindRank(k string) int {
	switch k {
	case KindServer:
		return 0
	case KindClient:
		return 1
	default:
		return 2
	}
}

func nonNil(f []cfgutil.Field) []cfgutil.Field {
	if f == nil {
		return []cfgutil.Field{}
	}
	return f
}
