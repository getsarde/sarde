package build

import (
	"testing"

	"github.com/getsarde/sarde/internal/engine"
)

// newSubPathResolver serves the site under /sub/ with English as the default
// language and French under /fr/.
func newSubPathResolver() *engine.URLResolver {
	return &engine.URLResolver{
		BasePath:    "/sub/",
		BaseURL:     "https://example.com",
		I18nEnabled: true,
		DefaultLang: "en",
		Strategy:    "prefix-except-default",
		Languages:   map[string]bool{"en": true, "fr": true},
	}
}

// A nav.yaml url that names another site keeps its URL; a site path gets the
// base path and the language prefix.
func TestResolveNavNodes_KeepsAbsoluteURLs(t *testing.T) {
	root := &engine.NavNode{Children: []*engine.NavNode{
		{URL: "https://example.com/x"},
		{URL: "//cdn.example.com/x"},
		{URL: "mailto:a@b.c"},
		{URL: "/docs/guide/"},
	}}
	resolveNavNodes(newSubPathResolver(), root, "fr", "")

	want := []string{"https://example.com/x", "//cdn.example.com/x", "mailto:a@b.c", "/sub/fr/docs/guide/"}
	for i, w := range want {
		if got := root.Children[i].URL; got != w {
			t.Errorf("child %d URL = %q, want %q", i, got, w)
		}
	}
}

// Custom prev/next links, header links and asset URLs that name another
// location pass through; site paths still resolve.
func TestResolveRouteAssets_KeepsAbsoluteLinks(t *testing.T) {
	rd := &engine.RouteData{}
	rd.Page = &engine.Page{}
	rd.Pagination = &engine.PaginationLinks{
		Prev: &engine.PaginationLink{URL: "https://example.com/prev/"},
		Next: &engine.PaginationLink{URL: "/docs/next/"},
	}
	rd.GlobalNav = &engine.GlobalNav{Items: []engine.GlobalNavItem{
		{URL: "mailto:a@b.c"},
		{URL: "/blog/"},
	}}
	rd.Styles = []string{"//cdn.example.com/a.css", "/assets/site.css"}

	resolveRouteAssets(newSubPathResolver(), rd)

	checks := []struct{ name, got, want string }{
		{"prev", rd.Pagination.Prev.URL, "https://example.com/prev/"},
		{"next", rd.Pagination.Next.URL, "/sub/docs/next/"},
		{"header mailto", rd.GlobalNav.Items[0].URL, "mailto:a@b.c"},
		{"header site path", rd.GlobalNav.Items[1].URL, "/sub/blog/"},
		{"cdn style", rd.Styles[0], "//cdn.example.com/a.css"},
		{"site style", rd.Styles[1], "/sub/assets/site.css"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
}

// Each translation link gets an absolute URL in its own language, so the
// English entry on a French page points at the English URL.
func TestResolveRouteAssets_TranslationAbsURL(t *testing.T) {
	for _, tc := range []struct{ name, basePath, wantFR, wantEN string }{
		{"root", "/", "https://example.com/fr/docs/guide/", "https://example.com/docs/guide/"},
		{"base path", "/sub/", "https://example.com/sub/fr/docs/guide/", "https://example.com/sub/docs/guide/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newSubPathResolver()
			r.BasePath = tc.basePath
			rd := &engine.RouteData{}
			rd.Page = &engine.Page{}
			rd.Page.Lang = "fr"
			links := func() []engine.TranslationLink {
				return []engine.TranslationLink{{Lang: "fr", URL: "/docs/guide/"}, {Lang: "en", URL: "/docs/guide/"}}
			}
			rd.Translations = links()
			rd.AllTranslations = links()

			resolveRouteAssets(r, rd)

			for name, got := range map[string][]engine.TranslationLink{"Translations": rd.Translations, "AllTranslations": rd.AllTranslations} {
				if got[0].AbsURL != tc.wantFR || got[1].AbsURL != tc.wantEN {
					t.Errorf("%s AbsURL = %q, %q; want %q, %q", name, got[0].AbsURL, got[1].AbsURL, tc.wantFR, tc.wantEN)
				}
			}
		})
	}
}
