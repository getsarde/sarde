package engine

import "strings"

// URLResolver resolves site-root-relative, prefix-free paths into final URLs.
// It is the single chokepoint for basePath and lang (and later version) prefixing.
type URLResolver struct {
	BasePath    string // normalized: "/docs/" or "/"
	BaseURL     string // origin only: "https://example.com"
	I18nEnabled bool
	DefaultLang string
	Strategy    string          // "prefix-except-default"
	Languages   map[string]bool // set of known language codes
}

// URL resolves a site-root-relative, prefix-free path to a final root-relative URL.
//
// relPath: e.g. "/guides/auth/" — always treated as site-root-relative.
// lang:    language code; "" means default language. Non-default languages
//
//	get a /<lang>/ segment inserted (prefix-except-default strategy).
//
// version: PHASE D — accepted but ignored. Pass "".
func (r *URLResolver) URL(relPath, lang, version string) string {
	rel := cleanJoin(relPath)

	// Strip basePath if already present (idempotency: a resolved URL passed
	// back through the resolver must not get double-prefixed).
	rel = r.stripBasePath(rel)

	// [version slot — Phase D]

	// lang segment
	if r.needLangSegment(lang) {
		resolved := lang
		if resolved == "" {
			resolved = r.DefaultLang
		}
		rel = r.insertLangSegment(rel, resolved)
	}

	return applyBasePath(r.BasePath, rel)
}

// AbsURL returns the fully-qualified URL (origin + resolved path).
func (r *URLResolver) AbsURL(relPath, lang, version string) string {
	origin := strings.TrimRight(r.BaseURL, "/")
	return origin + r.URL(relPath, lang, version)
}

// OutputRelPath returns the on-disk output path: lang-prefixed but WITHOUT basePath.
// Used to compute filesystem write paths where lang creates real directories
// but basePath does not (the web server's mount handles basePath).
func (r *URLResolver) OutputRelPath(relPath, lang, version string) string {
	rel := cleanJoin(relPath)
	if r.needLangSegment(lang) {
		resolved := lang
		if resolved == "" {
			resolved = r.DefaultLang
		}
		rel = r.insertLangSegment(rel, resolved)
	}
	return rel
}

func (r *URLResolver) stripBasePath(rel string) string {
	if r.BasePath == "/" {
		return rel
	}
	bp := strings.TrimRight(r.BasePath, "/")
	if rel == bp || strings.HasPrefix(rel, bp+"/") {
		return rel[len(bp):]
	}
	return rel
}

func (r *URLResolver) needLangSegment(lang string) bool {
	if !r.I18nEnabled {
		return false
	}
	resolved := lang
	if resolved == "" {
		resolved = r.DefaultLang
	}
	return resolved != r.DefaultLang
}

func (r *URLResolver) insertLangSegment(rel, code string) string {
	if r.Languages[firstSegment(rel)] {
		return rel
	}
	return cleanJoin("/"+code, rel)
}

func firstSegment(rel string) string {
	for _, p := range strings.Split(rel, "/") {
		if p != "" {
			return p
		}
	}
	return ""
}

// applyBasePath joins basePath + relPath. basePath is assumed already
// normalized (canonical "/docs/" or "/").
func applyBasePath(basePath, relPath string) string {
	rel := cleanJoin(relPath)
	if basePath == "/" {
		return rel
	}
	return cleanJoin(basePath, rel)
}

// cleanJoin joins path segments with exactly one slash between them,
// guarantees a single leading slash, and preserves a trailing slash if
// the final non-empty segment had one. Empty segments are skipped.
func cleanJoin(segments ...string) string {
	trailing := false
	var parts []string
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		trailing = strings.HasSuffix(seg, "/")
		for _, p := range strings.Split(seg, "/") {
			if p != "" {
				parts = append(parts, p)
			}
		}
	}
	if len(parts) == 0 {
		return "/"
	}
	out := "/" + strings.Join(parts, "/")
	if trailing {
		out += "/"
	}
	return out
}
