package content

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var (
	// Matches one or more digits followed by a hyphen or underscore at the start.
	numericPrefixRe = regexp.MustCompile(`^(\d+)[-_](.+)$`)
	// Matches a blog-style date prefix: "2024-03-15-my-post".
	datePrefixRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-(.+)$`)
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

// SlugOrHash is Slugify, or, when nothing URL-safe is left (a name written
// only in non-Latin script, such as "日本語"), the first 8 hex digits of the
// name's SHA-256. Permalinks, page slugs and taxonomy terms share it, so a
// page's slug matches its URL and a term's lookup matches its page.
func SlugOrHash(s string) string {
	if slug := Slugify(s); slug != "" {
		return slug
	}
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:4])
}

// SplitDatePrefix splits a "YYYY-MM-DD-rest" name (without extension) into
// its date and the rest. ok is false without a valid date prefix.
func SplitDatePrefix(name string) (date time.Time, rest string, ok bool) {
	m := datePrefixRe.FindStringSubmatch(name)
	if m == nil {
		return time.Time{}, name, false
	}
	t, err := time.Parse("2006-01-02", m[1])
	if err != nil {
		return time.Time{}, name, false
	}
	return t, m[2], true
}

// FileSlug is the slug of a regular content file, from its name without
// extension, and the sidebar order its name carries. A date prefix and then
// a numeric prefix are stripped ("2024-01-15-01-intro" gives "intro" and
// order 1); a name left with nothing URL-safe falls back to the whole name's
// slug, then to its hash (SlugOrHash). ComputePermalink and the inferrer
// both use it, so a page's slug and its URL agree.
func FileSlug(nameNoExt string) (slug string, order int) {
	rest := nameNoExt
	if _, r, ok := SplitDatePrefix(nameNoExt); ok {
		rest = r
	}
	if w, clean, found := ExtractNumericPrefix(rest); found {
		rest, order = clean, w
	}
	if slug = Slugify(rest); slug == "" {
		slug = SlugOrHash(nameNoExt)
	}
	return slug, order
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
	// Strip a date prefix, then a numeric prefix, if present
	if _, rest, ok := SplitDatePrefix(name); ok {
		name = rest
	}
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
