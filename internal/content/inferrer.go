package content

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/engine"
)

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
//   - Date:     frontmatter → YYYY-MM-DD filename prefix → file modification time
//   - Updated:  frontmatter → git commit time or file mtime, per LastUpdatedStrategy
//   - Weight:   frontmatter → numeric prefix from filename → 0
//   - Slug:     frontmatter → filename with date and numeric prefixes stripped (FileSlug)
//   - Template: "splash" for home pages if not set
func (inf *Inferrer) Infer(page *engine.Page, filePath string) error {
	filename := filepath.Base(filePath)
	nameNoExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	// A YYYY-MM-DD- prefix (blog-style names) gives the date; FileSlug strips
	// it, and then a numeric prefix, for the slug and the order.
	filenameDate, _, _ := SplitDatePrefix(nameNoExt)
	isIndex := filename == "_index.md" || filename == "index.md"
	fileSlug, fileOrder := FileSlug(nameNoExt)

	// Slug from the file name, or for an index page from its folder.
	if page.Slug == "" {
		if isIndex {
			page.Slug = SlugOrHash(filepath.Base(filepath.Dir(filePath)))
		} else {
			page.Slug = fileSlug
		}
	}

	// Order from a numeric prefix, after any date prefix (if frontmatter did
	// not set it): "2024-03-15-hello" carries no order, "02-setup" carries 2.
	if page.Sidebar.Order == 0 && !isIndex {
		page.Sidebar.Order = fileOrder
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
	// did not set it. A filename date is a deliberate authoring choice, so it
	// also counts as explicit for DateExplicit purposes. buildPage pre-fills
	// page.Date from the file's mtime before calling Infer, except for a
	// date-prefixed name, so this branch sees those.
	if page.Date.IsZero() && !filenameDate.IsZero() {
		page.Date = filenameDate
		page.DateExplicit = true
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
