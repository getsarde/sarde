package cfgutil

import (
	"sort"
	"strings"
)

// CamelAlias returns the deprecated camelCase spelling of a snake_case config
// key (show_tooltip -> showTooltip), or "" when the key contains no underscore
// and therefore has no distinct alias.
func CamelAlias(snake string) string {
	if !strings.Contains(snake, "_") {
		return ""
	}
	parts := strings.Split(snake, "_")
	var b strings.Builder
	b.WriteString(parts[0])
	for _, p := range parts[1:] {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		b.WriteString(p[1:])
	}
	return b.String()
}

// ResolveAliases re-keys deprecated camelCase spellings in userCfg to their
// canonical snake_case names, taken from the keys of defaults. When both
// spellings are present the canonical one wins; the alias is still reported in
// used. Keys that are neither canonical nor a known alias pass through
// unchanged. used is sorted for deterministic output.
func ResolveAliases(defaults, userCfg map[string]any) (map[string]any, []string) {
	if len(userCfg) == 0 {
		return userCfg, nil
	}
	aliasToCanon := make(map[string]string, len(defaults))
	for canon := range defaults {
		if alias := CamelAlias(canon); alias != "" && alias != canon {
			aliasToCanon[alias] = canon
		}
	}
	resolved := make(map[string]any, len(userCfg))
	var used []string
	for k, v := range userCfg {
		canon, isAlias := aliasToCanon[k]
		if !isAlias {
			resolved[k] = v
			continue
		}
		used = append(used, k)
		if _, ok := userCfg[canon]; ok {
			continue
		}
		resolved[canon] = v
	}
	sort.Strings(used)
	return resolved, used
}
