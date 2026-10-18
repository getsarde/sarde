package content

import (
	"path"
	"path/filepath"
	"strings"
)

// ComputePermalink returns the clean URL for a content file.
// All permalinks end with "/" and use forward slashes.
//
// Examples:
//
//	content/_index.md              → "/"
//	content/about.md               → "/about/"
//	content/docs/_index.md         → "/docs/"
//	content/docs/getting-started.md → "/docs/getting-started/"
//	content/docs/guide/index.md    → "/docs/guide/"
// PrefixPermalink prepends a language prefix to a permalink for non-default languages.
// Used when generating fallback pages that need language-prefixed URLs.
// For the default language, it returns the permalink unchanged.
func PrefixPermalink(permalink, lang, defaultLang string) string {
	if lang == defaultLang || lang == "" {
		return permalink
	}
	return "/" + lang + permalink
}

// ComputePermalinkFromRelPath computes a permalink from a forward-slash
// relative path (e.g. cf.LangRelPath). Unlike ComputePermalink, this operates
// on a path already stripped of any language directory prefix, producing a
// language-free RelPermalink that is identical across translations.
func ComputePermalinkFromRelPath(relPath string) string {
	base := path.Base(relPath)
	dir := path.Dir(relPath)
	if dir == "." {
		dir = ""
	}

	switch {
	case base == "_index.md":
		if dir == "" {
			return "/"
		}
		return "/" + dir + "/"
	case base == "index.md":
		if dir == "" {
			return "/"
		}
		return "/" + dir + "/"
	default:
		slug, _ := FileSlug(strings.TrimSuffix(base, path.Ext(base)))
		return fileURL(dir, slug)
	}
}

// LegacyDatePermalink returns the URL Sarde 1.5 gave a date-prefixed file,
// such as "/blog/03-15-hello/" for "blog/2026-03-15-hello.md" (1.5 read the
// year as a numeric order prefix and kept the rest of the date). It returns
// "" for any other file, and when the 1.5 URL equals the current one. relPath
// is forward-slash and relative to the content root, like
// ComputePermalinkFromRelPath takes. The build redirects the 1.5 URL to the
// current one; drop this with those redirects in the next major release.
func LegacyDatePermalink(relPath string) string {
	base := path.Base(relPath)
	if base == "_index.md" || base == "index.md" {
		return ""
	}
	name := strings.TrimSuffix(base, path.Ext(base))
	if _, _, ok := SplitDatePrefix(name); !ok {
		return ""
	}
	legacy := legacyFileSlug(name)
	if slug, _ := FileSlug(name); legacy == slug {
		return ""
	}
	dir := path.Dir(relPath)
	if dir == "." {
		dir = ""
	}
	return fileURL(dir, legacy)
}

// fileURL is the URL of a regular file with the given slug in dir (a
// forward-slash path relative to the content root, "" for the root).
func fileURL(dir, slug string) string {
	if dir == "" {
		return "/" + slug + "/"
	}
	return "/" + dir + "/" + slug + "/"
}

// PermalinkVars holds the values available for pattern interpolation.
type PermalinkVars struct {
	Slug       string
	Year       string
	Month      string
	Day        string
	Section    string
	Collection string
	Title      string
}

// ComputePatternPermalink generates a permalink from a pattern string.
// Supported variables: :slug, :year, :month, :day, :section, :collection, :title
func ComputePatternPermalink(pattern string, vars PermalinkVars) string {
	r := strings.NewReplacer(
		":slug", vars.Slug,
		":year", vars.Year,
		":month", vars.Month,
		":day", vars.Day,
		":section", vars.Section,
		":collection", vars.Collection,
		":title", vars.Title,
	)
	result := r.Replace(pattern)
	if !strings.HasPrefix(result, "/") {
		result = "/" + result
	}
	if !strings.HasSuffix(result, "/") {
		result = result + "/"
	}
	for strings.Contains(result, "//") {
		result = strings.ReplaceAll(result, "//", "/")
	}
	return result
}

func ComputePermalink(contentDir, filePath string) string {
	rel, err := filepath.Rel(contentDir, filePath)
	if err != nil {
		rel = filePath
	}
	// Normalize to forward slashes
	rel = filepath.ToSlash(rel)

	base := filepath.Base(rel)
	dir := filepath.Dir(rel)
	if dir == "." {
		dir = ""
	}
	dir = filepath.ToSlash(dir)

	switch {
	case base == "_index.md":
		// Section or home index
		if dir == "" {
			return "/"
		}
		return "/" + dir + "/"

	case base == "index.md":
		// Page bundle — URL is the parent directory
		if dir == "" {
			return "/"
		}
		return "/" + dir + "/"

	default:
		// Regular page or standalone
		slug, _ := FileSlug(strings.TrimSuffix(base, filepath.Ext(base)))
		return fileURL(dir, slug)
	}
}
