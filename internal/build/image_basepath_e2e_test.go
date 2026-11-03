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

// Images exist once, at the site root: the homepage hero and a lab card on a
// translated page must not get the page's language prefix.
func TestBuild_ImagesNeverGetLanguagePrefix(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n")
	writeFixture(t, dir, "content/fr/_index.md", "---\ntitle: Accueil\n---\n")
	writeFixture(t, dir, "public/images/hero.svg", "x")
	writeFixture(t, dir, "public/images/lab.svg", "x")
	writeFixture(t, dir, "content/labs/_index.md", "---\ntitle: Labs\n---\n")
	writeFixture(t, dir, "content/fr/labs/_index.md", "---\ntitle: Ateliers\n---\n")
	writeFixture(t, dir, "content/labs/web/_index.md", "---\ntitle: Web\nimage: /images/lab.svg\n---\n")
	writeFixture(t, dir, "content/fr/labs/web/_index.md", "---\ntitle: Web\nimage: /images/lab.svg\n---\n")
	writeFixture(t, dir, "content/labs/web/setup.md", "---\ntitle: Setup\n---\nStep.\n")

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
	for _, page := range []string{"labs/index.html", "fr/labs/index.html"} {
		if html := read(page); !strings.Contains(html, `src="/images/lab.svg"`) {
			t.Errorf("%s: lab card image is not /images/lab.svg", page)
		}
	}
}

// A relative frontmatter image names a file beside the page, so every
// template that shows the page reads it from the page's folder: the blog
// grid card, the cover, and a lab card all point at the bundle's file. A
// translated post gets no language prefix, because bundle files are written
// once, at the language-free path.
func TestBuild_RelativeFrontmatterImageResolvesAgainstItsPage(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n")
	writeFixture(t, dir, "content/fr/_index.md", "---\ntitle: Accueil\n---\n")
	writeFixture(t, dir, "content/blog/_index.md", "---\ntitle: Blog\ntemplate: blog/list-grid\n---\n")
	writeFixture(t, dir, "content/blog/trip/index.md", "---\ntitle: Field Trip\ndate: 2025-05-01\ntemplate: blog/single-cover\nimage: cover.jpg\n---\nNotes.\n")
	writeFixture(t, dir, "content/blog/trip/cover.jpg", "x")
	writeFixture(t, dir, "content/fr/blog/sortie/index.md", "---\ntitle: Sortie\ndate: 2025-05-02\ntemplate: blog/single-cover\nimage: photo.jpg\n---\nNotes.\n")
	writeFixture(t, dir, "content/fr/blog/sortie/photo.jpg", "x")
	writeFixture(t, dir, "content/labs/_index.md", "---\ntitle: Labs\n---\n")
	writeFixture(t, dir, "content/labs/web/_index.md", "---\ntitle: Web\nimage: card.png\n---\n")
	writeFixture(t, dir, "content/labs/web/card.png", "x")
	writeFixture(t, dir, "content/labs/web/setup.md", "---\ntitle: Setup\n---\nStep.\n")

	cfg := config.Defaults()
	cfg.Build.BasePath = "/demo/"
	cfg.I18n.DefaultLanguage = "en"
	cfg.I18n.Languages = map[string]config.LanguageConfig{
		"en": {Name: "English", Weight: 1, Dir: "ltr"},
		"fr": {Name: "French", Weight: 2, Dir: "ltr"},
	}
	read := buildImageSite(t, dir, cfg)

	if html := read("blog/trip/index.html"); !strings.Contains(html, `<img src="/demo/blog/trip/cover.jpg"`) {
		t.Error("blog cover: relative image not resolved against the post")
	}
	if html := read("blog/index.html"); !strings.Contains(html, `src="/demo/blog/trip/cover.jpg"`) {
		t.Error("blog grid card: relative image not resolved against the post")
	}
	if html := read("fr/blog/sortie/index.html"); !strings.Contains(html, `<img src="/demo/blog/sortie/photo.jpg"`) {
		t.Error("translated blog cover: want the language-free bundle path")
	}
	if html := read("labs/index.html"); !strings.Contains(html, `src="/demo/labs/web/card.png"`) {
		t.Error("lab card: relative image not resolved against the course page")
	}
	if _, err := os.Stat(filepath.Join(dir, "dist", "blog", "trip", "cover.jpg")); err != nil {
		t.Errorf("bundle file not written where the URL points: %v", err)
	}
}
