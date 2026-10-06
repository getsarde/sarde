package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/theme"
)

// buildHeadFixtureSite builds a docs site with the `clean` preset (Plus
// Jakarta Sans and Fira Code, neither bundled), theme.web_fonts set to
// bunny, one site-level head tag and a head tag cascaded from the docs
// section, and returns the rendered docs/guide page.
func buildHeadFixtureSite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\ncascade:\n  head:\n    - tag: meta\n      attrs:\n        name: x-section\n        content: docs\n---\n")
	writeFixture(t, dir, "content/docs/guide.md", "---\ntitle: Guide\nweight: 1\n---\n# Guide\nBody.\n")

	cfg := config.Defaults()
	cfg.Build.Minify = config.BoolPtr(false)
	cfg.Theme.Preset = "clean"
	cfg.Theme.WebFonts = "bunny"
	cfg.Head.Tags = []config.HeadTag{
		{Tag: "meta", Attrs: map[string]string{"name": "x-site", "content": "everywhere"}},
		{Tag: "style", Content: `.brand { font-family: "Brand Sans"; }`},
	}

	thm, _ := theme.LoadFromFS(embedded.ThemeFS(), ".")
	light := theme.DeriveTokens(theme.ResolveTokens(theme.DefaultTokens(), thm, "clean", nil))
	dark := theme.ResolveDarkTokens(theme.DefaultDarkTokens(), thm, "clean", nil)
	themeCfg := &engine.ThemeConfig{
		Name:        "default",
		Tokens:      light,
		DarkTokens:  dark,
		DarkEnabled: true,
		StyleTag:    theme.GenerateStyleTag(light, dark),
	}

	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "dist", "docs", "guide", "index.html"))
	if err != nil {
		t.Fatalf("reading rendered guide page: %v", err)
	}
	return string(data)
}

// TestBuild_HeadLoadsPresetWebFontsAndSiteTags covers theme.web_fonts and
// head.tags end to end: the clean preset's fonts come from Bunny Fonts in
// one stylesheet, and the site and cascaded head tags reach the page with
// their style content intact.
func TestBuild_HeadLoadsPresetWebFontsAndSiteTags(t *testing.T) {
	html := buildHeadFixtureSite(t)

	for _, want := range []string{
		`<link rel="preconnect" href="https://fonts.bunny.net" crossorigin>`,
		`https://fonts.bunny.net/css?family=Plus+Jakarta+Sans:`,
		`|Fira+Code:`,
		`<meta content="everywhere" name="x-site">`,
		`<style>.brand { font-family: "Brand Sans"; }</style>`,
		`<meta content="docs" name="x-section">`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("rendered page is missing %s", want)
		}
	}
	if n := strings.Count(html, `rel="stylesheet" href="https://fonts.bunny.net`); n != 1 {
		t.Errorf("want one web font stylesheet, got %d", n)
	}
	// The font stylesheet comes before the theme's own CSS.
	head := html[:strings.Index(html, "</head>")]
	if fonts, themeCSS := strings.Index(head, "fonts.bunny.net/css"), strings.Index(head, `href="/assets/`); themeCSS == -1 || fonts > themeCSS {
		t.Errorf("the web font stylesheet should come before the theme CSS:\n%s", head)
	}
}
