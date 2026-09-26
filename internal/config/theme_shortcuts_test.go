package config

import "testing"

func TestFoldThemeShortcuts_MapsFields(t *testing.T) {
	th := ThemeSettings{
		AccentColor: "#ff0066",
		FontFamily:  "Inter, sans-serif",
		FontMono:    "Fira Code, monospace",
		FontHeading: "Georgia, serif",
		FontScale:   1.25,
	}
	FoldThemeShortcuts(&th)

	want := map[string]string{
		"accent":       "#ff0066",
		"font-sans":    "Inter, sans-serif",
		"font-mono":    "Fira Code, monospace",
		"font-heading": "Georgia, serif",
		"text-scale":   "1.25",
	}
	for token, v := range want {
		if got := th.Overrides[token]; got != v {
			t.Errorf("Overrides[%q] = %q, want %q", token, got, v)
		}
	}
}

func TestFoldThemeShortcuts_ExplicitOverrideWins(t *testing.T) {
	th := ThemeSettings{
		FontHeading: "Georgia, serif",
		FontScale:   1.25,
		Overrides: map[string]string{
			"font-heading": "Merriweather, serif",
			"text-scale":   "0.9",
		},
	}
	FoldThemeShortcuts(&th)

	if got := th.Overrides["font-heading"]; got != "Merriweather, serif" {
		t.Errorf("font-heading = %q, want the explicit override", got)
	}
	if got := th.Overrides["text-scale"]; got != "0.9" {
		t.Errorf("text-scale = %q, want the explicit override", got)
	}
}

func TestFoldThemeShortcuts_UnsetFieldsAddNothing(t *testing.T) {
	var th ThemeSettings
	FoldThemeShortcuts(&th)
	if len(th.Overrides) != 0 {
		t.Errorf("empty theme folded into %v, want no overrides", th.Overrides)
	}
}

func TestMergeTheme_FontHeadingAndScale(t *testing.T) {
	base := ThemeSettings{FontHeading: "Georgia, serif", FontScale: 1.1}
	mergeTheme(&base, &ThemeSettings{})
	if base.FontHeading != "Georgia, serif" || base.FontScale != 1.1 {
		t.Errorf("empty layer changed base: %+v", base)
	}
	mergeTheme(&base, &ThemeSettings{FontHeading: "Merriweather", FontScale: 0.9})
	if base.FontHeading != "Merriweather" || base.FontScale != 0.9 {
		t.Errorf("later layer did not win: %+v", base)
	}
}
