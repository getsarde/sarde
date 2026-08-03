package clientplugins

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/asset"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/plugin"
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
)

func TestMain(m *testing.M) {
	if err := Initialize(); err != nil {
		panic("clientplugins.Initialize: " + err.Error())
	}
	os.Exit(m.Run())
}

func TestManifestParsed(t *testing.T) {
	if len(manifest.Plugins) == 0 {
		t.Fatal("manifest has no plugins")
	}
	if len(manifest.Plugins) != 11 {
		t.Errorf("expected 11 manifest plugins, got %d", len(manifest.Plugins))
	}
}

func TestAllManifestPluginsHaveDefaults(t *testing.T) {
	for slug := range manifest.Plugins {
		defaults := Defaults(slug)
		if defaults == nil {
			t.Errorf("plugin %q has no defaults file", slug)
		}
	}
}

func TestDefaultsExtraction(t *testing.T) {
	d := Defaults("scroll_to_top")
	if d == nil {
		t.Fatal("scroll-to-top defaults not found")
	}
	threshold, ok := d["threshold"]
	if !ok {
		t.Fatal("threshold field not in defaults")
	}
	if v, ok := threshold.(int); !ok || v != 30 {
		t.Errorf("threshold = %v (%T), want 30", threshold, threshold)
	}
}

func TestMergeConfig(t *testing.T) {
	defaults := map[string]any{"a": 1, "b": "hello"}
	user := map[string]any{"b": "world", "c": true}
	merged := mergeConfig(defaults, user)

	if merged["a"] != 1 {
		t.Errorf("a = %v, want 1", merged["a"])
	}
	if merged["b"] != "world" {
		t.Errorf("b = %v, want world", merged["b"])
	}
	if merged["c"] != true {
		t.Errorf("c = %v, want true", merged["c"])
	}
}

func TestShouldInject(t *testing.T) {
	page := &engine.Page{PageIdentity: engine.PageIdentity{Kind: engine.KindPage}}
	rd := &engine.RouteData{Layout: engine.LayoutDocs}

	tests := []struct {
		rule string
		want bool
	}{
		{"always", true},
		{"has_sidebar", true},
		{"has_toc", false}, // no headings
		{"has_code_blocks", false},
		{"has_images", false},
		{"has_prev_next", false},
		{"is_content_page", true},
		{"has_updated", false},
		{"has_headings", false},
		{"unknown_rule", false},
	}

	for _, tt := range tests {
		got := shouldInject(tt.rule, page, rd)
		if got != tt.want {
			t.Errorf("shouldInject(%q) = %v, want %v", tt.rule, got, tt.want)
		}
	}
}

func TestShouldInject_ContentFlags(t *testing.T) {
	rd := &engine.RouteData{Layout: engine.LayoutDocs}

	page := &engine.Page{PageContent: engine.PageContent{HasCodeBlocks: true}}
	if !shouldInject("has_code_blocks", page, rd) {
		t.Error("has_code_blocks should be true")
	}

	page = &engine.Page{PageContent: engine.PageContent{HasImages: true}}
	if !shouldInject("has_images", page, rd) {
		t.Error("has_images should be true")
	}

	page = &engine.Page{
		PageContent: engine.PageContent{Headings: []engine.Heading{{Level: 2, ID: "test", Text: "Test"}}},
	}
	if !shouldInject("has_toc", page, rd) {
		t.Error("has_toc should be true with docs layout and headings")
	}
	if !shouldInject("has_headings", page, rd) {
		t.Error("has_headings should be true")
	}

	page = &engine.Page{PageIdentity: engine.PageIdentity{Updated: time.Now()}}
	if !shouldInject("has_updated", page, rd) {
		t.Error("has_updated should be true with non-zero Updated")
	}

	page = &engine.Page{
		PageRelationships: engine.PageRelationships{PrevPage: &engine.Page{}},
	}
	if !shouldInject("has_prev_next", page, rd) {
		t.Error("has_prev_next should be true with PrevPage set")
	}
}

func TestShouldInject_LayoutGating(t *testing.T) {
	page := &engine.Page{}

	rd := &engine.RouteData{Layout: engine.LayoutDefault}
	if shouldInject("has_sidebar", page, rd) {
		t.Error("has_sidebar should be false for default layout")
	}

	rd = &engine.RouteData{Layout: engine.LayoutDocs}
	if !shouldInject("has_sidebar", page, rd) {
		t.Error("has_sidebar should be true for docs layout")
	}

	rd = &engine.RouteData{Layout: engine.LayoutWide}
	if !shouldInject("has_sidebar", page, rd) {
		t.Error("has_sidebar should be true for wide layout")
	}
}

func TestRegisterAll(t *testing.T) {
	mgr := plugin.NewManager()
	RegisterAll(mgr, []string{"scroll_to_top", "focus_mode"}, nil, "")
}

// testBundle builds a bundle from the embedded assets for the given slugs.
func testBundle(slugs ...string) bundle {
	return buildBundle(assetsFS, "assets/", slugs)
}

