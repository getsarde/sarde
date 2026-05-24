package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Defaults
// ---------------------------------------------------------------------------

func TestDefaults_ReturnsPopulatedConfig(t *testing.T) {
	cfg := Defaults()

	if cfg.Site.Title != "My Site" {
		t.Errorf("Site.Title = %q, want %q", cfg.Site.Title, "My Site")
	}
	if cfg.Site.Language != "en" {
		t.Errorf("Site.Language = %q, want %q", cfg.Site.Language, "en")
	}
	if cfg.Build.Output != "dist" {
		t.Errorf("Build.Output = %q, want %q", cfg.Build.Output, "dist")
	}
	if cfg.Server.Port != 4727 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 4727)
	}
	if !BoolVal(cfg.TOC.Enabled, false) {
		t.Error("TOC.Enabled should be true")
	}
	if cfg.TOC.MinLevel != 2 {
		t.Errorf("TOC.MinLevel = %d, want %d", cfg.TOC.MinLevel, 2)
	}
	if !BoolVal(cfg.Build.Clean, false) {
		t.Error("Build.Clean should be true")
	}
	if BoolVal(cfg.Build.Drafts, true) {
		t.Error("Build.Drafts should be false")
	}
	if !BoolVal(cfg.Search.Enabled, false) {
		t.Error("Search.Enabled should be true")
	}
	if cfg.Search.Provider != "orama" {
		t.Errorf("Search.Provider = %q, want %q", cfg.Search.Provider, "orama")
	}
	if cfg.Content.Dir != "content" {
		t.Errorf("Content.Dir = %q, want %q", cfg.Content.Dir, "content")
	}
	if cfg.Markdown.Highlighting.Style != "class" {
		t.Errorf("Markdown.Highlighting.Style = %q, want %q", cfg.Markdown.Highlighting.Style, "class")
	}
	if cfg.Taxonomies["tags"].Singular != "tag" {
		t.Errorf("Taxonomies[tags].Singular = %q, want %q", cfg.Taxonomies["tags"].Singular, "tag")
	}
}

func TestDefaults_NoPanic(t *testing.T) {
	// Should not panic — if it does, the test fails.
	_ = Defaults()
}

// ---------------------------------------------------------------------------
// TaxonomyConfig YAML parsing
// ---------------------------------------------------------------------------

func TestTaxonomyConfig_ScalarForm(t *testing.T) {
	input := []byte("taxonomies:\n  tags: tag\n  categories: category\n")
	var cfg SiteConfig
	if err := yaml.Unmarshal(input, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Taxonomies["tags"].Singular != "tag" {
		t.Errorf("tags.Singular = %q, want %q", cfg.Taxonomies["tags"].Singular, "tag")
	}
	if cfg.Taxonomies["categories"].Singular != "category" {
		t.Errorf("categories.Singular = %q, want %q", cfg.Taxonomies["categories"].Singular, "category")
	}
}

func TestTaxonomyConfig_MapForm(t *testing.T) {
	input := []byte("taxonomies:\n  tags:\n    singular: tag\n    paginate_by: 20\n")
	var cfg SiteConfig
	if err := yaml.Unmarshal(input, &cfg); err != nil {
		t.Fatal(err)
	}
	tc := cfg.Taxonomies["tags"]
	if tc.Singular != "tag" {
		t.Errorf("Singular = %q, want %q", tc.Singular, "tag")
	}
	if tc.PaginateBy != 20 {
		t.Errorf("PaginateBy = %d, want 20", tc.PaginateBy)
	}
}

func TestMergeTaxonomies(t *testing.T) {
	base := map[string]TaxonomyConfig{
		"tags": {Singular: "tag", PaginateBy: 10},
	}
	over := map[string]TaxonomyConfig{
		"tags":       {PaginateBy: 20},
		"categories": {Singular: "category"},
	}
	mergeTaxonomies(&base, over)
	if base["tags"].Singular != "tag" {
		t.Errorf("tags.Singular = %q, want %q (should keep base)", base["tags"].Singular, "tag")
	}
	if base["tags"].PaginateBy != 20 {
		t.Errorf("tags.PaginateBy = %d, want 20 (should override)", base["tags"].PaginateBy)
	}
	if base["categories"].Singular != "category" {
		t.Errorf("categories.Singular = %q, want %q (should add new)", base["categories"].Singular, "category")
	}
}

// ---------------------------------------------------------------------------
// Loader
// ---------------------------------------------------------------------------

func TestLoadFile_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sarde.yaml")
	content := []byte("site:\n  title: \"Test Site\"\nbuild:\n  output: \"public\"\n")
	os.WriteFile(path, content, 0644)

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile error: %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadFile returned nil config for valid file")
	}
	if cfg.Site.Title != "Test Site" {
		t.Errorf("Site.Title = %q, want %q", cfg.Site.Title, "Test Site")
	}
	if cfg.Build.Output != "public" {
		t.Errorf("Build.Output = %q, want %q", cfg.Build.Output, "public")
	}
}

