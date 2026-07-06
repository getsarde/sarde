package config

import (
	"os"
	"testing"

	"github.com/getsarde/sarde/internal/validate"
)

func TestValidate_DefaultsClean(t *testing.T) {
	cfg := Defaults()
	errs, _ := Validate(cfg, nil)
	if len(errs) != 0 {
		t.Errorf("Defaults() should produce zero hard errors, got:\n%s", validate.FormatErrors(errs))
	}
}

func TestValidate_DefaultsWarnings(t *testing.T) {
	cfg := Defaults()
	_, warns := Validate(cfg, nil)
	// Defaults have empty site.url and site.description — those produce warnings.
	if len(warns) < 2 {
		t.Errorf("expected at least 2 warnings for empty url/description, got %d", len(warns))
	}
}

func TestValidate_InvalidEnums(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*SiteConfig)
		path  string
	}{
		{"icons.render", func(c *SiteConfig) { c.Icons.Render = "banana" }, "icons.render"},
		{"link_validation.on_broken", func(c *SiteConfig) { c.LinkValidation.OnBroken = "banana" }, "link_validation.on_broken"},
		{"link_validation.report", func(c *SiteConfig) { c.LinkValidation.Report = "banana" }, "link_validation.report"},
		{"link_validation.level", func(c *SiteConfig) { c.LinkValidation.Level = "banana" }, "link_validation.level"},
		{"link_validation.same_site_policy", func(c *SiteConfig) { c.LinkValidation.SameSitePolicy = "banana" }, "link_validation.same_site_policy"},
		{"deploy.provider", func(c *SiteConfig) { c.Deploy.Provider = "banana" }, "deploy.provider"},
		{"deploy.redirect_format", func(c *SiteConfig) { c.Deploy.RedirectFormat = "banana" }, "deploy.redirect_format"},
		{"prefetch.strategy", func(c *SiteConfig) { c.Prefetch.Strategy = "banana" }, "prefetch.strategy"},
		{"images.placeholder", func(c *SiteConfig) { c.Images.Placeholder = "banana" }, "images.placeholder"},
		{"link_validation.external.method", func(c *SiteConfig) { c.LinkValidation.External.Method = "banana" }, "link_validation.external.method"},
		{"search.provider", func(c *SiteConfig) { c.Search.Provider = "banana" }, "search.provider"},
		{"markdown.codeblocks.style", func(c *SiteConfig) { c.Markdown.Codeblocks.Style = "banana" }, "markdown.codeblocks.style"},
		{"build.last_updated", func(c *SiteConfig) { c.Build.LastUpdated = "banana" }, "build.last_updated"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			tt.setup(cfg)
			errs, _ := Validate(cfg, nil)
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
		{"prefetch.delay negative", func(c *SiteConfig) { c.Prefetch.Delay = IntPtr(-1) }, "prefetch.delay"},
		{"heading_max_length zero", func(c *SiteConfig) { c.ContentLint.Rules.HeadingMaxLength = -1 }, "content_lint.rules.heading_max_length"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			tt.setup(cfg)
			errs, _ := Validate(cfg, nil)
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
	errs, _ := Validate(cfg, nil)
	found := false
	for _, e := range errs {
		if e.Path == "toc.min_level" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected interdependency error, got: %v", errs)
	}
}

func TestValidate_ZeroValuesNoEnumErrors(t *testing.T) {
	cfg := &SiteConfig{}
	errs, _ := Validate(cfg, nil)
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
	errs, _ := Validate(cfg, nil)

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

func TestValidate_RecommendedWarnings(t *testing.T) {
	cfg := Defaults()
	cfg.Site.URL = ""
	cfg.Site.Description = ""
	_, warns := Validate(cfg, nil)

	paths := map[string]bool{}
	for _, w := range warns {
		paths[w.Path] = true
	}
	if !paths["site.url"] {
		t.Error("expected warning for site.url")
	}
	if !paths["site.description"] {
		t.Error("expected warning for site.description")
	}
}

func TestValidate_RecommendedNoWarningWhenSet(t *testing.T) {
	cfg := Defaults()
	cfg.Site.URL = "https://example.com"
	cfg.Site.Description = "A test site"
	_, warns := Validate(cfg, nil)
	if len(warns) != 0 {
		t.Errorf("expected no warnings when url/description set, got: %v", warns)
	}
}

func TestValidate_CollectionEnums(t *testing.T) {
	cfg := Defaults()
	cfg.Collections = map[string]*CollectionSiteConfig{
		"docs": {Layout: "banana", I18nFallback: "banana"},
	}
	errs, _ := Validate(cfg, nil)
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

func TestValidate_CollectionRanges(t *testing.T) {
	cfg := Defaults()
	cfg.Collections = map[string]*CollectionSiteConfig{
		"docs": {
			Paginate: -1,
			Sidebar:  &CollectionSidebarConfig{MaxDepth: 99},
			TOC:      &CollectionTOCConfig{Depth: 9},
		},
	}
	errs, _ := Validate(cfg, nil)
	paths := map[string]bool{}
	for _, e := range errs {
		paths[e.Path] = true
	}
	if !paths["collections.docs.paginate"] {
		t.Error("expected error for collections.docs.paginate")
	}
	if !paths["collections.docs.sidebar.max_depth"] {
		t.Error("expected error for collections.docs.sidebar.max_depth")
	}
	if !paths["collections.docs.toc.depth"] {
		t.Error("expected error for collections.docs.toc.depth")
	}
}

func TestValidate_VersioningFallback(t *testing.T) {
	cfg := Defaults()
	cfg.Collections = map[string]*CollectionSiteConfig{
		"docs": {
			Versioning: &VersioningConfig{Fallback: "banana"},
		},
	}
	errs, _ := Validate(cfg, nil)
	found := false
	for _, e := range errs {
		if e.Path == "collections.docs.versioning.fallback" {
			found = true
		}
	}
	if !found {
		t.Error("expected error for collections.docs.versioning.fallback")
	}
}

func TestValidate_TaxonomyPaginateBy(t *testing.T) {
	cfg := Defaults()
	cfg.Taxonomies = map[string]TaxonomyConfig{
		"tags": {PaginateBy: -5},
	}
	errs, _ := Validate(cfg, nil)
	found := false
	for _, e := range errs {
		if e.Path == "taxonomies.tags.paginate_by" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error for taxonomies.tags.paginate_by, got: %v", errs)
	}
}

func TestValidate_ImageFormats(t *testing.T) {
	cfg := Defaults()
	cfg.Images.Formats = []string{"webp", "banana"}
	errs, _ := Validate(cfg, nil)
	found := false
	for _, e := range errs {
		if e.Path == "images.formats[1]" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error for images.formats[1], got: %v", errs)
	}
}

func TestValidate_ImageWidths(t *testing.T) {
	cfg := Defaults()
	cfg.Images.Widths = []int{400, -1, 800}
	errs, _ := Validate(cfg, nil)
	found := false
	for _, e := range errs {
		if e.Path == "images.widths[1]" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error for images.widths[1], got: %v", errs)
	}
}

func TestValidate_PluginNames(t *testing.T) {
	known := []string{"search", "seo", "scroll-to-top"}
	cfg := Defaults()
	cfg.Plugins.Enabled = []string{"search", "serach", "seo", "scroll-to-top"}
	errs, _ := Validate(cfg, known)
	found := false
	for _, e := range errs {
		if e.Path == "plugins.enabled[1]" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error for plugins.enabled[1] (typo 'serach'), got: %v", errs)
	}
	for _, e := range errs {
		if e.Path == "plugins.enabled[3]" {
			t.Errorf("scroll-to-top should be valid, but got error: %v", e)
		}
	}
}

func TestValidate_PluginNamesSkippedWhenNil(t *testing.T) {
	cfg := Defaults()
	cfg.Plugins.Enabled = []string{"anything", "goes", "here"}
	errs, _ := Validate(cfg, nil)
	for _, e := range errs {
		if e.Path == "plugins.enabled[0]" || e.Path == "plugins.enabled[1]" || e.Path == "plugins.enabled[2]" {
			t.Errorf("plugin validation should be skipped when knownPlugins is nil, got: %v", e)
		}
	}
}

func TestValidate_I18nLanguageDir(t *testing.T) {
	cfg := Defaults()
	cfg.I18n.Languages = map[string]LanguageConfig{
		"ar": {Name: "Arabic", Dir: "banana"},
	}
	errs, _ := Validate(cfg, nil)
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
	os.WriteFile(path, []byte("tab: false\n"), 0644)

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
	os.WriteFile(path, []byte("site:\n  title: \"Test\"\n"), 0644)

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
