package config

import "strings"

// NormalizeBasePath canonicalizes a user-supplied base path to the internal form.
// The canonical form always has a leading and trailing slash (e.g. "/docs/"),
// except the root case which is just "/".
//
//	""            -> "/"
//	"/"           -> "/"
//	"docs"        -> "/docs/"
//	"/docs"       -> "/docs/"
//	"docs/"       -> "/docs/"
//	"/docs/"      -> "/docs/"
//	"  /docs/  "  -> "/docs/"
//	"//docs//"    -> "/docs/"
//	"/a/b/"       -> "/a/b/"
//	"a//b"        -> "/a/b/"
//	"///"         -> "/"
func NormalizeBasePath(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, "/")
	if s == "" {
		return "/"
	}
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '/' })
	return "/" + strings.Join(parts, "/") + "/"
}

// dateFormatPresets maps the friendly names accepted by theme.date_format to Go
// layouts. Any other value is treated as a raw Go layout.
var dateFormatPresets = map[string]string{
	"short": "Jan 2, 2006",
	"long":  "January 2, 2006",
	"iso":   "2006-01-02",
}

// NormalizeDateFormat resolves theme.date_format to a Go layout string.
// An empty value yields the "short" preset, matching the format the theme used
// before the setting existed.
//
//	""        -> "Jan 2, 2006"
//	"short"   -> "Jan 2, 2006"
//	"long"    -> "January 2, 2006"
//	"iso"     -> "2006-01-02"
//	"2006/01" -> "2006/01"   (raw Go layout, passed through)
func NormalizeDateFormat(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return dateFormatPresets["short"]
	}
	if layout, ok := dateFormatPresets[strings.ToLower(s)]; ok {
		return layout
	}
	return s
}