func TestLoadFile_MissingFile(t *testing.T) {
	cfg, err := LoadFile("/nonexistent/path/sarde.yaml")
	if err != nil {
		t.Fatalf("LoadFile should not error for missing file, got: %v", err)
	}
	if cfg != nil {
		t.Error("LoadFile should return nil config for missing file")
	}
}

func TestLoadFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sarde.yaml")
	os.WriteFile(path, []byte("{{invalid yaml::"), 0644)

	cfg, err := LoadFile(path)
	if err == nil {
		t.Error("LoadFile should error for invalid YAML")
	}
	if cfg != nil {
		t.Error("LoadFile should return nil config for invalid YAML")
	}
}

// ---------------------------------------------------------------------------
// Logo UnmarshalYAML
// ---------------------------------------------------------------------------

func TestLogo_StringForm(t *testing.T) {
	input := []byte(`logo: "/img/logo.svg"`)
	var s struct {
		Logo Logo `yaml:"logo"`
	}
	if err := yaml.Unmarshal(input, &s); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if s.Logo.Light != "/img/logo.svg" {
		t.Errorf("Logo.Light = %q, want %q", s.Logo.Light, "/img/logo.svg")
	}
	if s.Logo.Dark != "/img/logo.svg" {
		t.Errorf("Logo.Dark = %q, want %q", s.Logo.Dark, "/img/logo.svg")
	}
}

func TestLogo_ObjectForm(t *testing.T) {
	input := []byte("logo:\n  light: \"/img/light.svg\"\n  dark: \"/img/dark.svg\"\n  alt: \"My Logo\"\n")
	var s struct {
		Logo Logo `yaml:"logo"`
	}
	if err := yaml.Unmarshal(input, &s); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if s.Logo.Light != "/img/light.svg" {
		t.Errorf("Logo.Light = %q, want %q", s.Logo.Light, "/img/light.svg")
	}
	if s.Logo.Dark != "/img/dark.svg" {
		t.Errorf("Logo.Dark = %q, want %q", s.Logo.Dark, "/img/dark.svg")
	}
	if s.Logo.Alt != "My Logo" {
		t.Errorf("Logo.Alt = %q, want %q", s.Logo.Alt, "My Logo")
	}
}

// ---------------------------------------------------------------------------
// LastUpdatedStrategy UnmarshalYAML
// ---------------------------------------------------------------------------

func TestLastUpdated_StringForms(t *testing.T) {
	cases := []struct {
		yamlSrc string
		want    LastUpdatedStrategy
	}{
		{"build:\n  last_updated: git\n", "git"},
		{"build:\n  last_updated: mtime\n", "mtime"},
		{"build:\n  last_updated: \"false\"\n", "false"},
	}
	for _, c := range cases {
		var cfg SiteConfig
		if err := yaml.Unmarshal([]byte(c.yamlSrc), &cfg); err != nil {
			t.Fatalf("yaml.Unmarshal(%q) error: %v", c.yamlSrc, err)
		}
		if cfg.Build.LastUpdated != c.want {
			t.Errorf("Build.LastUpdated = %q, want %q (src: %q)", cfg.Build.LastUpdated, c.want, c.yamlSrc)
		}
	}
}

func TestLastUpdated_LegacyBoolForms(t *testing.T) {
	// true → "mtime", false → "false" (deprecation warning logged via log.Printf).
	var cfgTrue SiteConfig
	if err := yaml.Unmarshal([]byte("build:\n  last_updated: true\n"), &cfgTrue); err != nil {
		t.Fatalf("Unmarshal(true) error: %v", err)
	}
	if cfgTrue.Build.LastUpdated != "mtime" {
		t.Errorf("last_updated:true → %q, want %q", cfgTrue.Build.LastUpdated, "mtime")
	}

	var cfgFalse SiteConfig
	if err := yaml.Unmarshal([]byte("build:\n  last_updated: false\n"), &cfgFalse); err != nil {
		t.Fatalf("Unmarshal(false) error: %v", err)
	}
	if cfgFalse.Build.LastUpdated != "false" {
		t.Errorf("last_updated:false → %q, want %q", cfgFalse.Build.LastUpdated, "false")
	}
}

// ---------------------------------------------------------------------------
// Merge
// ---------------------------------------------------------------------------

func TestMerge_OverlayWins(t *testing.T) {
	base := Defaults()
	over := &SiteConfig{}
	over.Site.Title = "Override Title"

	mergeConfig(base, over)

	if base.Site.Title != "Override Title" {
		t.Errorf("Title = %q, want %q", base.Site.Title, "Override Title")
	}
}

func TestMerge_ZeroValuesSkipped(t *testing.T) {
	base := Defaults()
	originalTitle := base.Site.Title
	over := &SiteConfig{} // all zero values

	mergeConfig(base, over)

	if base.Site.Title != originalTitle {
		t.Errorf("Title changed to %q, should remain %q", base.Site.Title, originalTitle)
	}
	if base.Server.Port != 4727 {
		t.Errorf("Port changed to %d, should remain 4727", base.Server.Port)
	}
}

