package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// buildSearchSite builds the rich fixture with the given config and returns
// the dist directory, the home page HTML, and the build warnings.
func buildSearchSite(t *testing.T, cfg *config.SiteConfig) (string, string, []string) {
	t.Helper()
	projDir := createRichFixtureSite(t)
	builder := NewSiteBuilder(BuildOptions{
		ProjectDir:  projDir,
		Config:      cfg,
		ThemeConfig: buildThemeConfig(),
		EmbeddedFS:  embedded.ThemeFS(),
	})
	result, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	msgs := make([]string, 0, len(result.Warnings))
	for _, w := range result.Warnings {
		msgs = append(msgs, w.Message)
	}
	distDir := filepath.Join(projDir, "dist")
	return distDir, readFixture(t, distDir, "index.html"), msgs
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// TestBuild_SearchEnabled_DefaultShipsEverything pins the baseline the two
// toggle tests below subtract from: button, modal, runtime script, strings
// script and index are all present by default.
func TestBuild_SearchEnabled_DefaultShipsEverything(t *testing.T) {
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"search"}

	distDir, homeHTML, _ := buildSearchSite(t, cfg)

	for _, want := range []string{
		`class="sarde-search-trigger"`,
		`id="sarde-search-modal"`,
		`/assets/js/static-search.js`,
		`window.__SARDE__.pluginConfig.search=`,
		`"recent":"Recent"`,
		`"results_count_other":"{count} results for '{term}'"`,
	} {
		if !strings.Contains(homeHTML, want) {
			t.Errorf("expected %q in home HTML", want)
		}
	}
	if !fileExists(filepath.Join(distDir, "search-index.en.json")) {
		t.Error("expected search-index.en.json")
	}
	if !fileExists(filepath.Join(distDir, "assets", "js", "static-search.js")) {
		t.Error("expected assets/js/static-search.js")
	}
}

// TestBuild_SearchDisabled_RemovesEverything covers `search.enabled: false`,
// the documented switch that used to be dead config: it must remove the
// header button, the modal markup, the runtime script and the index, and
// must not warn that plugins.config.search is unused.
func TestBuild_SearchDisabled_RemovesEverything(t *testing.T) {
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"search"}
	cfg.Search.Enabled = config.BoolPtr(false)
	cfg.Plugins.Config = map[string]map[string]any{
		"search": {"max_content_length": 1000},
	}

	distDir, homeHTML, warnings := buildSearchSite(t, cfg)

	for _, unwanted := range []string{
		`sarde-search-trigger`,
		`sarde-search-modal`,
		`static-search.js`,
		`pluginConfig.search=`,
	} {
		if strings.Contains(homeHTML, unwanted) {
			t.Errorf("did not expect %q in home HTML when search is disabled", unwanted)
		}
	}
	if fileExists(filepath.Join(distDir, "search-index.en.json")) {
		t.Error("search-index.en.json should not be written when search is disabled")
	}
	if fileExists(filepath.Join(distDir, "assets", "js", "static-search.js")) {
		t.Error("static-search.js should not be written when search is disabled")
	}
	if containsSubstring(warnings, `plugin "search"`) {
		t.Errorf("search.enabled: false should not warn about plugins.config.search, warnings: %v", warnings)
	}
}

// TestBuild_HeaderSearchHidden_KeepsIndex covers `header.search: false`: only
// the header button goes away; the index, the runtime and the modal stay so a
// custom [data-search-trigger] elsewhere can still open search.
func TestBuild_HeaderSearchHidden_KeepsIndex(t *testing.T) {
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"search"}
	cfg.Header.Search = config.BoolPtr(false)

	distDir, homeHTML, _ := buildSearchSite(t, cfg)

	if strings.Contains(homeHTML, `sarde-search-trigger`) {
		t.Error("header.search: false should hide the header search button")
	}
	if !strings.Contains(homeHTML, `/assets/js/static-search.js`) {
		t.Error("header.search: false should keep the runtime script")
	}
	if !fileExists(filepath.Join(distDir, "search-index.en.json")) {
		t.Error("header.search: false should keep the search index")
	}
}

// TestBuild_SearchHighlighter_PluginConfigSignal covers the contract the
// runtime relies on to decide whether result links carry ?q=: the
// search_highlighter slug is present in pluginConfig exactly when the plugin
// is enabled, and the client-plugin config script merges rather than
// replaces pluginConfig so the search strings survive.
func TestBuild_SearchHighlighter_PluginConfigSignal(t *testing.T) {
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"search", "search_highlighter"}

	distDir, homeHTML, _ := buildSearchSite(t, cfg)

	if !strings.Contains(homeHTML, `"search_highlighter"`) {
		t.Error("expected search_highlighter in pluginConfig when enabled")
	}
	if !strings.Contains(homeHTML, `window.__SARDE__.pluginConfig=Object.assign(window.__SARDE__.pluginConfig||{},`) {
		t.Error("client plugin config script should merge into pluginConfig")
	}
	if strings.Contains(homeHTML, `window.__SARDE__.pluginConfig={`) {
		t.Error("client plugin config script must not replace pluginConfig wholesale")
	}
	if !strings.Contains(homeHTML, `pluginConfig.search=`) {
		t.Error("search strings script should still be present alongside client plugin config")
	}

	runtime := readFixture(t, distDir, "assets/js/static-search.js")
	for _, want := range []string{"function withQuery(", "function highlighterOn(", `pluginConfig.search_highlighter`} {
		if !strings.Contains(runtime, want) {
			t.Errorf("static-search.js should contain %q", want)
		}
	}

	// Highlighter off: the slug must not be advertised.
	cfg = config.Defaults()
	cfg.Plugins.Enabled = []string{"search"}
	_, homeHTML, _ = buildSearchSite(t, cfg)
	if strings.Contains(homeHTML, `search_highlighter`) {
		t.Error("search_highlighter should be absent from pluginConfig when not enabled")
	}
}
