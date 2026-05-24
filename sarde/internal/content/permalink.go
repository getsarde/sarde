package content

import (
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
		name := strings.TrimSuffix(base, filepath.Ext(base))
		slug, _ := FilenameSlug(name + ".md")
		if slug == "" {
			slug = Slugify(name)
		}
		if dir == "" {
			return "/" + slug + "/"
		}
		return "/" + dir + "/" + slug + "/"
	}
}