func TestMerge_BoolPtrOverride(t *testing.T) {
	base := Defaults()
	// base.Build.Drafts should be false from defaults
	if BoolVal(base.Build.Drafts, true) {
		t.Fatal("precondition: Build.Drafts should be false in defaults")
	}

	over := &SiteConfig{}
	over.Build.Drafts = BoolPtr(true)

	mergeConfig(base, over)

	if !BoolVal(base.Build.Drafts, false) {
		t.Error("Build.Drafts should be true after merge")
	}
}

func TestMerge_SliceReplace(t *testing.T) {
	base := Defaults()
	base.Social = []SocialLink{{Label: "Old", URL: "https://old.com", Icon: "globe"}}

	over := &SiteConfig{}
	over.Social = []SocialLink{
		{Label: "GitHub", URL: "https://github.com", Icon: "github"},
		{Label: "Twitter", URL: "https://twitter.com", Icon: "twitter"},
	}

	mergeConfig(base, over)

	if len(base.Social) != 2 {
		t.Errorf("Social len = %d, want 2", len(base.Social))
	}
	if base.Social[0].Label != "GitHub" {
		t.Errorf("Social[0].Label = %q, want %q", base.Social[0].Label, "GitHub")
	}
}

func TestMerge_MapReplace(t *testing.T) {
	base := Defaults()
	over := &SiteConfig{}
	over.Redirects = map[string]string{"/old": "/new"}

	mergeConfig(base, over)

	if base.Redirects["/old"] != "/new" {
		t.Errorf("Redirects[/old] = %q, want %q", base.Redirects["/old"], "/new")
	}
}

// ---------------------------------------------------------------------------
// Resolve (integration)
// ---------------------------------------------------------------------------

func TestResolve_DefaultsOnly(t *testing.T) {
	cfg, err := Resolve(ResolveOptions{
		ConfigPath: "/nonexistent/sarde.yaml",
	})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if cfg.Site.Title != "My Site" {
		t.Errorf("Site.Title = %q, want %q", cfg.Site.Title, "My Site")
	}
	if cfg.Server.Port != 4727 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 4727)
	}
}

func TestResolve_UserOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sarde.yaml")
	os.WriteFile(path, []byte("site:\n  title: \"User Site\"\n"), 0644)

	cfg, err := Resolve(ResolveOptions{ConfigPath: path})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if cfg.Site.Title != "User Site" {
		t.Errorf("Site.Title = %q, want %q", cfg.Site.Title, "User Site")
	}
	// Other defaults should remain
	if cfg.Server.Port != 4727 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 4727)
	}
}

func TestResolve_CLIFlagsWin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sarde.yaml")
	os.WriteFile(path, []byte("build:\n  drafts: false\n"), 0644)

	cfg, err := Resolve(ResolveOptions{
		ConfigPath: path,
		CLIFlags:   map[string]any{"build.drafts": true},
	})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if !BoolVal(cfg.Build.Drafts, false) {
		t.Error("Build.Drafts should be true (CLI flag wins)")
	}
}

func TestResolve_EnvVarsWin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sarde.yaml")
	os.WriteFile(path, []byte("site:\n  title: \"YAML Title\"\n"), 0644)

	t.Setenv("SARDE_SITE_TITLE", "Env Title")

	cfg, err := Resolve(ResolveOptions{ConfigPath: path})
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if cfg.Site.Title != "Env Title" {
		t.Errorf("Site.Title = %q, want %q", cfg.Site.Title, "Env Title")
	}
}

// ---------------------------------------------------------------------------
// Env overrides
// ---------------------------------------------------------------------------

func TestEnv_StringField(t *testing.T) {
	cfg := Defaults()
	t.Setenv("SARDE_SITE_TITLE", "From Env")

	applyEnvOverrides(cfg, "SARDE")

	if cfg.Site.Title != "From Env" {
		t.Errorf("Site.Title = %q, want %q", cfg.Site.Title, "From Env")
	}
}

func TestEnv_BoolField(t *testing.T) {
	cfg := Defaults()
	t.Setenv("SARDE_BUILD_DRAFTS", "true")

	applyEnvOverrides(cfg, "SARDE")

	if !BoolVal(cfg.Build.Drafts, false) {
		t.Error("Build.Drafts should be true from env")
	}
}

func TestEnv_IntField(t *testing.T) {
	cfg := Defaults()
	t.Setenv("SARDE_SERVER_PORT", "8080")

	applyEnvOverrides(cfg, "SARDE")

	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 8080)
	}
}

func TestEnv_UnsetVarNoEffect(t *testing.T) {
	cfg := Defaults()
	originalTitle := cfg.Site.Title

	applyEnvOverrides(cfg, "SARDE")

	if cfg.Site.Title != originalTitle {
		t.Errorf("Site.Title changed to %q without env var set", cfg.Site.Title)
	}
}
