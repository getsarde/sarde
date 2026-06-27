package linkrender

import (
	"testing"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/links"
)

// buildSiteAbsoluteRenderer wires a Renderer with a real page index and a
// base-path URL resolver for exercising resolveSiteAbsolute via ResolveHref.
func buildSiteAbsoluteRenderer(t *testing.T) (*Renderer, *links.LinkGraph) {
	t.Helper()

	docsCol := &engine.Collection{Name: "docs", Config: &engine.CollectionConfig{}}
	targetPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/auth/", Permalink: "/docs/guide/auth/", Slug: "auth"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/auth.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}
	currentPage := &engine.Page{
		PageIdentity:      engine.PageIdentity{RelPermalink: "/docs/guide/intro/", Permalink: "/docs/guide/intro/", Slug: "intro"},
		PageI18n:          engine.PageI18n{Lang: "en", LangRelPath: "docs/guide/intro.md"},
		PageRelationships: engine.PageRelationships{Collection: docsCol},
	}

	idx := content.BuildPageIndex([]*engine.Page{targetPage, currentPage})
	graph := links.NewLinkGraph()

	r := &Renderer{
		CurrentPage: currentPage,
		PageIndex:   idx,
		URLResolver: &engine.URLResolver{BasePath: "/web-course/"},
		LinkGraph:   graph,
	}
	return r, graph
}

func lastRef(t *testing.T, graph *links.LinkGraph) links.LinkRef {
	t.Helper()
	refs := graph.Refs()
	if len(refs) == 0 {
		t.Fatal("expected a recorded link ref, got none")
	}
	return refs[len(refs)-1]
}

func TestResolveSiteAbsolute_Found(t *testing.T) {
	r, graph := buildSiteAbsoluteRenderer(t)

	// Extension-less site-absolute href to an existing page.
	got := r.ResolveHref("/docs/guide/auth")
	if want := "/web-course/docs/guide/auth/"; got != want {
		t.Errorf("ResolveHref = %q, want %q", got, want)
	}
	if ref := lastRef(t, graph); ref.Status != links.StatusOK {
		t.Errorf("status = %v, want StatusOK", ref.Status)
	}
}

func TestResolveSiteAbsolute_FoundWithFragment(t *testing.T) {
	r, graph := buildSiteAbsoluteRenderer(t)

	got := r.ResolveHref("/docs/guide/auth#api-keys")
	if want := "/web-course/docs/guide/auth/#api-keys"; got != want {
		t.Errorf("ResolveHref = %q, want %q", got, want)
	}
	// A fragment defers an anchor check rather than recording a ref immediately.
	if n := graph.Len(); n != 0 {
		t.Errorf("expected no immediate ref for fragmented link (deferred anchor), got %d", n)
	}
	if got := len(r.PendingAnchors); got != 1 {
		t.Errorf("expected 1 pending anchor check, got %d", got)
	}
}

func TestResolveSiteAbsolute_StaticAsset(t *testing.T) {
	r, graph := buildSiteAbsoluteRenderer(t)

	// A file with an extension is a static asset: base-path applied, never flagged.
	got := r.ResolveHref("/img/logo.png")
	if want := "/web-course/img/logo.png"; got != want {
		t.Errorf("ResolveHref = %q, want %q", got, want)
	}
	if ref := lastRef(t, graph); ref.Status != links.StatusExternal {
		t.Errorf("status = %v, want StatusExternal (asset never flagged)", ref.Status)
	}
}

func TestResolveSiteAbsolute_PageLikeNotFound(t *testing.T) {
	r, graph := buildSiteAbsoluteRenderer(t)

	// Extension-less, page-like, but no matching page: base-path applied AND
	// recorded as unverified so the link checker surfaces it.
	got := r.ResolveHref("/docs/guide/ath")
	if want := "/web-course/docs/guide/ath/"; got != want {
		t.Errorf("ResolveHref = %q, want %q", got, want)
	}
	if ref := lastRef(t, graph); ref.Status != links.StatusUnverified {
		t.Errorf("status = %v, want StatusUnverified", ref.Status)
	}
}

