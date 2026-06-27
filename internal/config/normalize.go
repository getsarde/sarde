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
