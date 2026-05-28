package i18n

import (
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

func TestGenerateFallbacks_CreatesMissing(t *testing.T) {
	enPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Getting Started", RelPermalink: "/docs/getting-started/", Permalink: "/docs/getting-started/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/getting-started.md"},
	}
	frPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Premiers pas", RelPermalink: "/fr/docs/getting-started/", Permalink: "/fr/docs/getting-started/"},
		PageI18n:     engine.PageI18n{Lang: "fr", LangRelPath: "docs/getting-started.md"},
	}
	// en has an extra page that fr doesn't have
	enOnly := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "API Reference", RelPermalink: "/docs/api/", Permalink: "/docs/api/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/api.md"},
	}

	pages := []*engine.Page{enPage, frPage, enOnly}
	fallbacks := GenerateFallbacks(pages, []string{"en", "fr"}, "en")

	if len(fallbacks) != 1 {
		t.Fatalf("expected 1 fallback, got %d", len(fallbacks))
	}

	fb := fallbacks[0]
	if fb.Lang != "fr" {
		t.Errorf("fallback Lang = %q, want %q", fb.Lang, "fr")
	}
	if fb.LangRelPath != "docs/api.md" {
		t.Errorf("fallback LangRelPath = %q, want %q", fb.LangRelPath, "docs/api.md")
	}
	if !fb.IsFallback {
		t.Error("fallback should have IsFallback = true")
	}
	if fb.RelPermalink != "/fr/docs/api/" {
		t.Errorf("fallback RelPermalink = %q, want %q", fb.RelPermalink, "/fr/docs/api/")
	}
	if fb.Title != "API Reference" {
		t.Errorf("fallback Title = %q, want %q", fb.Title, "API Reference")
	}
}

func TestGenerateFallbacks_SingleLanguage(t *testing.T) {
	pages := []*engine.Page{
		{PageI18n: engine.PageI18n{Lang: "en", LangRelPath: "docs/api.md"}},
	}

	fallbacks := GenerateFallbacks(pages, []string{"en"}, "en")
	if len(fallbacks) != 0 {
		t.Errorf("expected 0 fallbacks for single language, got %d", len(fallbacks))
	}
}

func TestGenerateFallbacks_ThreeLanguages(t *testing.T) {
	enPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/intro/", Permalink: "/docs/intro/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
	}

	pages := []*engine.Page{enPage}
	fallbacks := GenerateFallbacks(pages, []string{"en", "fr", "ar"}, "en")

	// Should create fallbacks for fr and ar
	if len(fallbacks) != 2 {
		t.Fatalf("expected 2 fallbacks, got %d", len(fallbacks))
	}

	langs := map[string]bool{}
	for _, fb := range fallbacks {
		langs[fb.Lang] = true
		if !fb.IsFallback {
			t.Errorf("fallback for %s should have IsFallback = true", fb.Lang)
		}
	}
	if !langs["fr"] || !langs["ar"] {
		t.Errorf("expected fallbacks for fr and ar, got %v", langs)
	}
}

func TestGenerateFallbacks_NoFallbackWhenTranslationExists(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{RelPermalink: "/about/", Permalink: "/about/"}, PageI18n: engine.PageI18n{Lang: "en", LangRelPath: "about.md"}},
		{PageIdentity: engine.PageIdentity{RelPermalink: "/fr/about/", Permalink: "/fr/about/"}, PageI18n: engine.PageI18n{Lang: "fr", LangRelPath: "about.md"}},
	}

	fallbacks := GenerateFallbacks(pages, []string{"en", "fr"}, "en")
	if len(fallbacks) != 0 {
		t.Errorf("expected 0 fallbacks when translations exist, got %d", len(fallbacks))
	}
}
