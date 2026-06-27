package taxonomy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

func setupEnrichTest(t *testing.T, tagsYAML string) (string, map[string]*engine.Taxonomy) {
	t.Helper()
	dir := t.TempDir()
	if tagsYAML != "" {
		dataDir := filepath.Join(dir, "data")
		os.MkdirAll(dataDir, 0755)
		os.WriteFile(filepath.Join(dataDir, "tags.yml"), []byte(tagsYAML), 0644)
	}

	tax := map[string]*engine.Taxonomy{
		"tags": {
			Name:      "tags",
			Singular:  "tag",
			Permalink: "/tags/",
			Terms: map[string]*engine.TaxonomyTerm{
				"go": {
					Name:      "Go",
					Slug:      "go",
					Permalink: "/tags/go/",
					Pages: []*engine.Page{
						{PageIdentity: engine.PageIdentity{Title: "Post 1", Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}},
						{PageIdentity: engine.PageIdentity{Title: "Post 2", Date: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)}},
						{PageIdentity: engine.PageIdentity{Title: "Post 3", Date: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)}},
					},
				},
				"rust": {
					Name:      "Rust",
					Slug:      "rust",
					Permalink: "/tags/rust/",
					Pages:     []*engine.Page{{PageIdentity: engine.PageIdentity{Title: "Rust Post"}}},
				},
			},
		},
	}
	return dir, tax
}

func TestEnrich_AppliesMetadata(t *testing.T) {
	dir, tax := setupEnrichTest(t, `
go:
  label: "Golang"
  description: "The Go language"
  color: "#00ADD8"
  icon: "gopher"
  hidden: true
  priority: 5
`)
	cfg := map[string]config.TaxonomyConfig{"tags": {UndefinedTags: "ignore"}}

	_, err := EnrichTaxonomies(tax, cfg, dir, "")
	if err != nil {
		t.Fatal(err)
	}

	term := tax["tags"].Terms["go"]
	if term.Label != "Golang" {
		t.Errorf("Label = %q, want %q", term.Label, "Golang")
	}
	if term.Description != "The Go language" {
		t.Errorf("Description = %q", term.Description)
	}
	if term.Color != "#00ADD8" {
		t.Errorf("Color = %q", term.Color)
	}
	if term.Icon != "gopher" {
		t.Errorf("Icon = %q", term.Icon)
	}
	if !term.Hidden {
		t.Error("Hidden = false, want true")
	}
	if term.Priority != 5 {
		t.Errorf("Priority = %d, want 5", term.Priority)
	}
}

func TestEnrich_DefaultLabel(t *testing.T) {
	dir, tax := setupEnrichTest(t, "")
	cfg := map[string]config.TaxonomyConfig{"tags": {}}

	_, err := EnrichTaxonomies(tax, cfg, dir, "")
	if err != nil {
		t.Fatal(err)
	}

	term := tax["tags"].Terms["go"]
	if term.Label != "Go" {
		t.Errorf("Label = %q, want %q (should default to Name)", term.Label, "Go")
	}
}

func TestEnrich_DateSorting(t *testing.T) {
	dir, tax := setupEnrichTest(t, "")
	cfg := map[string]config.TaxonomyConfig{"tags": {}}

	_, err := EnrichTaxonomies(tax, cfg, dir, "")
	if err != nil {
		t.Fatal(err)
	}

	pages := tax["tags"].Terms["go"].Pages
	if pages[0].Title != "Post 2" {
		t.Errorf("pages[0] = %q, want Post 2 (June, newest)", pages[0].Title)
	}
	if pages[1].Title != "Post 3" {
		t.Errorf("pages[1] = %q, want Post 3 (March)", pages[1].Title)
	}
	if pages[2].Title != "Post 1" {
		t.Errorf("pages[2] = %q, want Post 1 (January, oldest)", pages[2].Title)
	}
}

func TestEnrich_UndefinedTagsWarn(t *testing.T) {
	dir, tax := setupEnrichTest(t, `
go:
  label: "Go"
`)
	cfg := map[string]config.TaxonomyConfig{"tags": {UndefinedTags: "warn"}}

	warnings, err := EnrichTaxonomies(tax, cfg, dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning (for 'rust'), got %d", len(warnings))
	}
}

