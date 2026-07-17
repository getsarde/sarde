package build

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/theme"
)

// createLargeSite creates a test site with the given number of pages
// split across blog and docs collections.
func createLargeSite(b *testing.B, pageCount int) string {
	b.Helper()
	dir := b.TempDir()

	writeBenchFixture(dir, "content/_index.md", "---\ntitle: Home\n---\n# Welcome\n")
	writeBenchFixture(dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeBenchFixture(dir, "content/blog/_index.md", "---\ntitle: Blog\n---\n")

	for i := 0; i < pageCount; i++ {
		if i%2 == 0 {
			writeBenchFixture(dir, fmt.Sprintf("content/docs/page-%04d.md", i),
				fmt.Sprintf("---\ntitle: Doc Page %d\nweight: %d\n---\n# Doc Page %d\n\nThis is documentation page number %d with some **markdown** content.\n\n## Section\n\nMore content here with `code` and [links](/docs/).\n", i, i, i, i))
		} else {
			writeBenchFixture(dir, fmt.Sprintf("content/blog/post-%04d.md", i),
				fmt.Sprintf("---\ntitle: Blog Post %d\ndate: 2025-01-%02dT00:00:00Z\n---\n# Blog Post %d\n\nThis is blog post number %d with some **markdown** content.\n\n## Section\n\nMore content here with `code` and [links](/blog/).\n", i, (i%28)+1, i, i))
		}
	}

	return dir
}

func writeBenchFixture(base, rel, body string) {
	path := filepath.Join(base, filepath.FromSlash(rel))
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(body), 0o644)
}

func benchmarkBuild(b *testing.B, pageCount int, parallel bool) {
	dir := createLargeSite(b, pageCount)
	benchmarkBuildDir(b, dir, parallel, nil)
}

func benchmarkBuildDir(b *testing.B, dir string, parallel bool, configure func(*config.SiteConfig)) {
	cfg := config.Defaults()
	cfg.Build.Parallel = &parallel
	if configure != nil {
		configure(cfg)
	}
	themeCfg := buildThemeConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewSiteBuilder(BuildOptions{
			ProjectDir:  dir,
			Config:      cfg,
			ThemeConfig: themeCfg,
			EmbeddedFS:  embedded.ThemeFS(),
		})
		result, err := builder.Build()
		if err != nil {
			b.Fatal(err)
		}
		if result.PageCount == 0 {
			b.Fatal("no pages rendered")
		}
	}
}

func BenchmarkBuild_100Pages_Parallel(b *testing.B) {
	benchmarkBuild(b, 100, true)
}

func BenchmarkBuild_100Pages_Serial(b *testing.B) {
	benchmarkBuild(b, 100, false)
}

func BenchmarkBuild_500Pages_Parallel(b *testing.B) {
	benchmarkBuild(b, 500, true)
}

func BenchmarkBuild_500Pages_Serial(b *testing.B) {
	benchmarkBuild(b, 500, false)
}

func BenchmarkBuild_2000Pages_Parallel(b *testing.B) {
	benchmarkBuild(b, 2000, true)
}

func BenchmarkBuild_2000Pages_Serial(b *testing.B) {
	benchmarkBuild(b, 2000, false)
}

func BenchmarkBuild_Multilingual_Parallel(b *testing.B) {
	dir := createMultilingualBenchSite(b, 250)
	benchmarkBuildDir(b, dir, true, configureMultilingualBench)
}

func BenchmarkBuild_Multilingual_Serial(b *testing.B) {
	dir := createMultilingualBenchSite(b, 250)
	benchmarkBuildDir(b, dir, false, configureMultilingualBench)
}

func BenchmarkBuild_AssetHeavy_Parallel(b *testing.B) {
	dir := createAssetHeavyBenchSite(b, 250)
	benchmarkBuildDir(b, dir, true, configureAssetBench)
}

