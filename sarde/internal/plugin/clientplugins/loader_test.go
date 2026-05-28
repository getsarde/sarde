package clientplugins

import (
	"strings"
	"testing"
	"time"

	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/plugin"
)

func TestManifestParsed(t *testing.T) {
	if len(manifest.Plugins) == 0 {
		t.Fatal("manifest has no plugins")
	}
	if len(manifest.Plugins) != 15 {
		t.Errorf("expected 15 manifest plugins, got %d", len(manifest.Plugins))
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
	d := Defaults("scroll-to-top")
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
	RegisterAll(mgr, []string{"scroll-to-top", "focus-mode"}, nil)
}

func TestBundleURLs(t *testing.T) {
	if bundleCSSURL == "" {
		t.Error("bundleCSSURL is empty")
	}
	if bundleJSURL == "" {
		t.Error("bundleJSURL is empty")
	}
	if !strings.HasPrefix(bundleCSSURL, "/assets/plugins/plugins.") {
		t.Errorf("bundleCSSURL = %q, want prefix /assets/plugins/plugins.", bundleCSSURL)
	}
	if !strings.HasSuffix(bundleCSSURL, ".css") {
		t.Errorf("bundleCSSURL = %q, want suffix .css", bundleCSSURL)
	}
	if !strings.HasPrefix(bundleJSURL, "/assets/plugins/plugins.") {
		t.Errorf("bundleJSURL = %q, want prefix /assets/plugins/plugins.", bundleJSURL)
	}
	if !strings.HasSuffix(bundleJSURL, ".js") {
		t.Errorf("bundleJSURL = %q, want suffix .js", bundleJSURL)
	}
}

func TestBundleData(t *testing.T) {
	if len(bundleCSS) == 0 {
		t.Error("bundleCSS is empty")
	}
	if len(bundleJS) == 0 {
		t.Error("bundleJS is empty")
	}
}

func TestBundleHashDeterministic(t *testing.T) {
	hash := contentHash(bundleCSS)
	expected := "/assets/plugins/plugins." + hash + ".css"
	if bundleCSSURL != expected {
		t.Errorf("bundleCSSURL = %q, want %q", bundleCSSURL, expected)
	}

	hash = contentHash(bundleJS)
	expected = "/assets/plugins/plugins." + hash + ".js"
	if bundleJSURL != expected {
		t.Errorf("bundleJSURL = %q, want %q", bundleJSURL, expected)
	}
}

func TestBundleIsMinified(t *testing.T) {
	css := string(bundleCSS)
	if strings.Contains(css, "/* scroll-to-top */") {
		t.Error("bundled CSS still contains source comments — minification not applied")
	}

	js := string(bundleJS)
	if strings.Contains(js, "/* scroll-to-top */") {
		t.Error("bundled JS still contains source comments — minification not applied")
	}
}

func TestPluginSlugs(t *testing.T) {
	slugs := PluginSlugs()
	if len(slugs) != 15 {
		t.Errorf("PluginSlugs() = %d entries, want 15", len(slugs))
	}

	slugSet := make(map[string]bool)
	for _, s := range slugs {
		slugSet[s] = true
	}
	for _, expected := range []string{"scroll-to-top", "focus-mode", "code-collapsible", "toc-progress"} {
		if !slugSet[expected] {
			t.Errorf("missing slug %q", expected)
		}
	}
}

func TestAppendUnique(t *testing.T) {
	list := []string{"a", "b"}
	list = appendUnique(list, "c")
	if len(list) != 3 {
		t.Errorf("expected 3, got %d", len(list))
	}
	list = appendUnique(list, "b")
	if len(list) != 3 {
		t.Errorf("expected 3 (no dup), got %d", len(list))
	}
}
