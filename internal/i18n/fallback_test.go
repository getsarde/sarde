package i18n

import (
	"sort"
	"testing"

	"github.com/getsarde/sarde/internal/engine"
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

// TestGenerateFallbacks_DeterministicOrder guards against the nondeterministic
// broken_anchor regression: GenerateFallbacks used to iterate defaultPages in
// Go map order, so the returned slice (and downstream last-write-wins page-index
// population) varied between runs. With ≥6 pages, an unsorted iteration would
// almost certainly produce a different order across 50 calls.
func TestGenerateFallbacks_DeterministicOrder(t *testing.T) {
	paths := []string{
		"docs/intro.md",
		"docs/install.md",
		"docs/config.md",
		"docs/api.md",
		"blog/first.md",
		"blog/second.md",
		"about.md",
		"contact.md",
	}
	pages := make([]*engine.Page, 0, len(paths))
	for _, p := range paths {
		rel := "/" + p[:len(p)-len(".md")] + "/"
		pages = append(pages, &engine.Page{
			PageIdentity: engine.PageIdentity{RelPermalink: rel, Permalink: rel},
			PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: p},
		})
	}

	orderOf := func(fbs []*engine.Page) []string {
		out := make([]string, len(fbs))
		for i, fb := range fbs {
			out[i] = fb.LangRelPath
		}
		return out
	}

	first := orderOf(GenerateFallbacks(pages, []string{"en", "fr"}, "en", defaultOpts))
	if len(first) != len(paths) {
		t.Fatalf("expected %d fallbacks, got %d", len(paths), len(first))
	}

	// The order must equal the sorted LangRelPath order (the deterministic key).
	wantSorted := append([]string(nil), paths...)
	sort.Strings(wantSorted)
	for i := range wantSorted {
		if first[i] != wantSorted[i] {
			t.Fatalf("fallback order not sorted: got %v, want %v", first, wantSorted)
		}
	}

	// And it must be identical across many invocations.
	for run := 0; run < 50; run++ {
		got := orderOf(GenerateFallbacks(pages, []string{"en", "fr"}, "en", defaultOpts))
		if len(got) != len(first) {
			t.Fatalf("run %d: length changed: got %d, want %d", run, len(got), len(first))
		}
		for i := range got {
			if got[i] != first[i] {
				t.Fatalf("run %d: order changed at %d: got %v, want %v", run, i, got, first)
			}
		}
	}
}

// Fallback clones must own an independent Params tree: BeforeRender plugins
// (seo, socialcards) write into Params (including nested maps) during the
// parallel render phase, so a shared map is a concurrent-map-write crash.
func TestGenerateFallbacks_ParamsDeepCopied(t *testing.T) {
	src := &engine.Page{
		PageIdentity: engine.PageIdentity{Title: "API Reference", RelPermalink: "/docs/api/", Permalink: "/docs/api/"},
		PageI18n:     engine.PageI18n{Lang: "en", LangRelPath: "docs/api.md"},
		Params: map[string]any{
			"seo":  map[string]any{"og_title": "orig"},
			"tags": []any{"a", "b"},
		},
	}

	fallbacks := GenerateFallbacks([]*engine.Page{src}, []string{"en", "fr"}, "en", defaultOpts)
	if len(fallbacks) != 1 {
		t.Fatalf("expected 1 fallback, got %d", len(fallbacks))
	}
	fb := fallbacks[0]

	// Mutating the clone's top-level and nested maps must not touch the source.
	fb.Params["new"] = true
	fb.Params["seo"].(map[string]any)["og_title"] = "mutated"

	if _, ok := src.Params["new"]; ok {
		t.Error("top-level Params map is shared between source and fallback")
	}
	if got := src.Params["seo"].(map[string]any)["og_title"]; got != "orig" {
		t.Errorf("nested seo map is shared: source og_title = %q, want %q", got, "orig")
	}
}
