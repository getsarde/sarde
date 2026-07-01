package attrutil

import "regexp"

var pairRe = regexp.MustCompile(`([\w-]+)(?:=(?:"([^"]*)"|'([^']*)'))?`)

// Parse extracts key="value", key='value', and bare flags from a
// space-separated attribute string. Bare flags are stored with an
// empty-string value; use Has to test presence.
func Parse(s string) map[string]string {
	attrs := make(map[string]string)
	if s == "" {
		return attrs
	}
	for _, m := range pairRe.FindAllStringSubmatch(s, -1) {
		switch {
		case m[2] != "":
			attrs[m[1]] = m[2]
		case m[3] != "":
			attrs[m[1]] = m[3]
		default:
			attrs[m[1]] = ""
		}
	}
	return attrs
}

// Has reports whether key is present at all (for bare boolean flags).
func Has(attrs map[string]string, key string) bool {
	_, ok := attrs[key]
	return ok
}

// Bool reports whether attrs[key] is the literal string "true".
func Bool(attrs map[string]string, key string) bool {
	return attrs[key] == "true"
}
