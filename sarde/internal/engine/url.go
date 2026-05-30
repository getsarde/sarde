package engine

import "strings"

// URLResolver resolves site-root-relative, prefix-free paths into final URLs.
// It is the single chokepoint for basePath (and later lang/version) prefixing.
type URLResolver struct {
	BasePath string // normalized: "/docs/" or "/"
	BaseURL  string // origin only: "https://example.com"
}

// URL resolves a site-root-relative, prefix-free path to a final root-relative URL.
//
// relPath: e.g. "/guides/auth/" — always treated as site-root-relative.
// lang:    PHASE B — accepted but ignored in Phase A. Pass "".
// version: PHASE D — accepted but ignored in Phase A. Pass "".
//
// Idempotent: if relPath is already prefixed with basePath, it is NOT prefixed again.
func (r *URLResolver) URL(relPath, lang, version string) string {
	return applyBasePath(r.BasePath, relPath)
}

// AbsURL returns the fully-qualified URL (origin + resolved path).
func (r *URLResolver) AbsURL(relPath, lang, version string) string {
	origin := strings.TrimRight(r.BaseURL, "/")
	return origin + r.URL(relPath, lang, version)
}

// applyBasePath joins basePath + relPath idempotently. If relPath is already
// prefixed with basePath (the classic double-prefix footgun), it is returned
// normalized but NOT prefixed a second time.
//
// basePath is assumed already normalized (canonical "/docs/" or "/").
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
