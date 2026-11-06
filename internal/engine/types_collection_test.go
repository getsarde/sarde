package engine

import "testing"

// Without per-language views (a single-language site) every accessor returns
// the shared field; with them, the language's own value, and a missing
// language falls back to the shared field too.
func TestCollection_LanguageAccessors(t *testing.T) {
	post := &Page{PageIdentity: PageIdentity{Title: "Post", Kind: KindPage}}
	index := &Page{PageIdentity: PageIdentity{Title: "Blog", Kind: KindSection}}
	col := &Collection{Title: "Blog", Pages: []*Page{index, post}, Featured: []*Page{post}, IndexPage: index}

	if got := col.PagesFor("en"); len(got) != 2 || &got[0] != &col.Pages[0] {
		t.Error("PagesFor without views must return Pages itself")
	}
	if got := col.ListPages(""); len(got) != 1 || got[0] != post {
		t.Errorf("ListPages = %v, want the post only", got)
	}
	if col.IndexFor("fr") != index || col.TitleFor("fr") != "Blog" || len(col.FeaturedFor("fr")) != 1 {
		t.Error("accessors without views must return the shared fields")
	}

	frIndex := &Page{PageIdentity: PageIdentity{Title: "Carnet", Kind: KindSection}}
	col.LangPages = map[string][]*Page{"fr": {frIndex}}
	col.LangIndex = map[string]*Page{"fr": frIndex, "de": nil}
	col.LangFeatured = map[string][]*Page{"fr": nil}
	if col.TitleFor("fr") != "Carnet" || col.IndexFor("fr") != frIndex {
		t.Error("TitleFor/IndexFor(fr) must use the French index")
	}
	if col.IndexFor("de") != nil || col.TitleFor("de") != "Blog" {
		t.Error("a language without an index page has no index and keeps the shared title")
	}
	if len(col.ListPages("fr")) != 0 || len(col.FeaturedFor("fr")) != 0 {
		t.Error("the French list holds only the French index page")
	}

	var none *Collection
	if none.PagesFor("en") != nil || none.IndexFor("en") != nil || none.TitleFor("en") != "" || none.LabNavTree("en", "/x/") != nil {
		t.Error("accessors on a nil collection must return zero values")
	}
}

func TestSection_ListingSection(t *testing.T) {
	root := &Section{Slug: "docs"}
	outer := &Section{Slug: "outer", Transparent: true, Parent: root}
	inner := &Section{Slug: "inner", Transparent: true, Parent: outer}
	plain := &Section{Slug: "plain", Parent: root}
	orphan := &Section{Slug: "orphan", Transparent: true}

	tests := []struct {
		sec, want *Section
	}{
		{inner, root},
		{outer, root},
		{plain, plain},
		{root, root},
		{orphan, orphan},
		{nil, nil},
	}
	for _, tt := range tests {
		if got := tt.sec.ListingSection(); got != tt.want {
			t.Errorf("ListingSection(%v) = %v, want %v", tt.sec, got, tt.want)
		}
	}
}
