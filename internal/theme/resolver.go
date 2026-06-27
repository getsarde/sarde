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
