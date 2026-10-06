package config

import (
	"strconv"
	"strings"

	"github.com/getsarde/sarde/internal/webfonts"
)

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
		"font-sans":    fontStack("font-sans", t.FontFamily),
		"font-mono":    fontStack("font-mono", t.FontMono),
		"font-heading": fontStack("font-heading", t.FontHeading),
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

// fontStack turns a bare family name from a font_* shortcut into a valid
// CSS stack: quoted, with a generic fallback. Configs often give just the
// name (`font_family: Plus Jakarta Sans`, which is what Studio writes);
// left unquoted, a name such as `Source Sans 3` is invalid CSS, and with no
// fallback a missing font drops to the browser default. A value that is
// already a stack (it has a comma or a quote) or a generic keyword such as
// system-ui is returned as written.
func fontStack(token, value string) string {
	name := strings.TrimSpace(value)
	if name == "" || strings.ContainsAny(name, `,'"`) || webfonts.IsGeneric(name) {
		return value
	}
	return "'" + strings.ReplaceAll(name, `\`, `\\`) + "', " + fontFallback(token, name)
}

// fontFallback picks the generic families after a bare name: from its
// Google Fonts category when the catalog knows it, else from the token.
func fontFallback(token, name string) string {
	if _, category, ok := webfonts.Lookup(name); ok {
		switch category {
		case "serif":
			return "serif"
		case "monospace":
			return "ui-monospace, monospace"
		case "handwriting":
			return "cursive"
		}
	}
	if token == "font-mono" {
		return "ui-monospace, monospace"
	}
	return "system-ui, sans-serif"
}
