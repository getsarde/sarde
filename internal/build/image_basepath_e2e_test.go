package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// buildImageSite builds dir with cfg and returns a reader for dist pages.
func buildImageSite(t *testing.T, dir string, cfg *config.SiteConfig) func(rel string) string {
	t.Helper()
	cfg.Build.Minify = config.BoolPtr(false)
	cfg.Site.URL = "https://example.com"
	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	return func(rel string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(dir, "dist", filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		return string(data)
	}
}

// On a site served from a subfolder, every image written with a site-absolute
// path gets the base path: Markdown images, figures, galleries, image
// comparisons, blog covers and grid cards, and og:image.
func TestBuild_SiteAbsoluteImagesGetBasePath(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/docs/guide.md", `---
title: Guide
image: /images/cover.png
---
![Logo](/images/logo.svg)

:::figure[The water cycle]
![Water cycle](/images/water-cycle.png)
:::

:::gallery
![Leaf](/images/leaf.jpg)
![Root](/images/root.jpg)
:::

:::image-compare
![Before](/images/before.png)
![After](/images/after.png)
:::
`)
	writeFixture(t, dir, "content/blog/_index.md", "---\ntitle: Blog\ntemplate: blog/list-grid\n---\n")
	writeFixture(t, dir, "content/blog/trip.md", "---\ntitle: Field Trip\ndate: 2025-05-01\ntemplate: blog/single-cover\nimage: /images/cover.png\n---\nNotes.\n")
	for _, name := range []string{"logo.svg", "water-cycle.png", "leaf.jpg", "root.jpg", "before.png", "after.png", "cover.png"} {
		writeFixture(t, dir, "public/images/"+name, "x")
	}

	cfg := config.Defaults()
	cfg.Build.BasePath = "/demo/"
	read := buildImageSite(t, dir, cfg)

	guide := read("docs/guide/index.html")
	for _, want := range []string{
		`src="/demo/images/logo.svg"`,
		`src="/demo/images/water-cycle.png"`,
		`src="/demo/images/leaf.jpg"`,
		`"src":"/demo/images/root.jpg"`,
		`src="/demo/images/before.png"`,
		`src="/demo/images/after.png"`,
		`content="https://example.com/demo/images/cover.png"`,
	} {
		if !strings.Contains(guide, want) {
			t.Errorf("docs/guide: missing %s", want)
		}
	}
	trip := read("blog/trip/index.html")
	if !strings.Contains(trip, `<img src="/demo/images/cover.png"`) {
		t.Error("blog cover image has no base path")
	}
	if list := read("blog/index.html"); !strings.Contains(list, `src="/demo/images/cover.png"`) {
		t.Error("blog grid card image has no base path")
	}
	for name, html := range map[string]string{"docs/guide": guide, "blog/trip": trip} {
		if strings.Contains(html, `src="/images/`) || strings.Contains(html, `"src":"/images/`) {
			t.Errorf("%s: an image kept its path without the base path", name)
		}
	}
}

// Images exist once, at the site root: the homepage hero on a translated page
// must not get the page's language prefix. (Lab cards use the same rule but
// do not render on a translated labs index yet; see the tracker.)
func TestBuild_ImagesNeverGetLanguagePrefix(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n")
	writeFixture(t, dir, "content/fr/_index.md", "---\ntitle: Accueil\n---\n")
	writeFixture(t, dir, "public/images/hero.svg", "x")

	cfg := config.Defaults()
	cfg.I18n.DefaultLanguage = "en"
	cfg.I18n.Languages = map[string]config.LanguageConfig{
		"en": {Name: "English", Weight: 1, Dir: "ltr"},
		"fr": {Name: "French", Weight: 2, Dir: "ltr"},
	}
	cfg.Homepage.Hero.Title = "Welcome"
	cfg.Homepage.Hero.Image = &config.HeroImageSettings{Src: "/images/hero.svg", Alt: "Hero"}
	read := buildImageSite(t, dir, cfg)

	for _, page := range []string{"index.html", "fr/index.html"} {
		if html := read(page); !strings.Contains(html, `src="/images/hero.svg"`) {
			t.Errorf("%s: hero image is not /images/hero.svg", page)
		}
	}
}
