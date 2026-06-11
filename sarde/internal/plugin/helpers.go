package plugin

import (
	"path"
	"strings"
)

// cfgString reads a string from plugin config with a fallback default.
func cfgString(cfg map[string]any, key, fallback string) string {
	if cfg == nil {
		return fallback
	}
	v, ok := cfg[key]
	if !ok {
		return fallback
	}
	s, ok := v.(string)
	if !ok {
		return fallback
	}
	return s
}

// cfgBool reads a bool from plugin config with a fallback default.
func cfgBool(cfg map[string]any, key string, fallback bool) bool {
	if cfg == nil {
		return fallback
	}
	v, ok := cfg[key]
	if !ok {
		return fallback
	}
	b, ok := v.(bool)
	if !ok {
		return fallback
	}
	return b
}

// cfgInt reads an int from plugin config with a fallback default.
func cfgInt(cfg map[string]any, key string, fallback int) int {
	if cfg == nil {
		return fallback
	}
	v, ok := cfg[key]
	if !ok {
		return fallback
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	default:
		return fallback
	}
}

// cfgStringSlice reads a string slice from plugin config.
func cfgStringSlice(cfg map[string]any, key string) []string {
	if cfg == nil {
		return nil
	}
	v, ok := cfg[key]
	if !ok {
		return nil
	}
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		var result []string
		for _, item := range s {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return nil
	}
}

// shouldExclude checks if a URL path matches any exclude pattern.
// Uses path.Match (slash-separated glob semantics on every platform);
// filepath.Match would make patterns behave differently on Windows.
func shouldExclude(urlPath string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := path.Match(pattern, urlPath); matched {
			return true
		}
		// Also check without trailing slash.
		trimmed := strings.TrimRight(urlPath, "/")
		if matched, _ := path.Match(pattern, trimmed); matched {
			return true
		}
	}
	return false
}
