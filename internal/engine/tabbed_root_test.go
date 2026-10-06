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

func TestTabbedRoot_UntabbedAndVersioned(t *testing.T) {
	col, root := tabbedFixture("")
	col.IsTabbed = false
	if IsTabbedRoot(root) {
		t.Error("untabbed collection: IsTabbedRoot = true")
	}

	// Versioned tabbed collections fill CompositeTabSets and leave Tabs empty.
	col.IsTabbed = true
	col.Tabs = nil
	col.CompositeTabSets = map[string][]*DocsTab{"": {{Permalink: "/courses/a/"}}}
	if IsTabbedRoot(root) || TabbedRootRedirect(root) != "" {
		t.Error("versioned tabbed collection: root matched")
	}
}
