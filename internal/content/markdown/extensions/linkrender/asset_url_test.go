package linkrender

import (
	"testing"

	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/links"
)

// An asset path written in content gets the base path and never a language
// prefix: assets exist once, at the site root. Anything else is left as
// written, and no link reference is recorded.
func TestAssetURL(t *testing.T) {
	graph := links.NewLinkGraph()
	r := &Renderer{
		CurrentPage: &engine.Page{PageI18n: engine.PageI18n{Lang: "fr"}},
		URLResolver: &engine.URLResolver{
			BasePath:    "/sub/",
			I18nEnabled: true,
			DefaultLang: "en",
			Languages:   map[string]bool{"en": true, "fr": true},
		},
		LinkGraph: graph,
	}
	for in, want := range map[string]string{
		"/images/x.svg":            "/sub/images/x.svg",
		"/images/x.svg?v=2#a":      "/sub/images/x.svg?v=2#a",
		"/sub/images/x.svg":        "/sub/images/x.svg",
		"https://cdn.x/a.png":      "https://cdn.x/a.png",
		"//cdn.x/a.png":            "//cdn.x/a.png",
		"data:image/png;base64,AA": "data:image/png;base64,AA",
		"photo.png":                "photo.png",
		"":                         "",
	} {
		if got := r.AssetURL(in); got != want {
			t.Errorf("AssetURL(%q) = %q, want %q", in, got, want)
		}
	}
	if n := graph.Len(); n != 0 {
		t.Errorf("AssetURL recorded %d link refs, want none", n)
	}

	var bare Renderer
	if got := bare.AssetURL("/images/x.svg"); got != "/images/x.svg" {
		t.Errorf("without a resolver, AssetURL = %q, want the path unchanged", got)
	}
}
