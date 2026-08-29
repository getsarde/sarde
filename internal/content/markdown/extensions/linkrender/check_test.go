package linkrender

import (
	"testing"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/links"
)

func checkFixture(t *testing.T) (source, target *engine.Page, idx *content.PageIndex, resolver *engine.URLResolver, ctx ResolveContext) {
	t.Helper()
	docsCol := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}
	target = &engine.Page{
		PageIdentity:      engine.PageIdentity{Slug: "auth", RelPermalink: "/docs/guide/auth/", Permalink: "/docs/guide/auth/"},
		PageI18n:          engine.PageI18n{LangRelPath: "docs/guide/auth.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	source = &engine.Page{
		PageIdentity:      engine.PageIdentity{Slug: "intro", RelPermalink: "/docs/guide/intro/", Permalink: "/docs/guide/intro/"},
		PageI18n:          engine.PageI18n{LangRelPath: "docs/guide/intro.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	idx = content.BuildPageIndex([]*engine.Page{source, target})
	resolver = &engine.URLResolver{BasePath: "/", DefaultLang: "en"}
	ctx = ResolveContext{
		PageIndex:   idx,
		URLResolver: resolver.URL,
		Collections: map[string]*engine.Collection{"docs": docsCol},
	}
	return source, target, idx, resolver, ctx
}

func TestCheckHref_ContentRootResolvesInCollection(t *testing.T) {
	source, _, idx, resolver, ctx := checkFixture(t)

	result := CheckHref("/guide/auth", source, ctx, idx, resolver, "")
	if result.Status != links.StatusOK {
		t.Errorf("content-root link should resolve in collection, got status %v", result.Status)
	}
	if result.TargetPermalink != "/docs/guide/auth/" {
		t.Errorf("TargetPermalink = %q, want /docs/guide/auth/", result.TargetPermalink)
	}
}

func TestCheckHref_RelativeResolvesInDir(t *testing.T) {
	source, _, idx, resolver, ctx := checkFixture(t)

	result := CheckHref("./auth.md", source, ctx, idx, resolver, "")
	if result.Status != links.StatusOK {
		t.Errorf("relative link should resolve, got status %v", result.Status)
	}
	if result.TargetPermalink != "/docs/guide/auth/" {
		t.Errorf("TargetPermalink = %q, want /docs/guide/auth/", result.TargetPermalink)
	}
}

func TestCheckHref_SiteAbsoluteResolves(t *testing.T) {
	source, _, idx, resolver, ctx := checkFixture(t)

	result := CheckHref("/docs/guide/auth", source, ctx, idx, resolver, "")
	if result.Status != links.StatusOK {
		t.Errorf("site-absolute link should resolve, got status %v", result.Status)
	}
}

func TestCheckHref_UnresolvedPageLike(t *testing.T) {
	source, _, idx, resolver, ctx := checkFixture(t)

	result := CheckHref("/nonexistent/page", source, ctx, idx, resolver, "")
	if result.Status != links.StatusUnverified {
		t.Errorf("unresolved page-like should be StatusUnverified, got %v", result.Status)
	}
}

func TestCheckHref_StaticAsset(t *testing.T) {
	source, _, idx, resolver, ctx := checkFixture(t)

	result := CheckHref("/favicon.ico", source, ctx, idx, resolver, "")
	if result.Status != links.StatusExternal {
		t.Errorf("static asset should be StatusExternal, got %v", result.Status)
	}
}

func TestCheckHref_External(t *testing.T) {
	source, _, idx, resolver, ctx := checkFixture(t)

	result := CheckHref("https://example.com/page", source, ctx, idx, resolver, "")
	if result.Status != links.StatusExternal {
		t.Errorf("external URL should be StatusExternal, got %v", result.Status)
	}
}

func TestCheckHref_BrokenRelative(t *testing.T) {
	source, _, idx, resolver, ctx := checkFixture(t)

	result := CheckHref("./missing.md", source, ctx, idx, resolver, "")
	if result.Status != links.StatusBrokenTarget {
		t.Errorf("missing relative target should be StatusBrokenTarget, got %v", result.Status)
	}
}

func TestCheckHref_BareMarkdownIsAmbiguous(t *testing.T) {
	source, _, idx, resolver, ctx := checkFixture(t)

	result := CheckHref("missing.md", source, ctx, idx, resolver, "")
	if result.Status != links.StatusAmbiguous {
		t.Errorf("bare name.md should be StatusAmbiguous, got %v", result.Status)
	}
}

func TestCheckHref_FragmentTracked(t *testing.T) {
	source, _, idx, resolver, ctx := checkFixture(t)

	result := CheckHref("./auth.md#setup", source, ctx, idx, resolver, "")
	if result.Status != links.StatusOK {
		t.Errorf("link should resolve, got status %v", result.Status)
	}
	if !result.HasFragment {
		t.Error("HasFragment should be true for links with #fragment")
	}
}

func TestCheckHref_SiteRootEscape(t *testing.T) {
	source, _, idx, resolver, ctx := checkFixture(t)

	result := CheckHref("site:/pricing", source, ctx, idx, resolver, "site:")
	if result.Status != links.StatusExternal {
		t.Errorf("site-root escape should be StatusExternal, got %v", result.Status)
	}
}
