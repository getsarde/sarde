package telescope

import (
	"reflect"
	"testing"
)

func TestCfgTelescopeConfig_Defaults(t *testing.T) {
	ensureAssets()
	tc := cfgTelescopeConfig(nil)

	if tc.shortcutKey != "/" {
		t.Errorf("shortcutKey = %q, want /", tc.shortcutKey)
	}
	if tc.maxResults != 20 {
		t.Errorf("maxResults = %d, want 20", tc.maxResults)
	}
	if tc.maxRecent != 10 {
		t.Errorf("maxRecent = %d, want 10", tc.maxRecent)
	}
	if tc.maxPinned != 50 {
		t.Errorf("maxPinned = %d, want 50", tc.maxPinned)
	}
	if tc.debounceMs != 120 {
		t.Errorf("debounceMs = %d, want 120", tc.debounceMs)
	}
	if tc.defaultTab != "search" {
		t.Errorf("defaultTab = %q, want search", tc.defaultTab)
	}
	if len(tc.exclude) != 0 {
		t.Errorf("exclude = %v, want empty", tc.exclude)
	}
	if tc.placeholder != "" {
		t.Errorf("placeholder = %q, want empty", tc.placeholder)
	}
}

func TestCfgTelescopeConfig_Overrides(t *testing.T) {
	ensureAssets()
	tc := cfgTelescopeConfig(map[string]any{
		"shortcut_key": "k",
		"max_results":  33,
		"max_recent":   5,
		"max_pinned":   10,
		"debounce_ms":  0,
		"default_tab":  "recent",
		"exclude":      []any{"/drafts/*", "/internal/*"},
		"placeholder":  "Jump to...",
	})

	if tc.shortcutKey != "k" {
		t.Errorf("shortcutKey = %q, want k", tc.shortcutKey)
	}
	if tc.maxResults != 33 {
		t.Errorf("maxResults = %d, want 33", tc.maxResults)
	}
	if tc.maxRecent != 5 {
		t.Errorf("maxRecent = %d, want 5", tc.maxRecent)
	}
	if tc.maxPinned != 10 {
		t.Errorf("maxPinned = %d, want 10", tc.maxPinned)
	}
	if tc.debounceMs != 0 {
		t.Errorf("debounceMs = %d, want 0", tc.debounceMs)
	}
	if tc.defaultTab != "recent" {
		t.Errorf("defaultTab = %q, want recent", tc.defaultTab)
	}
	if want := []string{"/drafts/*", "/internal/*"}; !reflect.DeepEqual(tc.exclude, want) {
		t.Errorf("exclude = %v, want %v", tc.exclude, want)
	}
	if tc.placeholder != "Jump to..." {
		t.Errorf("placeholder = %q, want Jump to...", tc.placeholder)
	}
}

func TestCfgTelescopeConfig_Clamping(t *testing.T) {
	ensureAssets()
	tc := cfgTelescopeConfig(map[string]any{
		"max_results": 5000,
		"max_recent":  0,
		"max_pinned":  -3,
		"debounce_ms": 99999,
	})

	if tc.maxResults != 100 {
		t.Errorf("maxResults = %d, want clamped 100", tc.maxResults)
	}
	if tc.maxRecent != 1 {
		t.Errorf("maxRecent = %d, want clamped 1", tc.maxRecent)
	}
	if tc.maxPinned != 1 {
		t.Errorf("maxPinned = %d, want clamped 1", tc.maxPinned)
	}
	if tc.debounceMs != 1000 {
		t.Errorf("debounceMs = %d, want clamped 1000", tc.debounceMs)
	}
}

func TestCfgTelescopeConfig_InvalidValues(t *testing.T) {
	ensureAssets()
	tc := cfgTelescopeConfig(map[string]any{
		"shortcut_key": "",
		"default_tab":  "bogus",
	})
	if tc.shortcutKey != "/" {
		t.Errorf("empty shortcutKey should fall back to /, got %q", tc.shortcutKey)
	}
	if tc.defaultTab != "search" {
		t.Errorf("invalid defaultTab should fall back to search, got %q", tc.defaultTab)
	}
}

func TestCfgTelescopeConfig_CamelAlias(t *testing.T) {
	ensureAssets()
	tc := cfgTelescopeConfig(map[string]any{
		"maxResults": 42,
	})
	if tc.maxResults != 42 {
		t.Errorf("deprecated camelCase maxResults should resolve, got %d", tc.maxResults)
	}
}

func TestClientConfig_OmitsExclude(t *testing.T) {
	ensureAssets()
	tc := cfgTelescopeConfig(map[string]any{"exclude": []any{"/secret/*"}})
	cc := tc.clientConfig(map[string]string{"pin": "Pin page"})
	if _, ok := cc["exclude"]; ok {
		t.Error("clientConfig must not expose exclude patterns to the browser")
	}
	strs, ok := cc["strings"].(map[string]string)
	if !ok || strs["pin"] != "Pin page" {
		t.Errorf("clientConfig strings = %v, want pin label present", cc["strings"])
	}
}

func TestFieldNames(t *testing.T) {
	names := FieldNames()
	want := []string{"debounce_ms", "default_tab", "exclude", "max_pinned", "max_recent", "max_results", "placeholder", "shortcut_key"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("FieldNames() = %v, want %v", names, want)
	}
}
