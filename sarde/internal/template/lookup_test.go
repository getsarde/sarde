package template

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/frostybee/sarde/internal/engine"
)

func tempResolver(t *testing.T) (*engine.ThemeResolver, string) {
	t.Helper()
	dir := t.TempDir()
	efs := fstest.MapFS{
		"_default/baseof.html": {Data: []byte("embedded-default-baseof")},
		"_default/single.html": {Data: []byte("embedded-default-single")},
		"_docs/baseof.html":    {Data: []byte("embedded-docs-baseof")},
		"_docs/single.html":    {Data: []byte("embedded-docs-single")},
		"partials/head.html":   {Data: []byte("embedded-partial-head")},
	}
	return &engine.ThemeResolver{
		ProjectDir: dir,
		EmbeddedFS: efs,
	}, dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveTemplate_EmbeddedFallback(t *testing.T) {
	resolver, _ := tempResolver(t)

	data, _, err := resolveTemplate(resolver, "", engine.LayoutDefault, "single.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "embedded-default-single" {
		t.Errorf("got %q, want embedded-default-single", data)
	}
}

func TestResolveTemplate_UserOverrideWins(t *testing.T) {
	resolver, dir := tempResolver(t)

	writeFile(t, filepath.Join(dir, "layouts", "_default", "single.html"), "user-default-single")

	data, _, err := resolveTemplate(resolver, "", engine.LayoutDefault, "single.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "user-default-single" {
		t.Errorf("got %q, want user-default-single", data)
	}
}

func TestResolveTemplate_CollectionOverrideWins(t *testing.T) {
	resolver, dir := tempResolver(t)

	writeFile(t, filepath.Join(dir, "layouts", "_default", "single.html"), "user-default-single")
	writeFile(t, filepath.Join(dir, "layouts", "blog", "single.html"), "user-blog-single")

	data, _, err := resolveTemplate(resolver, "blog", engine.LayoutDefault, "single.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "user-blog-single" {
		t.Errorf("got %q, want user-blog-single", data)
	}
}

func TestResolveTemplate_ThemeFallback(t *testing.T) {
	resolver, dir := tempResolver(t)
	resolver.ThemeName = "aurora"

	writeFile(t, filepath.Join(dir, "themes", "aurora", "layouts", "_default", "single.html"), "theme-default-single")

	data, _, err := resolveTemplate(resolver, "", engine.LayoutDefault, "single.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "theme-default-single" {
		t.Errorf("got %q, want theme-default-single", data)
	}
}

func TestResolveTemplate_ThemeCollectionOverride(t *testing.T) {
	resolver, dir := tempResolver(t)
	resolver.ThemeName = "aurora"

	writeFile(t, filepath.Join(dir, "themes", "aurora", "layouts", "blog", "single.html"), "theme-blog-single")
	writeFile(t, filepath.Join(dir, "themes", "aurora", "layouts", "_default", "single.html"), "theme-default-single")

	data, _, err := resolveTemplate(resolver, "blog", engine.LayoutDefault, "single.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "theme-blog-single" {
		t.Errorf("got %q, want theme-blog-single", data)
	}
}

func TestResolveTemplate_DocsLayout_DocsLayerChecked(t *testing.T) {
	resolver, _ := tempResolver(t)

	// Docs layout should find _docs/single.html from embedded FS before _default
	data, _, err := resolveTemplate(resolver, "docs", engine.LayoutDocs, "single.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "embedded-docs-single" {
		t.Errorf("got %q, want embedded-docs-single", data)
	}
}

func TestResolveTemplate_DocsLayout_UserDocsOverride(t *testing.T) {
	resolver, dir := tempResolver(t)

	writeFile(t, filepath.Join(dir, "layouts", "_docs", "single.html"), "user-docs-single")

	data, _, err := resolveTemplate(resolver, "docs", engine.LayoutDocs, "single.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "user-docs-single" {
		t.Errorf("got %q, want user-docs-single", data)
	}
}

func TestResolveTemplate_NotFound(t *testing.T) {
	resolver, _ := tempResolver(t)

	_, _, err := resolveTemplate(resolver, "", engine.LayoutDefault, "nonexistent.html")
	if err == nil {
		t.Fatal("expected error for missing template")
	}
}

func TestResolvePartial_EmbeddedFallback(t *testing.T) {
	resolver, _ := tempResolver(t)

	data, _, err := resolvePartial(resolver, "head.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "embedded-partial-head" {
		t.Errorf("got %q, want embedded-partial-head", data)
	}
}

func TestResolvePartial_UserOverrideWins(t *testing.T) {
	resolver, dir := tempResolver(t)

	writeFile(t, filepath.Join(dir, "layouts", "partials", "head.html"), "user-partial-head")

	data, _, err := resolvePartial(resolver, "head.html")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "user-partial-head" {
		t.Errorf("got %q, want user-partial-head", data)
	}
}

func TestResolveAllPartials(t *testing.T) {
	resolver, dir := tempResolver(t)

	// Add a user partial that overrides embedded
	writeFile(t, filepath.Join(dir, "layouts", "partials", "head.html"), "user-partial-head")
	// Add a new user partial
	writeFile(t, filepath.Join(dir, "layouts", "partials", "custom.html"), "user-partial-custom")

	partials := resolveAllPartials(resolver)

	if string(partials["head.html"]) != "user-partial-head" {
		t.Errorf("head.html: got %q, want user-partial-head", partials["head.html"])
	}
	if string(partials["custom.html"]) != "user-partial-custom" {
		t.Errorf("custom.html: got %q, want user-partial-custom", partials["custom.html"])
	}
}