func TestBundleURLs(t *testing.T) {
	b := testBundle("scroll_to_top", "focus_mode")

	if b.cssURL == "" {
		t.Error("cssURL is empty")
	}
	if b.jsURL == "" {
		t.Error("jsURL is empty")
	}
	if !strings.HasPrefix(b.cssURL, "/assets/plugins/plugins.") {
		t.Errorf("cssURL = %q, want prefix /assets/plugins/plugins.", b.cssURL)
	}
	if !strings.HasSuffix(b.cssURL, ".css") {
		t.Errorf("cssURL = %q, want suffix .css", b.cssURL)
	}
	if !strings.HasPrefix(b.jsURL, "/assets/plugins/plugins.") {
		t.Errorf("jsURL = %q, want prefix /assets/plugins/plugins.", b.jsURL)
	}
	if !strings.HasSuffix(b.jsURL, ".js") {
		t.Errorf("jsURL = %q, want suffix .js", b.jsURL)
	}
}

func TestBundleData(t *testing.T) {
	b := testBundle("scroll_to_top", "focus_mode")

	if len(b.css) == 0 {
		t.Error("css is empty")
	}
	if len(b.js) == 0 {
		t.Error("js is empty")
	}
}

// TestBundleOnlyEnabled guards the contract that plugins.enabled governs:
// a plugin that is not enabled must contribute nothing to the bundle.
func TestBundleOnlyEnabled(t *testing.T) {
	b := testBundle("scroll_to_top")

	if strings.Contains(string(b.js), "Navigate between pages") {
		t.Error("bundle contains keyboard-nav JS, which was not enabled")
	}
	if strings.Contains(string(b.css), "sarde-kbd-nav-hint") {
		t.Error("bundle contains keyboard-nav CSS, which was not enabled")
	}
	if !strings.Contains(string(b.css), "sarde-scroll-to-top") {
		t.Error("bundle is missing scroll-to-top CSS, which was enabled")
	}
}

// TestBundleEmptyWhenNothingEnabled covers a site that enables no client
// plugins at all: nothing to bundle means nothing to write or reference.
func TestBundleEmptyWhenNothingEnabled(t *testing.T) {
	b := testBundle()

	if len(b.css) != 0 || len(b.js) != 0 {
		t.Error("bundle is non-empty with no plugins enabled")
	}
	if b.cssURL != "" || b.jsURL != "" {
		t.Errorf("bundle URLs are set with no plugins enabled: %q / %q", b.cssURL, b.jsURL)
	}
}

// TestBundleSlugOrderIndependent ensures the fingerprint depends on which
// plugins are bundled, not on the order they were passed in.
func TestBundleSlugOrderIndependent(t *testing.T) {
	a := testBundle("scroll_to_top", "focus_mode")
	b := testBundle("focus_mode", "scroll_to_top")

	if a.cssURL != b.cssURL || a.jsURL != b.jsURL {
		t.Errorf("bundle URLs differ by slug order: %q/%q vs %q/%q", a.cssURL, a.jsURL, b.cssURL, b.jsURL)
	}
}

// TestAssetSourceDir covers the 'sarde dev --theme-dev' path: assets read from
// a directory must match the embedded ones, and a bad dir must fall back to
// embedded rather than producing an empty bundle.
func TestAssetSourceDir(t *testing.T) {
	slugs := []string{"scroll_to_top", "focus_mode"}
	want := testBundle(slugs...)

	fsys, prefix := assetSource("assets")
	if got := buildBundle(fsys, prefix, slugs); got.jsURL != want.jsURL || got.cssURL != want.cssURL {
		t.Errorf("dir-sourced bundle differs from embedded: %q/%q vs %q/%q",
			got.cssURL, got.jsURL, want.cssURL, want.jsURL)
	}

	fsys, prefix = assetSource(filepath.Join(t.TempDir(), "does-not-exist"))
	if got := buildBundle(fsys, prefix, slugs); got.jsURL != want.jsURL {
		t.Errorf("missing dir did not fall back to embedded assets: jsURL = %q", got.jsURL)
	}
}

func TestBundleHashDeterministic(t *testing.T) {
	b := testBundle("scroll_to_top", "focus_mode")

	hash := asset.Fingerprint(b.css)
	expected := "/assets/plugins/plugins." + hash + ".css"
	if b.cssURL != expected {
		t.Errorf("cssURL = %q, want %q", b.cssURL, expected)
	}

	hash = asset.Fingerprint(b.js)
	expected = "/assets/plugins/plugins." + hash + ".js"
	if b.jsURL != expected {
		t.Errorf("jsURL = %q, want %q", b.jsURL, expected)
	}
}

func TestBundleIsMinified(t *testing.T) {
	b := testBundle("scroll_to_top", "focus_mode")

	if strings.Contains(string(b.css), "/* scroll_to_top */") {
		t.Error("bundled CSS still contains source comments, minification not applied")
	}
	if strings.Contains(string(b.js), "/* scroll_to_top */") {
		t.Error("bundled JS still contains source comments, minification not applied")
	}
}

func TestPluginSlugs(t *testing.T) {
	slugs := PluginSlugs()
	if len(slugs) != 11 {
		t.Errorf("PluginSlugs() = %d entries, want 11", len(slugs))
	}

	slugSet := make(map[string]bool)
	for _, s := range slugs {
		slugSet[s] = true
	}
	for _, expected := range []string{"scroll_to_top", "focus_mode", "reading_progress"} {
		if !slugSet[expected] {
			t.Errorf("missing slug %q", expected)
		}
	}
}

func TestAppendUnique(t *testing.T) {
	list := []string{"a", "b"}
	list = cfgutil.AppendUnique(list, "c")
	if len(list) != 3 {
		t.Errorf("expected 3, got %d", len(list))
	}
	list = cfgutil.AppendUnique(list, "b")
	if len(list) != 3 {
		t.Errorf("expected 3 (no dup), got %d", len(list))
	}
}
