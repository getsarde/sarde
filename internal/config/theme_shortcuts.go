package config

import "strconv"

// Bounds for theme.font_scale, the multiplier on the --sd-text-* scale.
const (
	MinFontScale = 0.5
	MaxFontScale = 2.0
)

// FoldThemeShortcuts maps the convenience theme fields (accent_color,
// font_family, font_heading, font_scale, ...) into Theme.Overrides so they
// flow through the token cascade. An explicit Overrides entry wins over its
// shortcut. Both the CLI build and the sidecar's project manager call this,
// so the two paths resolve the same tokens.
func FoldThemeShortcuts(t *ThemeSettings) {
	accentVal := t.AccentColor
	if accentVal == "" {
		accentVal = t.PrimaryColor
	}
	shortcuts := map[string]string{
		"accent":       accentVal,
		"font-sans":    t.FontFamily,
		"font-mono":    t.FontMono,
		"font-heading": t.FontHeading,
	}
	if t.FontScale != 0 {
		shortcuts["text-scale"] = strconv.FormatFloat(t.FontScale, 'f', -1, 64)
	}
	if t.Overrides == nil {
		t.Overrides = make(map[string]string)
	}
	for token, val := range shortcuts {
		if val == "" {
			continue
		}
		if _, exists := t.Overrides[token]; !exists {
			t.Overrides[token] = val
		}
	}
}
