package engine

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

// URLResolver resolves site-root-relative, prefix-free paths into final URLs.
// It is the single chokepoint for basePath, lang, and version prefixing.
type URLResolver struct {
	BasePath    string // normalized: "/docs/" or "/"
	BaseURL     string // origin only: "https://example.com"
	I18nEnabled bool
	DefaultLang string
	Strategy    string          // "prefix-except-default"
	Languages   map[string]bool // set of known language codes

	CollectionMounts []string        // ["/docs", "/blog"] — populated by builder
	VersionIDs       map[string]bool // union of all version IDs across versioned collections
}

// URL resolves a site-root-relative, prefix-free path to a final root-relative URL.
//
// relPath: e.g. "/docs/guides/auth/" — always treated as site-root-relative.
// lang:    language code; "" means default language. Non-default languages
//
//	get a /<lang>/ segment inserted (prefix-except-default strategy).
//
// version: version ID for non-latest versions (e.g. "v1"); "" for latest/unversioned.
//
//	Inserted AFTER the collection mount, not as a global prefix.
func (r *URLResolver) URL(relPath, lang, version string) string {
	rel := cleanJoin(relPath)

	// Strip basePath if already present (idempotency: a resolved URL passed
	// back through the resolver must not get double-prefixed).
	rel = r.stripBasePath(rel)

	// version — per-collection, inserted AFTER the collection mount.
	// Done first while rel is still the bare collection path.
	if r.needVersionSegment(version) {
		rel = r.insertVersionSegment(rel, version)
	}

	// lang — site-wide global prefix, outer of the collection.
	if r.needLangSegment(lang) {
		resolved := lang
		if resolved == "" {
			resolved = r.DefaultLang
		}
		rel = r.insertLangSegment(rel, resolved)
	}

	return applyBasePath(r.BasePath, rel)
}

// CacheKey returns a deterministic digest of every field that affects URL
// resolution. The page-render cache must fold this into its content hash:
// rendered HTML embeds resolved links, so a change to base path, base URL,
// i18n, version, or collection layout must bust otherwise-identical content.
// Maps are sorted so the key is stable across map-iteration order.
func (r *URLResolver) CacheKey() string {
	langs := sortedSetKeys(r.Languages)
	versions := sortedSetKeys(r.VersionIDs)
	mounts := append([]string(nil), r.CollectionMounts...)
	sort.Strings(mounts)

	raw := fmt.Sprintf("bp=%s\x00bu=%s\x00i18n=%t\x00dl=%s\x00st=%s\x00langs=%s\x00mounts=%s\x00vers=%s",
		r.BasePath, r.BaseURL, r.I18nEnabled, r.DefaultLang, r.Strategy,
		strings.Join(langs, ","), strings.Join(mounts, ","), strings.Join(versions, ","))
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}

func sortedSetKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// AbsURL returns the fully-qualified URL (origin + resolved path).
func (r *URLResolver) AbsURL(relPath, lang, version string) string {
	origin := strings.TrimRight(r.BaseURL, "/")
	return origin + r.URL(relPath, lang, version)
}

// OutputRelPath returns the on-disk output path: version- and lang-prefixed but
// WITHOUT basePath. Used to compute filesystem write paths where version and lang
// create real directories but basePath does not (the web server's mount handles basePath).
func (r *URLResolver) OutputRelPath(relPath, lang, version string) string {
	rel := cleanJoin(relPath)
	if r.needVersionSegment(version) {
		rel = r.insertVersionSegment(rel, version)
	}
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

func (r *URLResolver) needVersionSegment(version string) bool {
	return version != ""
}

// insertVersionSegment places "/<version>" immediately after the collection mount.
// Idempotent: if the post-mount segment is already a version ID, it is a no-op.
func (r *URLResolver) insertVersionSegment(rel, version string) string {
	mount := r.collectionMountFor(rel)
	if mount == "" {
		return rel
	}
	rest := strings.TrimPrefix(rel, mount)
	if r.IsVersionID(firstSegment(rest)) {
		return rel
	}
	return cleanJoin(mount, "/"+version, rest)
}

// IsVersionID reports whether seg matches any configured version ID
// (union over all versioned collections).
func (r *URLResolver) IsVersionID(seg string) bool {
	return r.VersionIDs[seg]
}

// collectionMountFor returns the longest collection mount that prefixes rel.
// Returns "" if no mount matches (e.g. standalone pages).
func (r *URLResolver) collectionMountFor(rel string) string {
	best := ""
	for _, m := range r.CollectionMounts {
		if rel == m || strings.HasPrefix(rel, m+"/") {
			if len(m) > len(best) {
				best = m
			}
		}
	}
	return best
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
