package engine

import "testing"

func tabbedFixture(body string) (*Collection, *Page) {
	col := &Collection{
		Name:     "courses",
		IsTabbed: true,
		Tabs:     []*DocsTab{{Slug: "a", Permalink: "/courses/a/"}, {Slug: "b", Permalink: "/courses/b/"}},
	}
	root := &Page{
		PageIdentity:      PageIdentity{Kind: KindSection, RelPermalink: "/courses/"},
		PageContent:       PageContent{RawContent: body},
		PageRelationships: PageRelationships{Collection: col},
	}
	col.IndexPage = root
	return col, root
}

func TestTabbedRoot_EmptyBodyRedirects(t *testing.T) {
	_, root := tabbedFixture("\n  \n")
	if !IsTabbedRoot(root) {
		t.Fatal("IsTabbedRoot = false, want true")
	}
	if got := TabbedRootRedirect(root); got != "/courses/a/" {
		t.Errorf("TabbedRootRedirect = %q, want /courses/a/", got)
	}
	if IsTabbedLanding(root) {
		t.Error("IsTabbedLanding = true for a root without a body")
	}
}

func TestTabbedRoot_BodyMakesLanding(t *testing.T) {
	_, root := tabbedFixture("Pick a course to start.")
	if got := TabbedRootRedirect(root); got != "" {
		t.Errorf("TabbedRootRedirect = %q, want empty for a root with a body", got)
	}
	if !IsTabbedLanding(root) {
		t.Error("IsTabbedLanding = false, want true")
	}
}

// A translation or a page rebuilt by the dev server is a different pointer
// with the same lane-free RelPermalink; it must be treated as the root too.
func TestTabbedRoot_MatchesByRelPermalinkNotPointer(t *testing.T) {
	col, _ := tabbedFixture("")
	other := &Page{
		PageIdentity:      PageIdentity{Kind: KindSection, RelPermalink: "/courses/"},
		PageRelationships: PageRelationships{Collection: col},
	}
	if got := TabbedRootRedirect(other); got != "/courses/a/" {
		t.Errorf("TabbedRootRedirect = %q, want /courses/a/", got)
	}
}

func TestTabbedRoot_NotRoot(t *testing.T) {
	col, _ := tabbedFixture("")
	tabPage := &Page{
		PageIdentity:      PageIdentity{Kind: KindSection, RelPermalink: "/courses/a/"},
		PageRelationships: PageRelationships{Collection: col},
	}
	leaf := &Page{
		PageIdentity:      PageIdentity{Kind: KindPage, RelPermalink: "/courses/"},
		PageRelationships: PageRelationships{Collection: col},
	}
	for name, p := range map[string]*Page{"tab section": tabPage, "non-section kind": leaf, "nil": nil} {
		if IsTabbedRoot(p) || TabbedRootRedirect(p) != "" || IsTabbedLanding(p) {
			t.Errorf("%s: matched as tabbed root", name)
		}
	}
}

func TestTabbedRoot_Untabbed(t *testing.T) {
	col, root := tabbedFixture("")
	col.IsTabbed = false
	if IsTabbedRoot(root) {
		t.Error("untabbed collection: IsTabbedRoot = true")
	}
}

// A collection that is both versioned and tabbed fills CompositeTabSets and
// leaves Tabs empty. Every version has its own root page with the same
// lane-free RelPermalink, and each sees its own version's tabs.
func TestTabbedRoot_Versioned(t *testing.T) {
	col, latest := tabbedFixture("")
	col.Tabs = nil
	col.CompositeTabSets = map[string][]*DocsTab{
		LangVersionKey("", "v2"): {{Slug: "guide", Permalink: "/courses/guide/"}, {Slug: "api", Permalink: "/courses/api/"}},
		LangVersionKey("", "v1"): {{Slug: "api", Permalink: "/courses/api/"}},
	}
	latest.Version = "v2"
	old := &Page{
		PageIdentity:      PageIdentity{Kind: KindSection, RelPermalink: "/courses/"},
		PageVersioning:    PageVersioning{Version: "v1"},
		PageRelationships: PageRelationships{Collection: col},
	}

	if got := TabbedRootRedirect(latest); got != "/courses/guide/" {
		t.Errorf("latest root: TabbedRootRedirect = %q, want /courses/guide/", got)
	}
	if got := TabbedRootRedirect(old); got != "/courses/api/" {
		t.Errorf("v1 root: TabbedRootRedirect = %q, want its own first tab /courses/api/", got)
	}

	old.RawContent = "Pick a section."
	if !IsTabbedLanding(old) || TabbedRootRedirect(old) != "" {
		t.Error("v1 root with a body: want a landing page and no redirect")
	}

	// A section below the root is not a root, and a version with no tabs
	// has a plain root.
	section := &Page{
		PageIdentity:      PageIdentity{Kind: KindSection, RelPermalink: "/courses/guide/"},
		PageVersioning:    PageVersioning{Version: "v2"},
		PageRelationships: PageRelationships{Collection: col},
	}
	untabbed := &Page{
		PageIdentity:      PageIdentity{Kind: KindSection, RelPermalink: "/courses/"},
		PageVersioning:    PageVersioning{Version: "v0"},
		PageRelationships: PageRelationships{Collection: col},
	}
	for name, p := range map[string]*Page{"section": section, "version without tabs": untabbed} {
		if IsTabbedRoot(p) {
			t.Errorf("%s: matched as tabbed root", name)
		}
	}
}

// The French root forwards to the first tab in French order.
func TestTabbedRoot_RedirectsToFirstTabOfPageLanguage(t *testing.T) {
	col, root := tabbedFixture("")
	col.LangTabs = map[string][]*DocsTab{
		"en": col.Tabs,
		"fr": {{Slug: "b", Permalink: "/courses/b/"}, {Slug: "a", Permalink: "/courses/a/"}},
	}
	root.Lang = "fr"
	if got := TabbedRootRedirect(root); got != "/courses/b/" {
		t.Errorf("TabbedRootRedirect = %q, want /courses/b/", got)
	}
	root.Lang = "en"
	if got := TabbedRootRedirect(root); got != "/courses/a/" {
		t.Errorf("TabbedRootRedirect = %q, want /courses/a/", got)
	}
}
