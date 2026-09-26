package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuild_BuiltinsOnly(t *testing.T) {
	cat := Build("")
	if cat.Version != Version {
		t.Fatalf("version %d", cat.Version)
	}
	byID := map[string]Entry{}
	for _, e := range cat.Plugins {
		if _, dup := byID[e.ID]; dup {
			t.Errorf("duplicate id %q", e.ID)
		}
		byID[e.ID] = e
		if e.Fields == nil {
			t.Errorf("%s: fields must be [] not null", e.ID)
		}
		if e.Description == "" || e.Label == "" {
			t.Errorf("%s: empty label/description", e.ID)
		}
	}
	for _, want := range []string{"sitemap", "telescope", "social_cards", "scroll_to_top", "reading_progress"} {
		if _, ok := byID[want]; !ok {
			t.Errorf("missing %q", want)
		}
	}
	if byID["link_validator"].ConfigKey != "link_validation" {
		t.Errorf("link_validator configKey = %q", byID["link_validator"].ConfigKey)
	}
	if byID["scroll_to_top"].Kind != KindClient || len(byID["scroll_to_top"].Fields) == 0 {
		t.Errorf("client plugin not described: %+v", byID["scroll_to_top"])
	}
	if len(byID["telescope"].Fields) == 0 || byID["telescope"].Kind != KindServer {
		t.Errorf("telescope not described: %+v", byID["telescope"])
	}
	// Server before client, ids sorted within a kind.
	if cat.Plugins[0].Kind != KindServer || cat.Plugins[len(cat.Plugins)-1].Kind != KindClient {
		t.Errorf("unexpected ordering: first=%s last=%s", cat.Plugins[0].ID, cat.Plugins[len(cat.Plugins)-1].ID)
	}
	if _, err := json.Marshal(cat); err != nil {
		t.Fatal(err)
	}
}

func TestBuild_IncludesExternalPlugins(t *testing.T) {
	dir := t.TempDir()
	pdir := filepath.Join(dir, "plugins", "acme")
	if err := os.MkdirAll(pdir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(pdir, "plugin.yaml"), []byte("name: Acme\nslug: acme\nversion: 1.0.0\ndescription: Acme things\n"), 0o644)
	os.WriteFile(filepath.Join(pdir, "blueprint.yaml"), []byte("fields:\n  speed:\n    type: select\n    default: fast\n    options:\n      - { value: fast, label: Fast }\n"), 0o644)
	cat := Build(dir)
	var found *Entry
	for i := range cat.Plugins {
		if cat.Plugins[i].ID == "acme" {
			found = &cat.Plugins[i]
		}
	}
	if found == nil {
		t.Fatal("external plugin not listed")
	}
	if found.Kind != KindExternal || found.Description != "Acme things" || !found.DefaultEnabled {
		t.Errorf("bad external entry: %+v", found)
	}
	if len(found.Fields) != 1 || len(found.Fields[0].Options) != 1 {
		t.Errorf("blueprint options not carried: %+v", found.Fields)
	}
}

func TestBuild_RequiresResolveToCatalogEntries(t *testing.T) {
	cat := Build("")
	byID := map[string]Entry{}
	for _, e := range cat.Plugins {
		byID[e.ID] = e
	}
	for _, e := range cat.Plugins {
		for _, dep := range e.Requires {
			if dep == e.ID {
				t.Errorf("%s requires itself", e.ID)
			}
			if _, ok := byID[dep]; !ok {
				t.Errorf("%s requires unknown plugin %q", e.ID, dep)
			}
		}
	}
	for id, want := range map[string]string{"social_cards": "seo", "search_highlighter": "search"} {
		got := byID[id].Requires
		if len(got) != 1 || got[0] != want {
			t.Errorf("%s requires = %v, want [%s]", id, got, want)
		}
	}
	if byID["sitemap"].Requires != nil {
		t.Errorf("sitemap requires = %v, want none", byID["sitemap"].Requires)
	}
}
