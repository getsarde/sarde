package content

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/getsarde/sarde/internal/engine"
)

// dateFilenameRegex matches blog-style filenames like "2024-03-15-my-post"
// (without the .md extension). Capture 1 is the date, capture 2 is the slug.
var dateFilenameRegex = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-(.+)$`)

// Inferrer fills missing frontmatter values using filesystem metadata.
// This is the "zero-config magic" — users get sensible defaults without
// specifying title, date, weight, or slug in frontmatter.
type Inferrer struct {
	// LastUpdatedStrategy selects how missing Updated timestamps are resolved:
	// "git" (via `git log`), "mtime" (default), or "false"/"off"/"none" (disabled).
	LastUpdatedStrategy string

	// GitIndex is the batched git-history snapshot used by the "git" strategy.
	// Nil falls back to resolving each file with its own subprocess.
	GitIndex *GitLastModIndex
}

// Infer populates empty fields on a Page from the filesystem and content.
//
// Inference cascade:
//   - Title:    frontmatter �� first H1 in RawContent → filename title-cased
//   - Date:     frontmatter → file modification time
//   - Updated:  frontmatter → git commit time or file mtime, per LastUpdatedStrategy
//   - Weight:   frontmatter → numeric prefix from filename → 0
//   - Slug:     frontmatter → filename with prefix stripped, slugified
//   - Template: "splash" for home pages if not set
func (inf *Inferrer) Infer(page *engine.Page, filePath string) error {
	filename := filepath.Base(filePath)
	nameNoExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Detect YYYY-MM-DD-slug blog-style filename once; reused for slug and date.
	var filenameDate time.Time
	var filenameSlugRemainder string
	if m := dateFilenameRegex.FindStringSubmatch(nameNoExt); m != nil {
		if t, err := time.Parse("2006-01-02", m[1]); err == nil {
			filenameDate = t
			filenameSlugRemainder = m[2]
		}
	}

	// Slug and weight from filename
	if page.Slug == "" {
		if filenameSlugRemainder != "" {
			if w, clean, found := ExtractNumericPrefix(filenameSlugRemainder); found {
				page.Slug = Slugify(clean)
				if page.Sidebar.Order == 0 {
					page.Sidebar.Order = w
				}
			} else {
				page.Slug = Slugify(filenameSlugRemainder)
			}
		} else if w, clean, found := ExtractNumericPrefix(nameNoExt); found {
			page.Slug = Slugify(clean)
			if page.Sidebar.Order == 0 {
				page.Sidebar.Order = w
			}
		} else if filename == "_index.md" || filename == "index.md" {
			// Slug comes from parent directory
			page.Slug = Slugify(filepath.Base(filepath.Dir(filePath)))
		} else {
			page.Slug = Slugify(nameNoExt)
		}
	}

	// Order from numeric prefix (if not already set by frontmatter or slug inference)
	if page.Sidebar.Order == 0 {
		if w, _, found := ExtractNumericPrefix(nameNoExt); found {
			page.Sidebar.Order = w
		}
	}

	// Title: frontmatter → first H1 → filename
	if page.Title == "" {
		if h1 := ExtractFirstH1(page.RawContent); h1 != "" {
			page.Title = h1
		} else {
			page.Title = FilenameToTitle(filename)
		}
	}

	// Date: filename date prefix takes precedence over mtime when frontmatter
	// did not set it.
	if page.Date.IsZero() && !filenameDate.IsZero() {
		page.Date = filenameDate
	}
	if page.Date.IsZero() {
		if info, err := os.Stat(filePath); err == nil {
			page.Date = info.ModTime()
		}
	}

	// Updated: honor the configured last_updated strategy (git/mtime/false).
	// This resolves the timestamp as data. Whether it is shown to visitors is a
	// separate concern gated by Page.ShowUpdated, so that show_updated: false
	// hides the badge without also stripping sitemap, SEO, and feed metadata.
	if page.Updated.IsZero() {
		if t := GetLastUpdated(filePath, inf.LastUpdatedStrategy, inf.GitIndex); t != nil {
			page.Updated = *t
		}
	}

	// Template: home pages default to "splash"
	if page.Kind == engine.KindHome {
		// Check if page has no explicit template set
		// (Template field is not on Page — it's determined during rendering.
		//  For now we leave this as a no-op; template resolution happens in Phase 5.)
	}

	return nil
}