func BenchmarkBuild_AssetHeavy_Serial(b *testing.B) {
	dir := createAssetHeavyBenchSite(b, 250)
	benchmarkBuildDir(b, dir, false, configureAssetBench)
}

func BenchmarkBuild_TestsiteBenchmark_Parallel(b *testing.B) {
	benchmarkRealTestsite(b, "benchmark", true)
}

func BenchmarkBuild_TestsiteBenchmark_Serial(b *testing.B) {
	benchmarkRealTestsite(b, "benchmark", false)
}

func BenchmarkBuild_TestsiteGeneral_Parallel(b *testing.B) {
	benchmarkRealTestsite(b, "general", true)
}

func BenchmarkBuild_TestsiteGeneral_Serial(b *testing.B) {
	benchmarkRealTestsite(b, "general", false)
}

func benchmarkRealTestsite(b *testing.B, name string, parallel bool) {
	src := repoTestsiteDir(b, name)
	if _, err := os.Stat(filepath.Join(src, "sarde.yaml")); err != nil {
		b.Skipf("real testsite %q not found at %s", name, src)
	}
	dir := copyRealTestsite(b, src)
	cfg, themeCfg := resolveBenchmarkSite(b, dir)
	cfg.Build.Parallel = &parallel

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder := NewSiteBuilder(BuildOptions{
			ProjectDir:  dir,
			Config:      cfg,
			ThemeConfig: themeCfg,
			EmbeddedFS:  embedded.ThemeFS(),
		})
		result, err := builder.Build()
		if err != nil {
			b.Fatal(err)
		}
		if result.PageCount == 0 {
			b.Fatal("no pages rendered")
		}
	}
}

func createMultilingualBenchSite(b *testing.B, pageCount int) string {
	b.Helper()
	dir := b.TempDir()
	writeBenchFixture(dir, "i18n/fr.yaml", "nav:\n  search: Rechercher\n")
	writeBenchFixture(dir, "content/_index.md", "---\ntitle: Home\n---\n# Welcome\n")
	writeBenchFixture(dir, "content/blog/_index.md", "---\ntitle: Blog\n---\n")
	writeBenchFixture(dir, "content/fr/blog/_index.md", "---\ntitle: Blog FR\n---\n")
	for i := 0; i < pageCount; i++ {
		body := fmt.Sprintf("---\ntitle: Post %d\ndate: 2025-01-%02dT00:00:00Z\n---\n# Post %d\n\nContent.\n", i, (i%28)+1, i)
		writeBenchFixture(dir, fmt.Sprintf("content/blog/post-%04d.md", i), body)
		writeBenchFixture(dir, fmt.Sprintf("content/fr/blog/post-%04d.md", i), body)
	}
	return dir
}

func configureMultilingualBench(cfg *config.SiteConfig) {
	cfg.I18n.DefaultLanguage = "en"
	cfg.I18n.Languages = map[string]config.LanguageConfig{
		"en": {Name: "English", Weight: 1, Dir: "ltr"},
		"fr": {Name: "French", Weight: 2, Dir: "ltr"},
	}
}

func createAssetHeavyBenchSite(b *testing.B, pageCount int) string {
	b.Helper()
	dir := b.TempDir()
	writeBenchFixture(dir, "assets/css/site.css", "body { color: #222; }\n")
	writeBenchFixture(dir, "assets/js/site.js", "console.log('bench');\n")
	writeBenchFixture(dir, "content/_index.md", "---\ntitle: Home\n---\n# Welcome\n")
	writeBenchFixture(dir, "content/blog/_index.md", "---\ntitle: Blog\n---\n")
	for i := 0; i < pageCount; i++ {
		base := fmt.Sprintf("content/blog/bundle-%04d", i)
		writeBenchFixture(dir, base+"/index.md", fmt.Sprintf("---\ntitle: Bundle %d\ndate: 2025-01-%02dT00:00:00Z\n---\n# Bundle %d\n", i, (i%28)+1, i))
		writeBenchFixture(dir, base+"/asset-a.txt", "asset a\n")
		writeBenchFixture(dir, base+"/asset-b.txt", "asset b\n")
	}
	return dir
}

