package theme

import (
	"testing"
)

func TestResolveTokens_DefaultsOnly(t *testing.T) {
	result := ResolveTokens(DefaultTokens(), nil, "", nil)

	if result["primary"] != "#6366f1" {
		t.Errorf("primary: got %q", result["primary"])
	}
	if result["bg"] != "#ffffff" {
		t.Errorf("bg: got %q", result["bg"])
	}
}

func TestResolveTokens_ThemeOverridesDefaults(t *testing.T) {
	theme := &Theme{
		Tokens: map[string]string{"primary": "#3b82f6"},
	}
	result := ResolveTokens(DefaultTokens(), theme, "", nil)

	if result["primary"] != "#3b82f6" {
		t.Errorf("primary: got %q", result["primary"])
	}
	// Non-overridden defaults should persist.
	if result["bg"] != "#ffffff" {
		t.Errorf("bg: got %q", result["bg"])
	}
}

func TestResolveTokens_PresetOverridesTheme(t *testing.T) {
	theme := &Theme{
		Tokens: map[string]string{"primary": "#3b82f6"},
		Presets: map[string]Preset{
			"ocean": {Tokens: map[string]string{"primary": "#0ea5e9"}},
		},
	}
	result := ResolveTokens(DefaultTokens(), theme, "ocean", nil)

	if result["primary"] != "#0ea5e9" {
		t.Errorf("primary: got %q", result["primary"])
	}
}

func TestResolveTokens_UserOverridesAll(t *testing.T) {
	theme := &Theme{
		Tokens: map[string]string{"primary": "#3b82f6"},
		Presets: map[string]Preset{
			"ocean": {Tokens: map[string]string{"primary": "#0ea5e9"}},
		},
	}
	overrides := map[string]string{"primary": "#e11d48"}
	result := ResolveTokens(DefaultTokens(), theme, "ocean", overrides)

	if result["primary"] != "#e11d48" {
		t.Errorf("primary: got %q", result["primary"])
	}
}

func TestResolveTokens_UnknownPreset(t *testing.T) {
	theme := &Theme{
		Tokens: map[string]string{"primary": "#3b82f6"},
	}
	result := ResolveTokens(DefaultTokens(), theme, "nonexistent", nil)

	// Unknown preset should be ignored; theme tokens apply.
	if result["primary"] != "#3b82f6" {
		t.Errorf("primary: got %q", result["primary"])
	}
}

func TestResolveDarkTokens(t *testing.T) {
	theme := &Theme{
		DarkTokens: map[string]string{"bg": "#111111"},
		Presets: map[string]Preset{
			"ocean": {DarkTokens: map[string]string{"bg": "#222222"}},
		},
	}
	result := ResolveDarkTokens(DefaultDarkTokens(), theme, "ocean", nil)

	if result["bg"] != "#222222" {
		t.Errorf("dark bg: got %q", result["bg"])
	}
}
