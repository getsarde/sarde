package telescope

import (
	"sort"

	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/plugin"
)

// indexEntry is one page in the telescope-pages.json search index. Path is the
// fully resolved root-relative URL (basePath, lang, and version applied), so
// the client can navigate to it directly.
type indexEntry struct {
	Title       string   `json:"title"`
	Path        string   `json:"path"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Lang        string   `json:"lang,omitempty"`
	Version     string   `json:"version,omitempty"`
	Collection  string   `json:"collection,omitempty"`
}

// buildIndex converts the site's pages into search index entries. Drafts,
// structural nodes (sections, taxonomies, terms), excluded paths, and
// duplicate URLs are skipped. Exclude patterns match the lane-free
// RelPermalink, the same semantics as the sitemap plugin. Deduplication uses
// the resolved URL because translations of a page share one lang-free
// RelPermalink. Entries are sorted by path so the emitted JSON is
// deterministic across builds.
func buildIndex(pages []*engine.Page, exclude []string, resolve func(relPath, lang, version string) string) []indexEntry {
	if resolve == nil {
		resolve = func(relPath, _, _ string) string { return relPath }
	}
	var out []indexEntry
	seen := make(map[string]bool, len(pages))
	for _, p := range pages {
		if p == nil || p.Draft {
			continue
		}
		switch p.Kind {
		case engine.KindPage, engine.KindBundle, engine.KindHome, engine.KindStandalone:
			// Real content pages.
		default:
			continue
		}
		if p.RelPermalink == "" {
			continue
		}
		if plugin.ShouldExcludePath(p.RelPermalink, exclude) {
			continue
		}
		url := resolve(p.RelPermalink, p.Lang, p.Version)
		if seen[url] {
			continue
		}
		seen[url] = true
		collection := ""
		if p.Collection != nil {
			collection = p.Collection.Title
			if collection == "" {
				collection = p.Collection.Name
			}
		}
		out = append(out, indexEntry{
			Title:       p.Title,
			Path:        url,
			Description: p.Description,
			Tags:        p.Tags,
			Lang:        p.Lang,
			Version:     p.Version,
			Collection:  collection,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
