package taxonomy

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/engine"
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
						{Title: "Post 1", Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
						{Title: "Post 2", Date: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)},
						{Title: "Post 3", Date: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)},
					},
				},
				"rust": {
					Name:      "Rust",
					Slug:      "rust",
					Permalink: "/tags/rust/",
					Pages:     []*engine.Page{{Title: "Rust Post"}},
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

	_, err := EnrichTaxonomies(tax, cfg, dir)
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

	_, err := EnrichTaxonomies(tax, cfg, dir)
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

	_, err := EnrichTaxonomies(tax, cfg, dir)
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

	warnings, err := EnrichTaxonomies(tax, cfg, dir)
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

	_, err := EnrichTaxonomies(tax, cfg, dir)
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

	warnings, err := EnrichTaxonomies(tax, cfg, dir)
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

	warnings, err := EnrichTaxonomies(tax, cfg, dir)
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

	warnings, err := EnrichTaxonomies(tax, cfg, dir)
	if err != nil {
		t.Fatalf("should not error when no data file exists: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning about missing data file, got %d", len(warnings))
	}
}

func TestEnrich_NoMetadataFile(t *testing.T) {
	dir, tax := setupEnrichTest(t, "")
	cfg := map[string]config.TaxonomyConfig{"tags": {}}

	warnings, err := EnrichTaxonomies(tax, cfg, dir)
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
