package telescope

import (
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
)

// telescopeConfig holds the resolved plugin configuration.
type telescopeConfig struct {
	shortcutKey string
	maxResults  int
	maxRecent   int
	maxPinned   int
	debounceMs  int
	defaultTab  string
	exclude     []string
	placeholder string
}

// clamp bounds n to [lo, hi].
func clamp(n, lo, hi int) int {
	if n < lo {
		return lo
	}
	if n > hi {
		return hi
	}
	return n
}

// cfgTelescopeConfig resolves the plugin config from defaults.yaml merged with
// the user's plugins.config.telescope block. Deprecated camelCase keys are
// rewritten to snake_case first.
func cfgTelescopeConfig(cfg map[string]any) telescopeConfig {
	merged := mergeWithDefaults(cfg)

	tc := telescopeConfig{
		shortcutKey: cfgutil.String(merged, "shortcut_key", "/"),
		maxResults:  clamp(cfgutil.Int(merged, "max_results", 20), 1, 100),
		maxRecent:   clamp(cfgutil.Int(merged, "max_recent", 10), 1, 50),
		maxPinned:   clamp(cfgutil.Int(merged, "max_pinned", 50), 1, 200),
		debounceMs:  clamp(cfgutil.Int(merged, "debounce_ms", 120), 0, 1000),
		defaultTab:  cfgutil.String(merged, "default_tab", "search"),
		exclude:     cfgutil.StringSlice(merged, "exclude"),
		placeholder: cfgutil.String(merged, "placeholder", ""),
	}

	if tc.shortcutKey == "" {
		tc.shortcutKey = "/"
	}
	if tc.defaultTab != "search" && tc.defaultTab != "recent" {
		tc.defaultTab = "search"
	}
	return tc
}

// mergeWithDefaults overlays the user config onto the blueprint defaults,
// resolving deprecated camelCase aliases to their snake_case form.
func mergeWithDefaults(cfg map[string]any) map[string]any {
	resolved, _ := cfgutil.ResolveAliases(defaultCfg, cfg)
	merged := make(map[string]any, len(defaultCfg)+len(resolved))
	for k, v := range defaultCfg {
		merged[k] = v
	}
	for k, v := range resolved {
		merged[k] = v
	}
	return merged
}

// clientConfig returns the subset of config serialized to the client as
// window.__SARDE__.pluginConfig.telescope. The exclude patterns are build-time
// only and never reach the browser. strings carries the resolved i18n labels.
func (tc telescopeConfig) clientConfig(strings map[string]string) map[string]any {
	return map[string]any{
		"shortcut_key": tc.shortcutKey,
		"max_results":  tc.maxResults,
		"max_recent":   tc.maxRecent,
		"max_pinned":   tc.maxPinned,
		"debounce_ms":  tc.debounceMs,
		"default_tab":  tc.defaultTab,
		"strings":      strings,
	}
}
