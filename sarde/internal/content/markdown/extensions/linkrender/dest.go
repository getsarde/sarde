package linkrender

import (
	"path"
	"strings"
)

// LinkKind classifies a Markdown link destination.
type LinkKind int

const (
	LinkExternal    LinkKind = iota // http://, https://, //, mailto:, tel:
	LinkRelative                    // ./ or ../
	LinkContentRoot                 // leading / (within collection)
	LinkAnchorOnly                  // starts with #
	LinkAmbiguous                   // bare name — rejected as ambiguous
)

// ParsedDest is a classified, cleaned link destination.
type ParsedDest struct {
	Kind     LinkKind
	Path     string // cleaned path, .md/.mdx extension stripped
	Fragment string // without '#'
	Query    string // without '?'
	Raw      string // original href as written
}

// ClassifyDest parses a raw Markdown link destination into its parts and
// classifies it. Only links with .md/.mdx extensions are treated as source-file
// references that need resolution. Links without these extensions (already-resolved
// URL paths like /about/) pass through as external.
func ClassifyDest(href string) ParsedDest {
	raw := href

	if href == "" {
		return ParsedDest{Kind: LinkExternal, Raw: raw}
	}

	// External: absolute URLs, protocol-relative, mailto, tel.
	if isExternal(href) {
		return ParsedDest{Kind: LinkExternal, Raw: raw}
	}

	// Split fragment.
	var fragment string
	if idx := strings.IndexByte(href, '#'); idx >= 0 {
		fragment = href[idx+1:]
		href = href[:idx]
	}

	// Split query.
	var query string
	if idx := strings.IndexByte(href, '?'); idx >= 0 {
		query = href[idx+1:]
		href = href[:idx]
	}

	// Anchor-only: no path component.
	if href == "" {
		return ParsedDest{
			Kind:     LinkAnchorOnly,
			Fragment: fragment,
			Query:    query,
			Raw:      raw,
		}
	}

	// Determine if this is a source-file reference (.md/.mdx extension present).
	hasMarkdownExt := hasMarkdownExtension(href)

	// Strip .md / .mdx extension if present.
	href = stripMarkdownExt(href)

	// Clean the path (normalize . / .. segments).
	href = path.Clean(href)

	// Relative: starts with ./ or ../ — always treated as source-file reference.
	if strings.HasPrefix(raw, "./") || strings.HasPrefix(raw, "../") {
		return ParsedDest{
			Kind:     LinkRelative,
			Path:     href,
			Fragment: fragment,
			Query:    query,
			Raw:      raw,
		}
	}

	// Content-root-relative: starts with / AND has .md/.mdx extension.
	// Without the extension, it's an already-resolved URL path — pass through.
	if strings.HasPrefix(href, "/") {
		if !hasMarkdownExt {
			return ParsedDest{Kind: LinkExternal, Raw: raw}
		}
		return ParsedDest{
			Kind:     LinkContentRoot,
			Path:     href,
			Fragment: fragment,
			Query:    query,
			Raw:      raw,
		}
	}

	// Bare name with .md extension — ambiguous (Hugo #4727 lesson).
	if hasMarkdownExt {
		return ParsedDest{
			Kind:     LinkAmbiguous,
			Path:     href,
			Fragment: fragment,
			Query:    query,
			Raw:      raw,
		}
	}

	// Bare name without .md extension — treat as already-resolved URL, pass through.
	return ParsedDest{Kind: LinkExternal, Raw: raw}
}

// hasMarkdownExtension reports whether the path ends with .md or .mdx.
func hasMarkdownExtension(p string) bool {
	return strings.HasSuffix(p, ".md") || strings.HasSuffix(p, ".mdx")
}

// isExternal returns true for absolute URLs, protocol-relative, and non-http schemes.
func isExternal(href string) bool {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return true
	}
	if strings.HasPrefix(href, "//") {
		return true
	}
	if strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") {
		return true
	}
	if strings.HasPrefix(href, "data:") || strings.HasPrefix(href, "javascript:") {
		return true
	}
	return false
}

// stripMarkdownExt removes .md or .mdx extension from a path.
func stripMarkdownExt(p string) string {
	if strings.HasSuffix(p, ".md") {
		return p[:len(p)-3]
	}
	if strings.HasSuffix(p, ".mdx") {
		return p[:len(p)-4]
	}
	return p
}
