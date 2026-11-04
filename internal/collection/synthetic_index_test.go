package collection

import (
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
)

// buildBilingual discovers contentDir as an en/fr site and builds its
// collections with the given fallback policy.
func buildBilingual(t *testing.T, contentDir, fallback string, versioned bool) map[string]*engine.Collection {
	t.Helper()
	scanner := &content.Scanner{Languages: map[string]bool{"en": true, "fr": true}, DefaultLang: "en"}
	siteCfg := config.Defaults()
	siteCfg.I18n.DefaultLanguage = "en"
	siteCfg.I18n.Fallback = fallback
	siteCfg.I18n.Languages = map[string]config.LanguageConfig{"en": {Weight: 1}, "fr": {Weight: 2}}
	if versioned {
		on := true
		siteCfg.Collections = map[string]*config.CollectionSiteConfig{
			"docs": {Versioning: &config.VersioningConfig{Enabled: &on, LastVersion: "v2",
				Versions: []config.VersionEntry{{ID: "v2"}, {ID: "v1"}}}},
		}
		scanner.VersionIDs = map[string]map[string]bool{"docs": {"v1": true, "v2": true}}
	}
	files, err := scanner.DiscoverFiles(contentDir)
	if err != nil {
		t.Fatal(err)
	}
	collections, _, err := BuildCollections(files, siteCfg, contentDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	return collections
}

// rootIndexes returns the collection's root section pages keyed by language.
func rootIndexes(col *engine.Collection) map[string][]*engine.Page {
	out := map[string][]*engine.Page{}
	for _, p := range col.Pages {
		if p.Kind == engine.KindSection && sectionDir(p.RelPermalink, col.Name) == "" {
			out[p.Lang] = append(out[p.Lang], p)
		}
	}
	return out
}

// A blog with no _index.md whose newest post is French: each language gets
// its own synthetic index, and the collection's index page is the English one.
func TestBuildCollections_SyntheticIndexPerLanguage(t *testing.T) {
	contentDir := filepath.Join(t.TempDir(), "content")
	writeTestFile(t, contentDir, filepath.Join("blog", "old.md"), "---\ntitle: Old\ndate: 2026-01-01T00:00:00Z\n---\n")
	writeTestFile(t, contentDir, filepath.Join("fr", "blog", "new.md"), "---\ntitle: Neuf\ndate: 2026-03-01T00:00:00Z\n---\n")

	blog := buildBilingual(t, contentDir, "default", false)["blog"]
	idx := rootIndexes(blog)
	for _, lang := range []string{"en", "fr"} {
		if len(idx[lang]) != 1 {
			t.Fatalf("%s: %d root indexes, want 1", lang, len(idx[lang]))
		}
		if got := idx[lang][0].LangRelPath; got != "blog/_index.md" {
			t.Errorf("%s synthetic index LangRelPath = %q, want blog/_index.md", lang, got)
		}
	}
	if blog.IndexPage == nil || blog.IndexPage.Lang != "en" {
		t.Errorf("IndexPage = %+v, want the English synthetic index", blog.IndexPage)
	}
}

// Only a French _index.md exists: English still gets a synthetic index, and
// it becomes the collection's index page.
func TestBuildCollections_SyntheticIndexDefaultBesideTranslatedIndex(t *testing.T) {
	contentDir := filepath.Join(t.TempDir(), "content")
	writeTestFile(t, contentDir, filepath.Join("blog", "a.md"), "---\ntitle: A\ndate: 2026-01-01T00:00:00Z\n---\n")
	writeTestFile(t, contentDir, filepath.Join("fr", "blog", "_index.md"), "---\ntitle: Carnet\n---\n")
	writeTestFile(t, contentDir, filepath.Join("fr", "blog", "b.md"), "---\ntitle: B\ndate: 2026-02-01T00:00:00Z\n---\n")

	blog := buildBilingual(t, contentDir, "default", false)["blog"]
	idx := rootIndexes(blog)
	if len(idx["en"]) != 1 || idx["en"][0].FilePath != "" {
		t.Errorf("en root indexes = %v, want one synthetic", idx["en"])
	}
	if len(idx["fr"]) != 1 || idx["fr"][0].Title != "Carnet" {
		t.Errorf("fr root indexes = %v, want the real Carnet only", idx["fr"])
	}
	if blog.IndexPage != idx["en"][0] {
		t.Errorf("IndexPage = %q (%s), want the English synthetic index", blog.IndexPage.Title, blog.IndexPage.Lang)
	}
}

// The default language has a real _index.md: under fallback default the
// i18n fallback copies it, so French gets no synthetic index.
func TestBuildCollections_SyntheticIndexNoTranslatedWhenDefaultHasIndex(t *testing.T) {
	contentDir := filepath.Join(t.TempDir(), "content")
	writeTestFile(t, contentDir, filepath.Join("blog", "_index.md"), "---\ntitle: Blog\n---\n")
	writeTestFile(t, contentDir, filepath.Join("blog", "a.md"), "---\ntitle: A\ndate: 2026-01-01T00:00:00Z\n---\n")
	writeTestFile(t, contentDir, filepath.Join("fr", "blog", "b.md"), "---\ntitle: B\ndate: 2026-02-01T00:00:00Z\n---\n")

	blog := buildBilingual(t, contentDir, "default", false)["blog"]
	idx := rootIndexes(blog)
	if len(idx["fr"]) != 0 {
		t.Errorf("fr root indexes = %d, want 0 (the fallback copy comes later)", len(idx["fr"]))
	}
	if blog.IndexPage == nil || blog.IndexPage.Title != "Blog" || blog.IndexPage.FilePath == "" {
		t.Errorf("IndexPage = %+v, want the real English index", blog.IndexPage)
	}
}

// Under fallback omit no copy of the English index is made, so French
// pages get a synthetic index of their own.
func TestBuildCollections_SyntheticIndexUnderOmit(t *testing.T) {
	contentDir := filepath.Join(t.TempDir(), "content")
	writeTestFile(t, contentDir, filepath.Join("blog", "_index.md"), "---\ntitle: Blog\n---\n")
	writeTestFile(t, contentDir, filepath.Join("blog", "a.md"), "---\ntitle: A\ndate: 2026-01-01T00:00:00Z\n---\n")
	writeTestFile(t, contentDir, filepath.Join("fr", "blog", "b.md"), "---\ntitle: B\ndate: 2026-02-01T00:00:00Z\n---\n")

	blog := buildBilingual(t, contentDir, "omit", false)["blog"]
	idx := rootIndexes(blog)
	if len(idx["fr"]) != 1 || idx["fr"][0].FilePath != "" {
		t.Errorf("fr root indexes = %v, want one synthetic", idx["fr"])
	}
}

// A versioned collection's synthetic index belongs to the latest version.
func TestBuildCollections_SyntheticIndexVersioned(t *testing.T) {
	contentDir := filepath.Join(t.TempDir(), "content")
	writeTestFile(t, contentDir, filepath.Join("docs", "intro.md"), "---\ntitle: Intro\n---\n")
	writeTestFile(t, contentDir, filepath.Join("docs", "v1", "intro.md"), "---\ntitle: Intro v1\n---\n")

	docs := buildBilingual(t, contentDir, "default", true)["docs"]
	idx := rootIndexes(docs)
	if len(idx["en"]) != 1 {
		t.Fatalf("en root indexes = %d, want 1", len(idx["en"]))
	}
	if got := idx["en"][0]; got.Version != "v2" || got.VersionRelPath != "_index.md" {
		t.Errorf("synthetic index Version = %q, VersionRelPath = %q, want v2 and _index.md", got.Version, got.VersionRelPath)
	}
}

// A single-language collection with no _index.md keeps its one synthetic
// index, now with a LangRelPath.
func TestBuildCollections_SyntheticIndexSingleLanguage(t *testing.T) {
	contentDir := filepath.Join(t.TempDir(), "content")
	writeTestFile(t, contentDir, filepath.Join("blog", "a.md"), "---\ntitle: A\ndate: 2026-01-01T00:00:00Z\n---\n")
	files, err := (&content.Scanner{}).DiscoverFiles(contentDir)
	if err != nil {
		t.Fatal(err)
	}
	collections, _, err := BuildCollections(files, config.Defaults(), contentDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	blog := collections["blog"]
	idx := rootIndexes(blog)
	if len(idx[""]) != 1 || blog.IndexPage != idx[""][0] {
		t.Fatalf("root indexes = %v, IndexPage = %v; want one synthetic index as the IndexPage", idx, blog.IndexPage)
	}
	if blog.IndexPage.Section == nil || blog.IndexPage.Section.IndexPage != blog.IndexPage {
		t.Error("the synthetic index is not attached to the root section")
	}
}
