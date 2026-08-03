package config

import (
	"os"
	"path/filepath"
	"strings"
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
	if cfg.Markdown.Codeblocks.Style != "class" {
		t.Errorf("Markdown.Codeblocks.Style = %q, want %q", cfg.Markdown.Codeblocks.Style, "class")
	}
	if cfg.Markdown.Asides.Style != "classic" {
		t.Errorf("Markdown.Asides.Style = %q, want %q", cfg.Markdown.Asides.Style, "classic")
	}
	if BoolVal(cfg.Markdown.HardWraps, true) {
		t.Error("Markdown.HardWraps should be false")
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
// Homepage
// ---------------------------------------------------------------------------

func TestHomepageHeroOptionalFields_Unmarshal(t *testing.T) {
	input := []byte(`
homepage:
  hero:
    eyebrow: Go HTTP Router
    cta:
      label: Get Started
      url: /docs/
      icon: rocket
    secondary_cta:
      label: GitHub
      url: https://github.com/example/velox
      icon: star
    stats:
      - value: "0"
        label: heap allocations
    code:
      title: Quick start
      language: go
      body: |
        r := velox.New()
        r.GET("/users/:id", getUser)
`)

	var cfg SiteConfig
	if err := yaml.Unmarshal(input, &cfg); err != nil {
		t.Fatal(err)
	}

	hero := cfg.Homepage.Hero
	if hero.Eyebrow != "Go HTTP Router" {
		t.Errorf("Eyebrow = %q, want Go HTTP Router", hero.Eyebrow)
	}
	if hero.CTA == nil || hero.CTA.Label != "Get Started" || hero.CTA.URL != "/docs/" || hero.CTA.Icon != "rocket" {
		t.Fatalf("CTA = %#v, want Get Started with rocket icon", hero.CTA)
	}
	if hero.SecondaryCTA == nil || hero.SecondaryCTA.Label != "GitHub" || hero.SecondaryCTA.URL != "https://github.com/example/velox" || hero.SecondaryCTA.Icon != "star" {
		t.Fatalf("SecondaryCTA = %#v, want GitHub link with star icon", hero.SecondaryCTA)
	}
	if len(hero.Stats) != 1 || hero.Stats[0].Value != "0" || hero.Stats[0].Label != "heap allocations" {
		t.Fatalf("Stats = %#v, want one heap allocations stat", hero.Stats)
	}
	if hero.Code == nil || hero.Code.Title != "Quick start" || hero.Code.Language != "go" {
		t.Fatalf("Code = %#v, want Go quick start", hero.Code)
	}
	if !strings.Contains(hero.Code.Body, `r.GET("/users/:id", getUser)`) {
		t.Errorf("Code.Body = %q, want router sample", hero.Code.Body)
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
	if s.Logo.ReplacesTitle != nil {
		t.Errorf("Logo.ReplacesTitle = %v, want nil when unset", *s.Logo.ReplacesTitle)
	}
}

func TestLogo_ReplacesTitle(t *testing.T) {
	input := []byte("logo:\n  light: \"/img/light.svg\"\n  replaces_title: true\n")
	var s struct {
		Logo Logo `yaml:"logo"`
	}
	if err := yaml.Unmarshal(input, &s); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if !BoolVal(s.Logo.ReplacesTitle, false) {
		t.Error("Logo.ReplacesTitle = false, want true")
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

func TestMerge_HomepageHeroOptionalFields(t *testing.T) {
	base := Defaults()
	over := &SiteConfig{}
	over.Homepage.Hero.Eyebrow = "Go HTTP Router"
	over.Homepage.Hero.SecondaryCTA = &HeroCTA{Label: "GitHub", URL: "https://github.com/example/velox"}
	over.Homepage.Hero.Stats = []HeroStat{{Value: "0", Label: "heap allocations"}}
	over.Homepage.Hero.Code = &HeroCode{Title: "Quick start", Language: "go", Body: "r := velox.New()"}

	mergeConfig(base, over)

	if base.Homepage.Hero.Eyebrow != "Go HTTP Router" {
		t.Errorf("Eyebrow = %q, want Go HTTP Router", base.Homepage.Hero.Eyebrow)
	}
	if base.Homepage.Hero.SecondaryCTA == nil || base.Homepage.Hero.SecondaryCTA.Label != "GitHub" {
		t.Fatalf("SecondaryCTA = %#v, want GitHub", base.Homepage.Hero.SecondaryCTA)
	}
	if len(base.Homepage.Hero.Stats) != 1 || base.Homepage.Hero.Stats[0].Label != "heap allocations" {
		t.Fatalf("Stats = %#v, want heap allocations stat", base.Homepage.Hero.Stats)
	}
	if base.Homepage.Hero.Code == nil || base.Homepage.Hero.Code.Body != "r := velox.New()" {
		t.Fatalf("Code = %#v, want code panel", base.Homepage.Hero.Code)
	}
}

func TestMergeServer_Host(t *testing.T) {
	base := &ServerSettings{Host: "", Port: 4727}
	over := &ServerSettings{Host: "0.0.0.0"}
	mergeServer(base, over)
	if base.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want %q", base.Host, "0.0.0.0")
	}
	if base.Port != 4727 {
		t.Errorf("Port = %d, want 4727 (should be preserved)", base.Port)
	}
}

func TestMergeServer_HostNotOverwrittenByZero(t *testing.T) {
	base := &ServerSettings{Host: "localhost", Port: 4727}
	over := &ServerSettings{Port: 8080}
	mergeServer(base, over)
	if base.Host != "localhost" {
		t.Errorf("Host = %q, want %q (should be preserved)", base.Host, "localhost")
	}
	if base.Port != 8080 {
		t.Errorf("Port = %d, want 8080", base.Port)
	}
}

func TestMergeCollections_DeepMerge(t *testing.T) {
	base := map[string]*CollectionSiteConfig{
		"docs": {Path: "content/docs", Sort: "order"},
	}
	over := map[string]*CollectionSiteConfig{
		"docs": {Layout: "doc"},
		"blog": {Path: "content/blog"},
	}
	mergeCollections(&base, over)

	if base["docs"].Path != "content/docs" {
		t.Errorf("docs.Path = %q, want %q (should preserve base)", base["docs"].Path, "content/docs")
	}
	if base["docs"].Layout != "doc" {
		t.Errorf("docs.Layout = %q, want %q (should apply override)", base["docs"].Layout, "doc")
	}
	if base["docs"].Sort != "order" {
		t.Errorf("docs.Sort = %q, want %q (should preserve base)", base["docs"].Sort, "order")
	}
	if base["blog"] == nil || base["blog"].Path != "content/blog" {
		t.Error("blog collection should be added from override")
	}
}

func TestMergeCollections_NilBase(t *testing.T) {
	var base map[string]*CollectionSiteConfig
	over := map[string]*CollectionSiteConfig{
		"blog": {Path: "content/blog"},
	}
	mergeCollections(&base, over)

	if base["blog"] == nil || base["blog"].Path != "content/blog" {
		t.Error("blog should be added to nil base")
	}
}

func TestMergeCollections_EmptyOverlay(t *testing.T) {
	base := map[string]*CollectionSiteConfig{
		"docs": {Path: "content/docs"},
	}
	mergeCollections(&base, nil)

	if base["docs"].Path != "content/docs" {
		t.Error("base should be unchanged with empty overlay")
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

func TestEnv_InvalidBoolIgnored(t *testing.T) {
	cfg := Defaults()
	original := BoolVal(cfg.Build.Drafts, false)
	t.Setenv("SARDE_BUILD_DRAFTS", "maybe")

	applyEnvOverrides(cfg, "SARDE")

	if BoolVal(cfg.Build.Drafts, false) != original {
		t.Error("invalid bool env value must leave Build.Drafts unchanged")
	}
}

func TestEnv_InvalidIntIgnored(t *testing.T) {
	cfg := Defaults()
	original := cfg.Server.Port
	t.Setenv("SARDE_SERVER_PORT", "abc")

	applyEnvOverrides(cfg, "SARDE")

	if cfg.Server.Port != original {
		t.Errorf("invalid int env value must leave Server.Port unchanged, got %d", cfg.Server.Port)
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