func TestEnrich_UndefinedTagsError(t *testing.T) {
	dir, tax := setupEnrichTest(t, `
go:
  label: "Go"
`)
	cfg := map[string]config.TaxonomyConfig{"tags": {UndefinedTags: "error"}}

	_, err := EnrichTaxonomies(tax, cfg, dir, "")
	if err == nil {
		t.Fatal("expected error for undefined term in error mode")
	}
}

func TestEnrich_UndefinedTagsIgnore(t *testing.T) {
	dir, tax := setupEnrichTest(t, `
go:
  label: "Go"
`)
	cfg := map[string]config.TaxonomyConfig{"tags": {UndefinedTags: "ignore"}}

	warnings, err := EnrichTaxonomies(tax, cfg, dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings in ignore mode, got %d", len(warnings))
	}
}

func TestEnrich_UndefinedTagsCreate(t *testing.T) {
	dir, tax := setupEnrichTest(t, `
go:
  label: "Go"
`)
	cfg := map[string]config.TaxonomyConfig{"tags": {UndefinedTags: "create"}}

	warnings, err := EnrichTaxonomies(tax, cfg, dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings in create mode, got %d", len(warnings))
	}
}

func TestEnrich_ErrorModeNoDataFile(t *testing.T) {
	dir, tax := setupEnrichTest(t, "")
	cfg := map[string]config.TaxonomyConfig{"tags": {UndefinedTags: "error"}}

	warnings, err := EnrichTaxonomies(tax, cfg, dir, "")
	if err != nil {
		t.Fatalf("should not error when no data file exists: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning about missing data file, got %d", len(warnings))
	}
}

func TestEnrich_PerLangOverlay(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(filepath.Join(dataDir, "tags.yml"), []byte(`
go:
  label: "Go"
  description: "The Go language"
rust:
  label: "Rust"
`), 0644)
	os.WriteFile(filepath.Join(dataDir, "tags.fr.yml"), []byte(`
go:
  label: "Langage Go"
`), 0644)

	tax := map[string]*engine.Taxonomy{
		"tags": {
			Name: "tags", Singular: "tag", Permalink: "/tags/",
			Terms: map[string]*engine.TaxonomyTerm{
				"go":   {Name: "Go", Slug: "go", Permalink: "/tags/go/", Pages: []*engine.Page{{PageIdentity: engine.PageIdentity{Title: "P1"}}}},
				"rust": {Name: "Rust", Slug: "rust", Permalink: "/tags/rust/", Pages: []*engine.Page{{PageIdentity: engine.PageIdentity{Title: "P2"}}}},
			},
		},
	}
	cfg := map[string]config.TaxonomyConfig{"tags": {UndefinedTags: "ignore"}}

	_, err := EnrichTaxonomies(tax, cfg, dir, "fr")
	if err != nil {
		t.Fatal(err)
	}

	goTerm := tax["tags"].Terms["go"]
	if goTerm.Label != "Langage Go" {
		t.Errorf("go Label = %q, want %q (French overlay)", goTerm.Label, "Langage Go")
	}
	if goTerm.Description != "The Go language" {
		t.Errorf("go Description = %q, want base value", goTerm.Description)
	}

	rustTerm := tax["tags"].Terms["rust"]
	if rustTerm.Label != "Rust" {
		t.Errorf("rust Label = %q, want %q (base, no French override)", rustTerm.Label, "Rust")
	}
}

func TestEnrich_NoMetadataFile(t *testing.T) {
	dir, tax := setupEnrichTest(t, "")
	cfg := map[string]config.TaxonomyConfig{"tags": {}}

	warnings, err := EnrichTaxonomies(tax, cfg, dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings with no data file and default mode, got %d", len(warnings))
	}
	// Label should still default to Name.
	if tax["tags"].Terms["go"].Label != "Go" {
		t.Errorf("Label not set to Name")
	}
}

