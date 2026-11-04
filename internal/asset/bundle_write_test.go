package asset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

// bundleResolver resolves URLs for an en/fr site with a versioned docs
// collection (latest v2 at the root, v1 under /docs/v1/).
func bundleResolver() *engine.URLResolver {
	return &engine.URLResolver{
		BasePath:         "/site/",
		I18nEnabled:      true,
		DefaultLang:      "en",
		Languages:        map[string]bool{"en": true, "fr": true},
		VersionIDs:       map[string]bool{"v1": true, "v2": true},
		CollectionMounts: []string{"/docs", "/blog"},
	}
}

func versionedDocs() *engine.Collection {
	return &engine.Collection{
		Name: "docs",
		Config: &engine.CollectionConfig{
			Versioning: &engine.VersionConfig{Enabled: true, LastVersion: "v2"},
		},
	}
}

// A resource's Permalink carries the base path and the owning page's
// language and version; RelPermalink stays prefix-free.
func TestEnhancePageResources_PermalinkCarriesLanguageAndVersion(t *testing.T) {
	r := bundleResolver()
	docs := versionedDocs()
	tests := []struct {
		name string
		page *engine.Page
		want string
	}{
		{"default language", &engine.Page{
			PageIdentity: engine.PageIdentity{FilePath: "/p/content/blog/trip/index.md", RelPermalink: "/blog/trip/"},
			PageI18n:     engine.PageI18n{Lang: "en"},
		}, "/site/blog/trip/cover.jpg"},
		{"translated", &engine.Page{
			PageIdentity: engine.PageIdentity{FilePath: "/p/content/fr/blog/trip/index.md", RelPermalink: "/blog/trip/"},
			PageI18n:     engine.PageI18n{Lang: "fr"},
		}, "/site/fr/blog/trip/cover.jpg"},
		{"older version", &engine.Page{
			PageIdentity:      engine.PageIdentity{FilePath: "/p/content/docs/v1/guide/index.md", RelPermalink: "/docs/guide/"},
			PageI18n:          engine.PageI18n{Lang: "en"},
			PageVersioning:    engine.PageVersioning{Version: "v1"},
			PageRelationships: engine.PageRelationships{Collection: docs},
		}, "/site/docs/v1/guide/cover.jpg"},
		{"latest version", &engine.Page{
			PageIdentity:      engine.PageIdentity{FilePath: "/p/content/docs/guide/index.md", RelPermalink: "/docs/guide/"},
			PageI18n:          engine.PageI18n{Lang: "en"},
			PageVersioning:    engine.PageVersioning{Version: "v2"},
			PageRelationships: engine.PageRelationships{Collection: docs},
		}, "/site/docs/guide/cover.jpg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.page.Resources = []engine.Resource{{Name: "cover.jpg"}}
			e := &ResourceEnhancer{DevMode: true, URLResolver: r}
			if err := e.EnhancePageResources(tt.page); err != nil {
				t.Fatal(err)
			}
			res := tt.page.Resources[0]
			if res.Permalink != tt.want {
				t.Errorf("Permalink = %q, want %q", res.Permalink, tt.want)
			}
			if res.RelPermalink != tt.page.RelPermalink+"cover.jpg" {
				t.Errorf("RelPermalink = %q, want the prefix-free %q", res.RelPermalink, tt.page.RelPermalink+"cover.jpg")
			}
		})
	}
}

// Without a resolver the Permalink stays empty and ResourceURL falls back to
// the RelPermalink.
func TestEnhancePageResources_NoResolverLeavesPermalinkEmpty(t *testing.T) {
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{FilePath: "/p/content/blog/trip/index.md", RelPermalink: "/blog/trip/"},
		Resources:    []engine.Resource{{Name: "doc.pdf"}},
	}
	if err := (&ResourceEnhancer{DevMode: true}).EnhancePageResources(page); err != nil {
		t.Fatal(err)
	}
	if got := ResourceURL(&page.Resources[0]); got != "/blog/trip/doc.pdf" {
		t.Errorf("ResourceURL = %q, want /blog/trip/doc.pdf", got)
	}
}

// Each page's bundle files are written beside the page: the English and
// French copies of a translated bundle, and an older version's copy, never
// share a path. Two pages naming one path write it once, the first winning.
func TestWriteBundleAssets_PerLanguageAndVersion(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, body string) string {
		t.Helper()
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	enIndex := write("content/blog/trip/index.md", "")
	write("content/blog/trip/cover.jpg", "english")
	frIndex := write("content/fr/blog/trip/index.md", "")
	write("content/fr/blog/trip/cover.jpg", "french")
	v1Index := write("content/docs/v1/guide/index.md", "")
	write("content/docs/v1/guide/cover.jpg", "v1")

	docs := versionedDocs()
	pages := []*engine.Page{
		{PageIdentity: engine.PageIdentity{FilePath: enIndex, RelPermalink: "/blog/trip/"}, PageI18n: engine.PageI18n{Lang: "en"}},
		{PageIdentity: engine.PageIdentity{FilePath: frIndex, RelPermalink: "/blog/trip/"}, PageI18n: engine.PageI18n{Lang: "fr"}},
		{PageIdentity: engine.PageIdentity{FilePath: v1Index, RelPermalink: "/docs/guide/"}, PageI18n: engine.PageI18n{Lang: "en"},
			PageVersioning: engine.PageVersioning{Version: "v1"}, PageRelationships: engine.PageRelationships{Collection: docs}},
		// A second English page naming the same output path: the first wins.
		{PageIdentity: engine.PageIdentity{FilePath: frIndex, RelPermalink: "/blog/trip/"}, PageI18n: engine.PageI18n{Lang: "en"}},
	}
	for _, p := range pages {
		p.Resources = []engine.Resource{{Name: "cover.jpg", RelPermalink: p.RelPermalink + "cover.jpg"}}
	}

	out := filepath.Join(dir, "dist")
	p := NewPipeline(PipelineOptions{ProjectDir: dir, OutputDir: out, Config: config.Defaults(), URLResolver: bundleResolver()})
	var written []string
	if err := p.WriteBundleAssetsWithOptions(pages, out, func(path string) { written = append(written, path) }, WriteOptions{Parallel: true, WorkerCount: 4}); err != nil {
		t.Fatal(err)
	}
	for rel, want := range map[string]string{
		"blog/trip/cover.jpg":     "english",
		"fr/blog/trip/cover.jpg":  "french",
		"docs/v1/guide/cover.jpg": "v1",
	} {
		data, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			continue
		}
		if string(data) != want {
			t.Errorf("%s = %q, want %q", rel, data, want)
		}
	}
	if len(written) != 3 {
		t.Errorf("wrote %d files, want 3 (the duplicate path once)", len(written))
	}

	// Without a resolver every copy goes to the prefix-free path, as before.
	legacy := filepath.Join(dir, "legacy")
	p = NewPipeline(PipelineOptions{ProjectDir: dir, OutputDir: legacy, Config: config.Defaults()})
	if err := p.WriteBundleAssetsWithOptions(pages[:1], legacy, nil, WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(legacy, "blog", "trip", "cover.jpg")); err != nil {
		t.Errorf("nil resolver: %v", err)
	}
}
