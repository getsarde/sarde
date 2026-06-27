package htmlutil

import "strings"

// EscapeHTML escapes the five HTML special characters.
func EscapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

// IsAllowedHref returns false when the href starts with a blocked scheme
// (e.g. "javascript:", "data:", "vbscript:"). All other schemes are allowed.
func IsAllowedHref(href string, blockedSchemes []string) bool {
	lower := strings.ToLower(strings.TrimSpace(href))
	for _, blocked := range blockedSchemes {
		if strings.HasPrefix(lower, blocked) {
			return false
		}
	}
	return true
}
