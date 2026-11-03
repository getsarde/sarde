package theme

import (
	"fmt"
	"sort"
	"strings"
)

// ValidateOverrides checks that all keys in overrides are known token names.
// Returns an error listing unknown tokens with did-you-mean suggestions.
func ValidateOverrides(label string, overrides map[string]string, known map[string]bool) error {
	var unknown []string
	for key := range overrides {
		if !known[key] {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	var msgs []string
	for _, key := range unknown {
		if suggestion := SuggestToken(key, known); suggestion != "" {
			msgs = append(msgs, fmt.Sprintf("unknown token %q (did you mean %q?)", key, suggestion))
		} else {
			msgs = append(msgs, fmt.Sprintf("unknown token %q", key))
		}
	}
	return fmt.Errorf("%s: %s", label, strings.Join(msgs, "; "))
}

// ThemeTokenWarnings checks the token names a theme.yaml sets, in tokens,
// dark_tokens and each preset, against the known tokens. An unknown token
// does nothing, so each is reported, with a did-you-mean suggestion where one
// is close. Warnings rather than errors: a third-party theme with a stray
// token keeps building after an upgrade.
func ThemeTokenWarnings(t *Theme, known map[string]bool) []string {
	if t == nil {
		return nil
	}
	label := "theme.yaml"
	if t.Name != "" {
		label = "theme " + strconvQuote(t.Name) + ": theme.yaml"
	}
	var msgs []string
	check := func(field string, tokens map[string]string) {
		if err := ValidateOverrides(label+" "+field, tokens, known); err != nil {
			msgs = append(msgs, err.Error())
		}
	}
	check("tokens", t.Tokens)
	check("dark_tokens", t.DarkTokens)
	names := make([]string, 0, len(t.Presets))
	for name := range t.Presets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		p := t.Presets[name]
		check("presets."+name+".tokens", p.Tokens)
		check("presets."+name+".dark_tokens", p.DarkTokens)
	}
	return msgs
}

func strconvQuote(s string) string { return fmt.Sprintf("%q", s) }

// ResolveTokens merges light-mode tokens through the 4-layer cascade.
// Order (last wins): defaults → theme.Tokens → preset.Tokens → overrides.
func ResolveTokens(defaults map[string]string, t *Theme, presetName string, overrides map[string]string) map[string]string {
	result := copyMap(defaults)

	if t != nil {
		mergeInto(result, t.Tokens)

		if presetName != "" {
			if preset, ok := t.Presets[presetName]; ok {
				mergeInto(result, preset.Tokens)
			}
		}
	}

	mergeInto(result, overrides)
	return result
}

// ResolveDarkTokens merges dark-mode tokens through the 4-layer cascade.
func ResolveDarkTokens(defaults map[string]string, t *Theme, presetName string, overrides map[string]string) map[string]string {
	result := copyMap(defaults)

	if t != nil {
		mergeInto(result, t.DarkTokens)

		if presetName != "" {
			if preset, ok := t.Presets[presetName]; ok {
				mergeInto(result, preset.DarkTokens)
			}
		}
	}

	mergeInto(result, overrides)
	return result
}

func copyMap(m map[string]string) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func mergeInto(dst, src map[string]string) {
	for k, v := range src {
		if v != "" {
			dst[k] = v
		}
	}
}
