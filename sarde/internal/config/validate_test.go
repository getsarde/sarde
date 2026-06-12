package config

import (
	"os"
	"testing"

	"github.com/frostybee/sarde/internal/validate"
)

func TestValidate_DefaultsClean(t *testing.T) {
	cfg := Defaults()
	errs := Validate(cfg)
	if len(errs) != 0 {
		t.Errorf("Defaults() should produce zero validation errors, got:\n%s", validate.FormatErrors(errs))
	}
}

func TestValidate_InvalidEnums(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*SiteConfig)
	}{
		{"icons.render", func(c *SiteConfig) { c.Icons.Render = "banana" }},
		{"link_validation.on_broken", func(c *SiteConfig) { c.LinkValidation.OnBroken = "banana" }},
		{"link_validation.report", func(c *SiteConfig) { c.LinkValidation.Report = "banana" }},
		{"deploy.provider", func(c *SiteConfig) { c.Deploy.Provider = "banana" }},
		{"deploy.redirect_format", func(c *SiteConfig) { c.Deploy.RedirectFormat = "banana" }},
		{"prefetch.strategy", func(c *SiteConfig) { c.Prefetch.Strategy = "banana" }},
		{"images.placeholder", func(c *SiteConfig) { c.Images.Placeholder = "banana" }},
		{"link_validation.external.method", func(c *SiteConfig) { c.LinkValidation.External.Method = "banana" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			tt.setup(cfg)
			errs := Validate(cfg)
			if len(errs) == 0 {
				t.Errorf("expected error for %s = banana, got none", tt.name)
			}
			found := false
			for _, e := range errs {
				if e.Path == tt.name {
					found = true
				}
			}
			if !found {
				t.Errorf("expected error with path %q, got: %v", tt.name, errs)
			}
		})
	}
}

func TestValidate_InvalidRanges(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*SiteConfig)
		path  string
	}{
		{"port negative", func(c *SiteConfig) { c.Server.Port = -1 }, "server.port"},
		{"port too high", func(c *SiteConfig) { c.Server.Port = 99999 }, "server.port"},
		{"quality too high", func(c *SiteConfig) { c.Images.Quality = 200 }, "images.quality"},
		{"min_level too high", func(c *SiteConfig) { c.TOC.MinLevel = 9 }, "toc.min_level"},
		{"max_width negative", func(c *SiteConfig) { c.Images.MaxWidth = -5 }, "images.max_width"},
		{"summary_length negative", func(c *SiteConfig) { c.Content.SummaryLength = -1 }, "content.summary_length"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			tt.setup(cfg)
			errs := Validate(cfg)
			found := false
			for _, e := range errs {
				if e.Path == tt.path {
					found = true
				}
			}
			if !found {
				t.Errorf("expected error with path %q, got: %v", tt.path, errs)
			}
		})
	}
}

func TestValidate_Interdependencies(t *testing.T) {
	cfg := Defaults()
	cfg.TOC.MinLevel = 5
	cfg.TOC.MaxLevel = 2
	errs := Validate(cfg)
	found := false
	for _, e := range errs {
		if e.Path == "toc.min_level" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected interdependency error for toc.min_level > toc.max_level, got: %v", errs)
	}
}

func TestValidate_ZeroValuesSkipped(t *testing.T) {
	// An empty struct triggers required-field errors but no enum/range errors.
	cfg := &SiteConfig{}
	errs := Validate(cfg)
	for _, e := range errs {
		if e.Message != "is required" {
			t.Errorf("unexpected non-required error on empty struct: %v", e)
		}
	}
}

func TestValidate_RequiredFields(t *testing.T) {
	cfg := Defaults()
	cfg.Site.Title = ""
	cfg.Site.Language = ""
	cfg.Build.Output = ""
	cfg.Content.Dir = ""
	errs := Validate(cfg)

	required := map[string]bool{
		"site.title":    false,
		"site.language": false,
		"build.output":  false,
		"content.dir":   false,
	}
	for _, e := range errs {
		if _, ok := required[e.Path]; ok {
			required[e.Path] = true
		}
	}
	for path, found := range required {
		if !found {
			t.Errorf("expected required error for %q, not found", path)
		}
	}
}

func TestValidate_RequiredFields_DefaultsSatisfy(t *testing.T) {
	cfg := Defaults()
	errs := Validate(cfg)
	for _, e := range errs {
		if e.Message == "is required" {
			t.Errorf("Defaults() should satisfy all required fields, but %q failed", e.Path)
		}
	}
}

func TestValidate_CollectionEnums(t *testing.T) {
	cfg := Defaults()
	cfg.Collections = map[string]*CollectionSiteConfig{
		"docs": {Layout: "banana", I18nFallback: "banana"},
	}
	errs := Validate(cfg)
	if len(errs) < 2 {
		t.Fatalf("expected at least 2 errors for invalid collection enums, got %d: %v", len(errs), errs)
	}
	paths := map[string]bool{}
	for _, e := range errs {
		paths[e.Path] = true
	}
	if !paths["collections.docs.layout"] {
		t.Error("expected error for collections.docs.layout")
	}
	if !paths["collections.docs.i18n_fallback"] {
		t.Error("expected error for collections.docs.i18n_fallback")
	}
}

func TestValidate_TaxonomyEnums(t *testing.T) {
	cfg := Defaults()
	cfg.Taxonomies = map[string]TaxonomyConfig{
		"tags": {UndefinedTags: "banana"},
	}
	errs := Validate(cfg)
	found := false
	for _, e := range errs {
		if e.Path == "taxonomies.tags.undefined_tags" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error for taxonomies.tags.undefined_tags, got: %v", errs)
	}
}

func TestValidate_I18nLanguageDir(t *testing.T) {
	cfg := Defaults()
	cfg.I18n.Languages = map[string]LanguageConfig{
		"ar": {Name: "Arabic", Dir: "banana"},
	}
	errs := Validate(cfg)
	found := false
	for _, e := range errs {
		if e.Path == "i18n.languages.ar.dir" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error for i18n.languages.ar.dir, got: %v", errs)
	}
}

func TestLoadFileStrict_RejectsUnknown(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/sarde.yaml"
	writeTestFile(t, path, "tab: false\n")

	_, err := LoadFileStrict(path)
	if err == nil {
		t.Error("expected error for unknown field 'tab', got nil")
	}
}

func TestLoadFileStrict_MissingFile(t *testing.T) {
	cfg, err := LoadFileStrict("/nonexistent/sarde.yaml")
	if err != nil {
		t.Fatalf("expected (nil, nil) for missing file, got error: %v", err)
	}
	if cfg != nil {
		t.Error("expected nil config for missing file")
	}
}

func TestLoadFileStrict_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/sarde.yaml"
	writeTestFile(t, path, "site:\n  title: \"Test\"\n")

	cfg, err := LoadFileStrict(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Site.Title != "Test" {
		t.Errorf("Site.Title = %q, want %q", cfg.Site.Title, "Test")
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
