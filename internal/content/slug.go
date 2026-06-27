package content

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	// Matches one or more digits followed by a hyphen or underscore at the start.
	numericPrefixRe = regexp.MustCompile(`^(\d+)[-_](.+)$`)
	// Matches a Markdown H1 heading at the start of a line.
	h1Re = regexp.MustCompile(`(?m)^#\s+(.+)$`)
	// Matches characters that are not alphanumeric or hyphens.
	nonSlugRe = regexp.MustCompile(`[^a-z0-9-]+`)
	// Matches multiple consecutive hyphens.
	multiHyphenRe = regexp.MustCompile(`-{2,}`)
)

// Slugify converts a string to a URL-safe slug.
// Lowercase, spaces/underscores become hyphens, non-alphanumeric stripped, collapsed.
func Slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")
	s = nonSlugRe.ReplaceAllString(s, "-")
	s = multiHyphenRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// ExtractNumericPrefix parses a leading numeric prefix from a filename (without extension).
// "01-intro" returns (1, "intro", true). "intro" returns (0, "intro", false).
func ExtractNumericPrefix(name string) (weight int, slug string, found bool) {
	matches := numericPrefixRe.FindStringSubmatch(name)
	if matches == nil {
		return 0, name, false
	}
	w, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, name, false
	}
	return w, matches[2], true
}

// ExtractFirstH1 finds the first Markdown H1 heading (# Title) in raw markdown.
// Returns empty string if no H1 is found.
func ExtractFirstH1(markdown string) string {
	matches := h1Re.FindStringSubmatch(markdown)
	if matches == nil {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

// FilenameToTitle converts a filename to a human-readable title.
// Strips extension, strips numeric prefix, replaces hyphens/underscores with spaces, title cases.
func FilenameToTitle(filename string) string {
	name := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	// Strip numeric prefix if present
	if _, clean, found := ExtractNumericPrefix(name); found {
		name = clean
	}
	// Replace hyphens and underscores with spaces
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	// Title case
	return titleCase(name)
}

// FilenameSlug extracts a slug from a filename, stripping extension and numeric prefix.
func FilenameSlug(filename string) (slug string, weight int) {
	name := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	if w, clean, found := ExtractNumericPrefix(name); found {
		return Slugify(clean), w
	}
	return Slugify(name), 0
}

func titleCase(s string) string {
	upper := true
	return strings.Map(func(r rune) rune {
		if upper && unicode.IsLetter(r) {
			upper = false
			return unicode.ToUpper(r)
		}
		if r == ' ' {
			upper = true
		}
		return r
	}, s)
}