func configureAssetBench(cfg *config.SiteConfig) {
	cfg.Head.CustomCSS = []string{"css/site.css"}
	cfg.Head.CustomJS = []string{"js/site.js"}
}

func repoTestsiteDir(b *testing.B, name string) string {
	b.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		b.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "testsite", name))
}

func copyRealTestsite(b *testing.B, src string) string {
	b.Helper()
	dst := b.TempDir()
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			switch rel {
			case ".cache", "dist", "cmd":
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, filepath.FromSlash(rel)), 0o755)
		}
		if strings.HasPrefix(rel, ".cache/") || strings.HasPrefix(rel, "dist/") || strings.HasPrefix(rel, "cmd/") {
			return nil
		}
		return copyBenchmarkFile(path, filepath.Join(dst, filepath.FromSlash(rel)))
	})
	if err != nil {
		b.Fatalf("copying real testsite: %v", err)
	}
	return dst
}

func copyBenchmarkFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func resolveBenchmarkSite(b *testing.B, projectDir string) (*config.SiteConfig, *engine.ThemeConfig) {
	b.Helper()
	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath:   filepath.Join(projectDir, "sarde.yaml"),
		EnvPrefix:    "SARDE",
		KnownPlugins: KnownPluginNames(projectDir),
	})
	if err != nil {
		b.Fatalf("resolving config: %v", err)
	}

	var thm *theme.Theme
	if cfg.Theme.Name != "" && cfg.Theme.Name != "default" {
		thm, _ = theme.LoadFromDir(filepath.Join(projectDir, "themes", cfg.Theme.Name))
	}
	if thm == nil {
		thm, _ = theme.LoadFromFS(embedded.ThemeFS(), ".")
	}

	foldBenchmarkThemeShortcuts(cfg)
	light := theme.ResolveTokens(theme.DefaultTokens(), thm, cfg.Theme.Preset, cfg.Theme.Overrides)
	light = theme.DeriveTokens(light)
	dark := theme.ResolveDarkTokens(theme.DefaultDarkTokens(), thm, cfg.Theme.Preset, nil)

	name := "Default"
	slug := "default"
	if thm != nil {
		if thm.Name != "" {
			name = thm.Name
		}
		if thm.Slug != "" {
			slug = thm.Slug
		}
	}
	return cfg, &engine.ThemeConfig{
		Name:        name,
		Slug:        slug,
		Tokens:      light,
		DarkTokens:  dark,
		DarkEnabled: config.BoolVal(cfg.Theme.Dark, true),
		StyleTag:    theme.GenerateStyleTag(light, dark),
	}
}

func foldBenchmarkThemeShortcuts(cfg *config.SiteConfig) {
	accentVal := cfg.Theme.AccentColor
	if accentVal == "" {
		accentVal = cfg.Theme.PrimaryColor
	}
	shortcuts := map[string]string{
		"accent":    accentVal,
		"font-sans": cfg.Theme.FontFamily,
		"font-mono": cfg.Theme.FontMono,
	}
	if cfg.Theme.Overrides == nil {
		cfg.Theme.Overrides = make(map[string]string)
	}
	for token, val := range shortcuts {
		if val != "" {
			if _, exists := cfg.Theme.Overrides[token]; !exists {
				cfg.Theme.Overrides[token] = val
			}
		}
	}
	if cfg.Theme.CodeLight != "" && cfg.Markdown.Codeblocks.LightTheme == "" {
		cfg.Markdown.Codeblocks.LightTheme = cfg.Theme.CodeLight
	}
	if cfg.Theme.CodeDark != "" && cfg.Markdown.Codeblocks.DarkTheme == "" {
		cfg.Markdown.Codeblocks.DarkTheme = cfg.Theme.CodeDark
	}
}