func TestResolveSiteAbsolute_SiteRoot(t *testing.T) {
	r, graph := buildSiteAbsoluteRenderer(t)

	// The site root is never flagged even though it has no file extension.
	got := r.ResolveHref("/")
	if want := "/web-course/"; got != want {
		t.Errorf("ResolveHref = %q, want %q", got, want)
	}
	if ref := lastRef(t, graph); ref.Status != links.StatusExternal {
		t.Errorf("status = %v, want StatusExternal (root never flagged)", ref.Status)
	}
}

func TestResolveSiteAbsolute_TrulyExternalUnaffected(t *testing.T) {
	r, graph := buildSiteAbsoluteRenderer(t)

	got := r.ResolveHref("https://example.com/page")
	if want := "https://example.com/page"; got != want {
		t.Errorf("ResolveHref = %q, want external unchanged %q", got, want)
	}
	if ref := lastRef(t, graph); ref.Status != links.StatusExternal {
		t.Errorf("status = %v, want StatusExternal", ref.Status)
	}
}

func TestResolveSiteRoot_BasePathOnly(t *testing.T) {
	r, graph := buildSiteAbsoluteRenderer(t)
	r.SiteRootEscapePrefix = "site:"

	got := r.ResolveHref("site:/pricing")
	if want := "/web-course/pricing/"; got != want {
		t.Errorf("ResolveHref = %q, want %q (base path, no lane segments)", got, want)
	}
	if ref := lastRef(t, graph); ref.Status != links.StatusExternal {
		t.Errorf("status = %v, want StatusExternal (escape never flagged)", ref.Status)
	}
}

func TestResolveSiteRoot_WithFragment(t *testing.T) {
	r, _ := buildSiteAbsoluteRenderer(t)
	r.SiteRootEscapePrefix = "site:"

	// Query then fragment ordering: path?query#fragment.
	got := r.ResolveHref("site:/pricing?ref=nav#plans")
	if want := "/web-course/pricing/?ref=nav#plans"; got != want {
		t.Errorf("ResolveHref = %q, want %q", got, want)
	}
	// A site-root escape is never deferred for anchor validation.
	if got := len(r.PendingAnchors); got != 0 {
		t.Errorf("expected no pending anchor checks for site-root escape, got %d", got)
	}
}

func TestResolveSiteRoot_DisabledWhenPrefixEmpty(t *testing.T) {
	r, _ := buildSiteAbsoluteRenderer(t) // SiteRootEscapePrefix defaults to ""

	// With the escape disabled, "site:/x" is an unrecognized scheme and passes through.
	got := r.ResolveHref("site:/pricing")
	if want := "site:/pricing"; got != want {
		t.Errorf("ResolveHref = %q, want %q (passthrough when escape disabled)", got, want)
	}
}

func TestResolveSiteRoot_LaneFreeWithI18n(t *testing.T) {
	graph := links.NewLinkGraph()
	r := &Renderer{
		CurrentPage: &engine.Page{
			PageIdentity: engine.PageIdentity{RelPermalink: "/fr/docs/guide/intro/"},
			PageI18n:     engine.PageI18n{Lang: "fr"},
		},
		PageIndex: content.BuildPageIndex(nil),
		URLResolver: &engine.URLResolver{
			BasePath:    "/",
			I18nEnabled: true,
			DefaultLang: "en",
			Strategy:    "prefix-except-default",
			Languages:   map[string]bool{"en": true, "fr": true},
		},
		LinkGraph:            graph,
		SiteRootEscapePrefix: "site:",
	}

	// A French page's site-root escape must NOT pick up a /fr/ prefix.
	got := r.ResolveHref("site:/pricing")
	if want := "/pricing/"; got != want {
		t.Errorf("ResolveHref = %q, want %q (lane-free, no /fr/ prefix)", got, want)
	}
}
