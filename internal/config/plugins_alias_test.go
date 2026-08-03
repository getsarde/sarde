package config

import (
	"reflect"
	"testing"
)

func TestNormalizePluginSlugs(t *testing.T) {
	known := []string{"scroll_to_top", "keyboard_nav", "content_lint", "search"}
	p := PluginSettings{
		Enabled:  []string{"scroll-to-top", "search", "my-widget"},
		Disabled: []string{"keyboard-nav"},
		Config: map[string]map[string]any{
			"scroll-to-top": {"threshold": 25},
		},
	}

	normalizePluginSlugs(&p, known)

	// my-widget stays: my_widget is not a known plugin, so the hyphenated
	// spelling is an external slug, not a legacy alias.
	if want := []string{"scroll_to_top", "search", "my-widget"}; !reflect.DeepEqual(p.Enabled, want) {
		t.Errorf("Enabled = %v, want %v", p.Enabled, want)
	}
	if want := []string{"keyboard_nav"}; !reflect.DeepEqual(p.Disabled, want) {
		t.Errorf("Disabled = %v, want %v", p.Disabled, want)
	}
	cfg, ok := p.Config["scroll_to_top"]
	if !ok || cfg["threshold"] != 25 {
		t.Errorf("Config not re-keyed to canonical slug: %v", p.Config)
	}
	if _, ok := p.Config["scroll-to-top"]; ok {
		t.Error("legacy config key must not survive")
	}
}

func TestNormalizePluginSlugs_BothSpellingsNoDoubleRegister(t *testing.T) {
	known := []string{"scroll_to_top"}
	p := PluginSettings{
		Enabled: []string{"scroll-to-top", "scroll_to_top"},
		Config: map[string]map[string]any{
			"scroll-to-top": {"threshold": 10},
			"scroll_to_top": {"threshold": 99},
		},
	}

	normalizePluginSlugs(&p, known)

	if want := []string{"scroll_to_top"}; !reflect.DeepEqual(p.Enabled, want) {
		t.Errorf("Enabled = %v, want %v (no double-register)", p.Enabled, want)
	}
	if len(p.Config) != 1 {
		t.Fatalf("Config = %v, want single canonical entry", p.Config)
	}
	if p.Config["scroll_to_top"]["threshold"] != 99 {
		t.Errorf("canonical config block must win, got %v", p.Config["scroll_to_top"])
	}
}

func TestNormalizePluginSlugs_NoKnownPluginsIsNoOp(t *testing.T) {
	p := PluginSettings{Enabled: []string{"scroll-to-top"}}
	normalizePluginSlugs(&p, nil)
	if want := []string{"scroll-to-top"}; !reflect.DeepEqual(p.Enabled, want) {
		t.Errorf("Enabled = %v, want %v (untouched without a known set)", p.Enabled, want)
	}
}
