package i18n

import (
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

var defaultOpts = FallbackOptions{SiteFallback: "default"}

func TestGenerateFallbacks_CreatesMissing(t *testing.T) {
	enPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Getting Started", RelPermalink: "/docs/getting-started/", Permalink: "/docs/getting-started/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/getting-started.md"},
	}
	frPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "Premiers pas", RelPermalink: "/docs/getting-started/", Permalink: "/fr/docs/getting-started/"},
		PageI18n:     engine.PageI18n{Lang: "fr", LangRelPath: "docs/getting-started.md"},
	}
	// en has an extra page that fr doesn't have
	enOnly := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "API Reference", RelPermalink: "/docs/api/", Permalink: "/docs/api/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/api.md"},
	}

	pages := []*engine.Page{enPage, frPage, enOnly}
	fallbacks := GenerateFallbacks(pages, []string{"en", "fr"}, "en", defaultOpts)

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
	if fb.RelPermalink != "/docs/api/" {
		t.Errorf("fallback RelPermalink = %q, want %q (lang-free)", fb.RelPermalink, "/docs/api/")
	}
	if fb.Title != "API Reference" {
		t.Errorf("fallback Title = %q, want %q", fb.Title, "API Reference")
	}
}

func TestGenerateFallbacks_SingleLanguage(t *testing.T) {
	pages := []*engine.Page{
		{PageI18n: engine.PageI18n{Lang: "en", LangRelPath: "docs/api.md"}},
	}

	fallbacks := GenerateFallbacks(pages, []string{"en"}, "en", defaultOpts)
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
	fallbacks := GenerateFallbacks(pages, []string{"en", "fr", "ar"}, "en", defaultOpts)

	if len(fallbacks) != 2 {
		t.Fatalf("expected 2 fallbacks, got %d", len(fallbacks))
	}

	langs := map[string]bool{}
	for _, fb := range fallbacks {
		langs[fb.Lang] = true
		if !fb.IsFallback {
			t.Errorf("fallback for %s should have IsFallback = true", fb.Lang)
		}
		if fb.RelPermalink != "/docs/intro/" {
			t.Errorf("fallback %s RelPermalink = %q, want /docs/intro/ (lang-free)", fb.Lang, fb.RelPermalink)
		}
	}
	if !langs["fr"] || !langs["ar"] {
		t.Errorf("expected fallbacks for fr and ar, got %v", langs)
	}
}

func TestGenerateFallbacks_NoFallbackWhenTranslationExists(t *testing.T) {
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{RelPermalink: "/about/", Permalink: "/about/"}, PageI18n: engine.PageI18n{Lang: "en", LangRelPath: "about.md"}},
		{PageIdentity: engine.PageIdentity{RelPermalink: "/about/", Permalink: "/fr/about/"}, PageI18n: engine.PageI18n{Lang: "fr", LangRelPath: "about.md"}},
	}

	fallbacks := GenerateFallbacks(pages, []string{"en", "fr"}, "en", defaultOpts)
	if len(fallbacks) != 0 {
		t.Errorf("expected 0 fallbacks when translations exist, got %d", len(fallbacks))
	}
}

func TestGenerateFallbacks_SiteOmitPolicy(t *testing.T) {
	enPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/intro/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/intro.md"},
	}

	fallbacks := GenerateFallbacks(
		[]*engine.Page{enPage},
		[]string{"en", "fr"},
		"en",
		FallbackOptions{SiteFallback: "omit"},
	)
	if len(fallbacks) != 0 {
		t.Errorf("expected 0 fallbacks with omit policy, got %d", len(fallbacks))
	}
}

func TestGenerateFallbacks_CollectionOmitOverride(t *testing.T) {
	blogCol := &engine.Collection{Name: "blog"}
	docsCol := &engine.Collection{Name: "docs"}

	enBlog := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/blog/post/"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "blog/post.md"},
		PageRelationships: engine.PageRelationships{Collection: blogCol},
	}
	enDocs := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}

	fallbacks := GenerateFallbacks(
		[]*engine.Page{enBlog, enDocs},
		[]string{"en", "fr"},
		"en",
		FallbackOptions{
			SiteFallback:       "default",
			CollectionFallback: map[string]string{"blog": "omit"},
		},
	)

	if len(fallbacks) != 1 {
		t.Fatalf("expected 1 fallback (docs only), got %d", len(fallbacks))
	}
	if fallbacks[0].LangRelPath != "docs/guide.md" {
		t.Errorf("wrong page got fallback: %s", fallbacks[0].LangRelPath)
	}
}

func TestGenerateFallbacks_CollectionDefaultOverridesSiteOmit(t *testing.T) {
	docsCol := &engine.Collection{Name: "docs"}

	enDocs := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}

	fallbacks := GenerateFallbacks(
		[]*engine.Page{enDocs},
		[]string{"en", "fr"},
		"en",
		FallbackOptions{
			SiteFallback:       "omit",
			CollectionFallback: map[string]string{"docs": "default"},
		},
	)

	if len(fallbacks) != 1 {
		t.Fatalf("expected 1 fallback (docs overrides site omit), got %d", len(fallbacks))
	}
}

func TestGenerateFallbacks_RelPermalinkIsLangFree(t *testing.T) {
	enPage := &engine.Page{
		PageIdentity: engine.PageIdentity{RelPermalink: "/docs/api/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/api.md"},
	}

	fallbacks := GenerateFallbacks(
		[]*engine.Page{enPage},
		[]string{"en", "fr"},
		"en",
		defaultOpts,
	)

	if len(fallbacks) != 1 {
		t.Fatalf("expected 1 fallback, got %d", len(fallbacks))
	}
	if fallbacks[0].RelPermalink != "/docs/api/" {
		t.Errorf("RelPermalink = %q, want /docs/api/ (lang-free)", fallbacks[0].RelPermalink)
	}
}