func TestEnrich_CustomSlugCollisionMergesPages(t *testing.T) {
	dir, tax := setupEnrichTest(t, `
go:
  permalink: "rust"
`)
	cfg := map[string]config.TaxonomyConfig{"tags": {UndefinedTags: "ignore"}}

	warnings, err := EnrichTaxonomies(tax, cfg, dir, "")
	if err != nil {
		t.Fatal(err)
	}

	if len(warnings) != 1 {
		t.Fatalf("expected exactly 1 collision warning, got %d: %v", len(warnings), warnings)
	}
	if strings.Contains(warnings[0], ": term ") {
		t.Errorf("warning must not contain %q (emitTaxonomyWarnings rewrites it): %q", ": term ", warnings[0])
	}

	if _, ok := tax["tags"].Terms["go"]; ok {
		t.Error("old slug \"go\" should be removed after re-key")
	}
	survivor := tax["tags"].Terms["rust"]
	if survivor == nil {
		t.Fatal("term \"rust\" missing after merge")
	}
	if survivor.Name != "Rust" {
		t.Errorf("survivor should keep existing metadata, Name = %q", survivor.Name)
	}
	if len(survivor.Pages) != 4 {
		t.Fatalf("merged pages = %d, want 4 (3 from go + 1 from rust)", len(survivor.Pages))
	}
	if survivor.Pages[0].Title != "Post 2" {
		t.Errorf("merged pages not sorted by date desc, first = %q", survivor.Pages[0].Title)
	}
}

func TestReKeyTerms_SameCustomSlugDeterministic(t *testing.T) {
	tax := &engine.Taxonomy{
		Name: "tags",
		Terms: map[string]*engine.TaxonomyTerm{
			"go":   {Name: "Go", Slug: "go", CustomSlug: "lang", Pages: []*engine.Page{{PageIdentity: engine.PageIdentity{Title: "G"}}}},
			"rust": {Name: "Rust", Slug: "rust", CustomSlug: "lang", Pages: []*engine.Page{{PageIdentity: engine.PageIdentity{Title: "R"}}}},
		},
	}
	warnings := reKeyTerms(tax)

	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if len(tax.Terms) != 1 {
		t.Fatalf("expected 1 surviving term, got %d", len(tax.Terms))
	}
	survivor := tax.Terms["lang"]
	if survivor == nil {
		t.Fatal("term \"lang\" missing")
	}
	// "go" sorts before "rust", so Go inserts first and wins.
	if survivor.Name != "Go" {
		t.Errorf("survivor = %q, want deterministic winner \"Go\"", survivor.Name)
	}
	if len(survivor.Pages) != 2 {
		t.Errorf("merged pages = %d, want 2", len(survivor.Pages))
	}
}

func TestReKeyTerms_ChainReKeyNoFalseCollision(t *testing.T) {
	tax := &engine.Taxonomy{
		Name: "tags",
		Terms: map[string]*engine.TaxonomyTerm{
			"go":   {Name: "Go", Slug: "go", CustomSlug: "rust"},
			"rust": {Name: "Rust", Slug: "rust", CustomSlug: "systems"},
		},
	}
	warnings := reKeyTerms(tax)

	if len(warnings) != 0 {
		t.Fatalf("chain re-key must not warn, got: %v", warnings)
	}
	if got := tax.Terms["rust"]; got == nil || got.Name != "Go" {
		t.Errorf("Terms[\"rust\"] should be the re-keyed Go term, got %+v", got)
	}
	if got := tax.Terms["systems"]; got == nil || got.Name != "Rust" {
		t.Errorf("Terms[\"systems\"] should be the re-keyed Rust term, got %+v", got)
	}
}

func TestReKeyTerms_SharedPageDeduped(t *testing.T) {
	shared := &engine.Page{PageIdentity: engine.PageIdentity{Title: "Shared"}}
	tax := &engine.Taxonomy{
		Name: "tags",
		Terms: map[string]*engine.TaxonomyTerm{
			"go":   {Name: "Go", Slug: "go", CustomSlug: "lang", Pages: []*engine.Page{shared}},
			"rust": {Name: "Rust", Slug: "rust", CustomSlug: "lang", Pages: []*engine.Page{shared}},
		},
	}
	reKeyTerms(tax)

	survivor := tax.Terms["lang"]
	if survivor == nil {
		t.Fatal("term \"lang\" missing")
	}
	if len(survivor.Pages) != 1 {
		t.Errorf("shared page must appear once, got %d entries", len(survivor.Pages))
	}
}
