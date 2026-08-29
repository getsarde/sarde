package serverplugins

import (
	"sort"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/plugin"
)

func TestInitialize_EveryEntryHasDefaults(t *testing.T) {
	if err := Initialize(); err != nil {
		t.Fatal(err)
	}
	if len(Slugs()) < 14 {
		t.Fatalf("expected at least 14 server plugins, got %v", Slugs())
	}
	for _, slug := range Slugs() {
		e, _ := Entry(slug)
		if e.Label == "" || e.Description == "" || e.Group == "" {
			t.Errorf("%s: missing description/group", slug)
		}
		if e.ConfigKey != "" && len(Fields(slug)) != 0 {
			t.Errorf("%s: config_key plugins must not declare fields", slug)
		}
		if Fields(slug) == nil {
			t.Errorf("%s: Fields returned nil (defaults file missing?)", slug)
		}
	}
}

// Every Go builtin is described by the manifest so the catalog cannot drift
// from the registry (subpackage plugins are asserted from internal/build).
func TestManifest_CoversBuiltinRegistry(t *testing.T) {
	for _, name := range plugin.BuiltinNames() {
		if _, ok := Entry(name); !ok {
			t.Errorf("builtin %q missing from serverplugins/manifest.yaml", name)
		}
	}
}

// default_enabled must agree with plugins.enabled in the embedded default
// sarde.yaml, which is what a new site actually gets.
func TestManifest_DefaultEnabledMatchesEmbeddedConfig(t *testing.T) {
	var doc struct {
		Plugins struct {
			Enabled []string `yaml:"enabled"`
		} `yaml:"plugins"`
	}
	if err := yaml.Unmarshal(embedded.DefaultsYAML, &doc); err != nil {
		t.Fatal(err)
	}
	want := append([]string(nil), doc.Plugins.Enabled...)
	sort.Strings(want)
	var got []string
	for _, slug := range Slugs() {
		if e, _ := Entry(slug); e.DefaultEnabled {
			got = append(got, slug)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("default_enabled set %v != embedded defaults %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("default_enabled set %v != embedded defaults %v", got, want)
		}
	}
}
