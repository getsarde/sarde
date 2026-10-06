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

func TestFoldThemeShortcuts_QuotesBareNamesAndAddsAFallback(t *testing.T) {
	th := ThemeSettings{
		FontFamily:  "Plus Jakarta Sans", // catalog sans-serif
		FontHeading: "Source Sans 3",     // a digit word: invalid CSS unquoted
		FontMono:    "Fira Code",         // catalog monospace
	}
	FoldThemeShortcuts(&th)
	want := map[string]string{
		"font-sans":    "'Plus Jakarta Sans', system-ui, sans-serif",
		"font-heading": "'Source Sans 3', system-ui, sans-serif",
		"font-mono":    "'Fira Code', ui-monospace, monospace",
	}
	for token, v := range want {
		if got := th.Overrides[token]; got != v {
			t.Errorf("Overrides[%q] = %q, want %q", token, got, v)
		}
	}
}

func TestFoldThemeShortcuts_FallbackFollowsCategoryThenRole(t *testing.T) {
	th := ThemeSettings{
		FontFamily:  "Merriweather",    // catalog serif
		FontHeading: "Dancing Script",  // catalog handwriting
		FontMono:    "My Company Mono", // unknown: the role decides
	}
	FoldThemeShortcuts(&th)
	want := map[string]string{
		"font-sans":    "'Merriweather', serif",
		"font-heading": "'Dancing Script', cursive",
		"font-mono":    "'My Company Mono', ui-monospace, monospace",
	}
	for token, v := range want {
		if got := th.Overrides[token]; got != v {
			t.Errorf("Overrides[%q] = %q, want %q", token, got, v)
		}
	}
}

func TestFoldThemeShortcuts_LeavesStacksAndGenericsAlone(t *testing.T) {
	th := ThemeSettings{
		FontFamily:  "system-ui",
		FontHeading: "'Playfair Display', Georgia, serif",
		FontMono:    `"Fira Code"`,
	}
	FoldThemeShortcuts(&th)
	want := map[string]string{
		"font-sans":    "system-ui",
		"font-heading": "'Playfair Display', Georgia, serif",
		"font-mono":    `"Fira Code"`,
	}
	for token, v := range want {
		if got := th.Overrides[token]; got != v {
			t.Errorf("Overrides[%q] = %q, want %q", token, got, v)
		}
	}
}

func TestMergeTheme_WebFonts(t *testing.T) {
	base := ThemeSettings{WebFonts: "google"}
	mergeTheme(&base, &ThemeSettings{})
	if base.WebFonts != "google" {
		t.Errorf("empty layer changed web_fonts to %q", base.WebFonts)
	}
	mergeTheme(&base, &ThemeSettings{WebFonts: "bunny"})
	if base.WebFonts != "bunny" {
		t.Errorf("later layer did not win: web_fonts = %q", base.WebFonts)
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
