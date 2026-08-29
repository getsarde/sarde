// Package serverplugins is the declarative registry of Sarde's built-in
// server-side plugins: manifest.yaml carries their metadata and
// defaults/<name>.yaml their configurable fields, in the same blueprint
// shape used by clientplugins and external plugin blueprints.
//
// The Go implementations live elsewhere (internal/plugin builtins and the
// subpackages registered in internal/build); this package only publishes
// what tooling such as `sarde plugins --format json` needs to describe them.
package serverplugins

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/getsarde/sarde/internal/plugin/cfgutil"
)

//go:embed manifest.yaml
var manifestData []byte

//go:embed defaults
var defaultsFS embed.FS

// ManifestEntry is one plugin's metadata from manifest.yaml.
type ManifestEntry struct {
	Label          string `yaml:"label"`
	Description    string `yaml:"description"`
	Group          string `yaml:"group"`
	DefaultEnabled bool   `yaml:"default_enabled"`
	ConfigKey      string `yaml:"config_key"`
}

type manifest struct {
	Plugins map[string]ManifestEntry `yaml:"plugins"`
}

var (
	initOnce sync.Once
	initErr  error
	entries  map[string]ManifestEntry
	raws     map[string][]byte
	fields   map[string][]cfgutil.Field
)

// Initialize parses the embedded manifest and defaults once.
func Initialize() error {
	initOnce.Do(func() {
		var m manifest
		if err := yaml.Unmarshal(manifestData, &m); err != nil {
			initErr = fmt.Errorf("serverplugins: bad manifest.yaml: %w", err)
			return
		}
		entries = m.Plugins
		raws = make(map[string][]byte, len(entries))
		fields = make(map[string][]cfgutil.Field, len(entries))
		for slug := range entries {
			data, err := fs.ReadFile(defaultsFS, "defaults/"+slug+".yaml")
			if err != nil {
				initErr = fmt.Errorf("serverplugins: %s has no defaults/%s.yaml", slug, slug)
				return
			}
			parsed, err := cfgutil.ParseFields(data)
			if err != nil {
				initErr = fmt.Errorf("serverplugins: defaults/%s.yaml: %w", slug, err)
				return
			}
			raws[slug] = data
			fields[slug] = parsed
		}
	})
	return initErr
}

// Slugs returns every manifest plugin name, sorted.
func Slugs() []string {
	_ = Initialize()
	out := make([]string, 0, len(entries))
	for slug := range entries {
		out = append(out, slug)
	}
	sort.Strings(out)
	return out
}

// Entry returns a plugin's manifest metadata.
func Entry(slug string) (ManifestEntry, bool) {
	_ = Initialize()
	e, ok := entries[slug]
	return e, ok
}

// Fields returns a plugin's declared config fields (sorted by name), or nil
// for an unknown plugin.
func Fields(slug string) []cfgutil.Field {
	_ = Initialize()
	return fields[slug]
}

// RawFields returns the raw defaults YAML for plugins that resolve their own
// config at build time (e.g. telescope), so the field list has one source.
func RawFields(slug string) []byte {
	_ = Initialize()
	return raws[slug]
}
