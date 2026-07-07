package content

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/getsarde/sarde/internal/engine"
)

// laneKey identifies a content lane (language + version).
type laneKey struct {
	Lang    string
	Version string // "" for latest/unversioned
}

// Collision records two distinct pages that resolve to the same Permalink.
// The first page registered at a URL is kept; later pages are dropped (first-match
// semantics). These are accumulated rather than logged inline so the builder can
// dedupe and cap them once per build (see emitCollisionWarnings).
type Collision struct {
	Permalink   string
	KeptFile    string // first page registered at this URL
	DroppedFile string // a later page that resolved to the same URL
}

// PageIndex provides O(1) lookups of pages by permalink, slug, heading ID,
// and lane-scoped RelPermalink for internal link resolution.
type PageIndex struct {
	byPermalink map[string]*engine.Page
	bySlug      map[string]*engine.Page
	byLane      map[laneKey]map[string]*engine.Page // relPermalink → *Page per (lang, version)
	headings    map[string][]string
	assets      map[string]bool
	collisions  []Collision // distinct pages sharing one Permalink (first-match kept)

	mu sync.RWMutex // protects headings (written concurrently during parallel render)
}

// BuildPageIndex constructs a PageIndex from all pages.
// The bySlug map uses first-match semantics for duplicate slugs.
// The byLane map indexes each page by its RelPermalink within its (lang, version) lane.
func BuildPageIndex(pages []*engine.Page) *PageIndex {
	idx := &PageIndex{
		byPermalink: make(map[string]*engine.Page, len(pages)),
		bySlug:      make(map[string]*engine.Page, len(pages)),
		byLane:      make(map[laneKey]map[string]*engine.Page),
		headings:    make(map[string][]string),
		assets:      make(map[string]bool),
	}
	for _, p := range pages {
		// First-match semantics (matching bySlug below). A collision means two
		// distinct pages claim the same URL/lane key — keep the first, rather than
		// silently last-write-wins (which, combined with map-ordered page
		// generation, made link checking nondeterministic).
		//
		// Only byPermalink collisions are recorded for reporting: a byLane
		// collision (same RelPermalink within one lane) implies the same Permalink,
		// so it is already captured here. byLane keeps first-match silently.
		if p.Permalink != "" {
			if existing, ok := idx.byPermalink[p.Permalink]; ok {
				if existing != p {
					idx.collisions = append(idx.collisions, Collision{
						Permalink:   p.Permalink,
						KeptFile:    existing.FilePath,
						DroppedFile: p.FilePath,
					})
				}
			} else {
				idx.byPermalink[p.Permalink] = p
			}
		}
		if p.Slug != "" {
			if _, exists := idx.bySlug[p.Slug]; !exists {
				idx.bySlug[p.Slug] = p
			}
		}
		// Lane index: register by RelPermalink within (lang, version). First-match,
		// silent — any collision here is also a byPermalink collision (above).
		if p.RelPermalink != "" {
			key := laneKey{Lang: p.Lang, Version: p.Version}
			lane, ok := idx.byLane[key]
			if !ok {
				lane = make(map[string]*engine.Page)
				idx.byLane[key] = lane
			}
			if _, ok := lane[p.RelPermalink]; !ok {
				lane[p.RelPermalink] = p
			}
		}
	}
	return idx
}

// Collisions returns the distinct-page permalink collisions recorded during
// BuildPageIndex (first-match kept). Empty when no two pages share a URL.
func (idx *PageIndex) Collisions() []Collision {
	return idx.collisions
}

// HasPage reports whether a page with the given permalink exists.
func (idx *PageIndex) HasPage(permalink string) bool {
	_, ok := idx.byPermalink[permalink]
	return ok
}

// LookupByPermalink returns the page with the given permalink, or nil.
func (idx *PageIndex) LookupByPermalink(permalink string) *engine.Page {
	return idx.byPermalink[permalink]
}

// LookupBySlug returns the first page matching the given slug, or nil.
func (idx *PageIndex) LookupBySlug(slug string) *engine.Page {
	return idx.bySlug[slug]
}

// LookupInLane returns the page with the given RelPermalink in the specified
// (lang, version) lane. Returns nil if not found.
func (idx *PageIndex) LookupInLane(relPermalink, lang, version string) *engine.Page {
	lane, ok := idx.byLane[laneKey{Lang: lang, Version: version}]
	if !ok {
		return nil
	}
	return lane[relPermalink]
}

// SetHeadings stores heading IDs for a page. "_top" is always prepended.
// Safe for concurrent use.
func (idx *PageIndex) SetHeadings(permalink string, headingIDs []string) {
	ids := make([]string, 0, len(headingIDs)+1)
	ids = append(ids, "_top")
	ids = append(ids, headingIDs...)

	idx.mu.Lock()
	idx.headings[permalink] = ids
	idx.mu.Unlock()
}

// HasHeading reports whether the given heading ID exists on the page.
// Safe for concurrent use.
func (idx *PageIndex) HasHeading(permalink, headingID string) bool {
	idx.mu.RLock()
	ids, ok := idx.headings[permalink]
	idx.mu.RUnlock()
	if !ok {
		return false
	}
	for _, id := range ids {
		if id == headingID {
			return true
		}
	}
	return false
}

// AddAssets walks a static directory and indexes all files as root-relative paths.
func (idx *PageIndex) AddAssets(staticDir string) {
	filepath.Walk(staticDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(staticDir, path)
		if err != nil {
			return nil
		}
		idx.assets["/"+filepath.ToSlash(rel)] = true
		return nil
	})
}

// CopyAssetsFrom copies another index's asset set. Used by the incremental
// rebuild's body-only fast path, which skips the static/ directory walk:
// static file changes never route through ContentRebuild (the dev-server
// watcher classifies them as static changes, which take the full-build path),
// so the previous build's asset set is still valid.
func (idx *PageIndex) CopyAssetsFrom(prev *PageIndex) {
	if prev == nil {
		return
	}
	for path := range prev.assets {
		idx.assets[path] = true
	}
}

// HasAsset reports whether a static asset with the given root-relative path exists.
func (idx *PageIndex) HasAsset(path string) bool {
	return idx.assets[path]
}

// PageCount returns the number of indexed pages.
func (idx *PageIndex) PageCount() int {
	return len(idx.byPermalink)
}

// Permalinks returns all indexed permalinks. Used for testing and debugging.
func (idx *PageIndex) Permalinks() []string {
	out := make([]string, 0, len(idx.byPermalink))
	for k := range idx.byPermalink {
		out = append(out, k)
	}
	return out
}

// HeadingsFor returns the heading IDs for a page, or nil if not set.
func (idx *PageIndex) HeadingsFor(permalink string) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.headings[permalink]
}

// NormalizePermalink ensures a permalink has a trailing slash (unless it's a file path with extension).
func NormalizePermalink(permalink string) string {
	if permalink == "" || permalink == "/" {
		return permalink
	}
	// File-like only when the last segment has a real extension. A digit-only
	// suffix (/docs/v1.2, /release-2.0) is a versioned page path, not a file.
	if ext := path.Ext(path.Base(permalink)); ext != "" && !isAllDigits(ext[1:]) {
		return permalink
	}
	if !strings.HasSuffix(permalink, "/") {
		return permalink + "/"
	}
	return permalink
}

// isAllDigits reports whether s is non-empty and consists only of ASCII digits.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
