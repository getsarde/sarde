package cfgutil

import (
	"reflect"
	"testing"
)

func TestCamelAlias(t *testing.T) {
	tests := []struct{ in, want string }{
		{"show_tooltip", "showTooltip"},
		{"show_progress_ring", "showProgressRing"},
		{"border_radius", "borderRadius"},
		{"threshold", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := CamelAlias(tt.in); got != tt.want {
			t.Errorf("CamelAlias(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFieldNames(t *testing.T) {
	raw := []byte("fields:\n  b_field:\n    type: number\n    default: 1\n  a_field:\n    type: string\n")
	names, err := FieldNames(raw)
	if err != nil {
		t.Fatalf("FieldNames: %v", err)
	}
	// a_field declares no default and must still be listed; output is sorted.
	want := []string{"a_field", "b_field"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("FieldNames = %v, want %v", names, want)
	}
}

func TestResolveAliases_AliasOnly(t *testing.T) {
	defaults := map[string]any{"show_tooltip": false, "threshold": 30}
	resolved, used := ResolveAliases(defaults, map[string]any{"showTooltip": true})
	if resolved["show_tooltip"] != true {
		t.Errorf("alias not re-keyed: %v", resolved)
	}
	if _, ok := resolved["showTooltip"]; ok {
		t.Error("alias key must not survive in resolved config")
	}
	if !reflect.DeepEqual(used, []string{"showTooltip"}) {
		t.Errorf("used = %v, want [showTooltip]", used)
	}
}

func TestResolveAliases_CanonicalOnly(t *testing.T) {
	defaults := map[string]any{"show_tooltip": false}
	resolved, used := ResolveAliases(defaults, map[string]any{"show_tooltip": true})
	if resolved["show_tooltip"] != true {
		t.Errorf("canonical key mangled: %v", resolved)
	}
	if len(used) != 0 {
		t.Errorf("used = %v, want empty", used)
	}
}

func TestResolveAliases_BothPresentCanonicalWins(t *testing.T) {
	defaults := map[string]any{"show_tooltip": false}
	resolved, used := ResolveAliases(defaults, map[string]any{
		"showTooltip":  false,
		"show_tooltip": true,
	})
	if resolved["show_tooltip"] != true {
		t.Errorf("canonical spelling must win, resolved = %v", resolved)
	}
	if _, ok := resolved["showTooltip"]; ok {
		t.Error("alias key must not survive in resolved config")
	}
	if !reflect.DeepEqual(used, []string{"showTooltip"}) {
		t.Errorf("alias must still be reported when losing, used = %v", used)
	}
}

func TestResolveAliases_UnknownKeyPassesThrough(t *testing.T) {
	defaults := map[string]any{"show_tooltip": false}
	resolved, used := ResolveAliases(defaults, map[string]any{"mystery": 1})
	if resolved["mystery"] != 1 {
		t.Errorf("unknown key must pass through, resolved = %v", resolved)
	}
	if len(used) != 0 {
		t.Errorf("used = %v, want empty", used)
	}
}
