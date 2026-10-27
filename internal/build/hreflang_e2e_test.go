package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// buildHreflangSite builds an en/fr site served from https://example.com under
// basePath. docs/guide exists in both languages, docs/solo only in English.
// It returns a reader for output files that reports whether the file exists.
func buildHreflangSite(t *testing.T, basePath string) func(rel string) (string, bool) {
	t.Helper()
	dir := t.TempDir()
	guide := `---
title: Guide
---
# Guide
Body.
`
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n")
	writeFixture(t, dir, "content/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/docs/guide.md", guide)
	writeFixture(t, dir, "content/docs/solo.md", "---\ntitle: Solo\n---\nOnly in English.\n")
	writeFixture(t, dir, "content/fr/_index.md", "---\ntitle: Accueil\n---\n")
	writeFixture(t, dir, "content/fr/docs/_index.md", "---\ntitle: Docs\n---\n")
	writeFixture(t, dir, "content/fr/docs/guide.md", guide)

	cfg := config.Defaults()
	cfg.Build.Minify = config.BoolPtr(false)
	cfg.Build.BasePath = basePath
	cfg.Site.URL = "https://example.com"
	cfg.I18n.DefaultLanguage = "en"
	cfg.I18n.Languages = map[string]config.LanguageConfig{
		"en": {Name: "English", Weight: 1, Dir: "ltr"},
		"fr": {Name: "French", Weight: 2, Dir: "ltr"},
	}
	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  dir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	if _, err := builder.Build(); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	return func(rel string) (string, bool) {
		data, err := os.ReadFile(filepath.Join(dir, "dist", filepath.FromSlash(rel)))
		return string(data), err == nil
	}
}

// hreflangHref returns the href of the hreflang tag for lang, or "".
func hreflangHref(html, lang string) string {
	marker := `hreflang="` + lang + `" href="`
	i := strings.Index(html, marker)
	if i < 0 {
		return ""
	}
	rest := html[i+len(marker):]
	return rest[:strings.Index(rest, `"`)]
}

// Both language versions list each language once, with that language's own
// absolute URL, and x-default points at the English page.
func TestBuild_Hreflang_OneTagPerLanguage(t *testing.T) {
	read := buildHreflangSite(t, "/")
	want := map[string]string{
		"en":        "https://example.com/docs/guide/",
		"fr":        "https://example.com/fr/docs/guide/",
		"x-default": "https://example.com/docs/guide/",
	}
	for _, rel := range []string{"docs/guide/index.html", "fr/docs/guide/index.html"} {
		html, ok := read(rel)
		if !ok {
			t.Fatalf("%s was not written", rel)
		}
		for lang, href := range want {
			if n := strings.Count(html, `rel="alternate" hreflang="`+lang+`"`); n != 1 {
				t.Errorf("%s: %d hreflang=%q tags, want 1", rel, n, lang)
			}
			if got := hreflangHref(html, lang); got != href {
				t.Errorf("%s: hreflang=%q href = %q, want %q", rel, lang, got, href)
			}
		}
	}

	// A page without a translation, and its French fallback, get no tags.
	for _, rel := range []string{"docs/solo/index.html", "fr/docs/solo/index.html"} {
		if html, ok := read(rel); ok && strings.Contains(html, "hreflang") {
			t.Errorf("%s has hreflang tags but no translation", rel)
		}
	}
}

// Under a base path the absolute URLs carry it.
func TestBuild_Hreflang_BasePath(t *testing.T) {
	read := buildHreflangSite(t, "/sub/")
	html, ok := read("fr/docs/guide/index.html")
	if !ok {
		t.Fatal("fr/docs/guide/index.html was not written")
	}
	for lang, href := range map[string]string{
		"en": "https://example.com/sub/docs/guide/",
		"fr": "https://example.com/sub/fr/docs/guide/",
	} {
		if got := hreflangHref(html, lang); got != href {
			t.Errorf("hreflang=%q href = %q, want %q", lang, got, href)
		}
	}
}
