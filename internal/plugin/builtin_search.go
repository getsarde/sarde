package plugin

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"sync"

	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
)

//go:embed all:search_assets
var searchAssetsFS embed.FS

var searchRuntimeScripts = []string{
	"/assets/js/static-search.js",
}

func newSearchPlugin(cfg map[string]any) *Plugin {
	cache := &searchDocCache{}
	return &Plugin{
		Name: "search",
		Hooks: PluginHooks{
			BeforeRender: func(ctx *BeforeRenderContext) error {
				for _, s := range searchRuntimeScripts {
					ctx.RouteData.Scripts = appendUniqueScript(ctx.RouteData.Scripts, s)
				}
				return nil
			},
			BuildDone: func(ctx *BuildDoneContext) error {
				return searchBuildDone(ctx, cfg, cache)
			},
		},
	}
}

// searchCacheEntry pairs a page's extracted search documents with the
// content digest they were derived from.
type searchCacheEntry struct {
	digest string
	docs   []searchDocument
}

// searchDocCache caches per-page search documents across rebuilds, keyed by
// page URL and validated against page.ContentDigest, so incremental rebuilds
// only re-extract text for pages that actually changed. Entries are only
// reused on incremental rebuilds: a full build can change breadcrumb inputs
// (collection and section titles from _index.md) without touching a page's
// own digest, so full builds recompute everything and repopulate the cache.
// If two distinct pages share a URL (permalink collision), only the
// first-seen page's entry is kept, consistent with the dedup below.
type searchDocCache struct {
	mu      sync.Mutex
	entries map[string]searchCacheEntry
}

type searchDocument struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Description string   `json:"description,omitempty"`
	Content     string   `json:"content,omitempty"`
	Section     string   `json:"section,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Version     string   `json:"version,omitempty"`
	Breadcrumb  string   `json:"breadcrumb,omitempty"`
	Kind        string   `json:"kind,omitempty"`
	Anchor      string   `json:"anchor,omitempty"`
	Lang        string   `json:"-"`
}

func searchBuildDone(ctx *BuildDoneContext, cfg map[string]any, cache *searchDocCache) error {
	maxLen := cfgutil.Int(cfg, "max_content_length", 5000)
	excludePatterns := cfgutil.StringSlice(cfg, "exclude")

	cache.mu.Lock()
	prev := cache.entries
	cache.mu.Unlock()
	next := make(map[string]searchCacheEntry, len(ctx.Pages))

	var docs []searchDocument
	seen := make(map[string]bool)
	for _, page := range ctx.Pages {
		if page.Draft {
			continue
		}
		// Frontmatter opt-out: pagefind: false (cascadable via section
		// _index.md). The typed field lives on FrontmatterMeta; Params is
		// where the page builder and cascade surface it.
		if v, ok := page.Params["pagefind"].(bool); ok && !v {
			continue
		}
		if shouldExclude(page.Permalink, excludePatterns) {
			continue
		}
		url := page.URL()
		if seen[url] {
			continue
		}
		seen[url] = true

		var pageDocs []searchDocument
		if ctx.Incremental && page.ContentDigest != "" {
			if e, ok := prev[url]; ok && e.digest == page.ContentDigest {
				pageDocs = e.docs
			}
		}
		if pageDocs == nil {
			pageDocs = buildSearchDocs(page, url, maxLen)
		}
		if page.ContentDigest != "" {
			next[url] = searchCacheEntry{digest: page.ContentDigest, docs: pageDocs}
		}

		docs = append(docs, pageDocs[0])
		for _, d := range pageDocs[1:] {
			if seen[d.ID] {
				continue
			}
			seen[d.ID] = true
			docs = append(docs, d)
		}
	}

	// Swapping in a freshly built map evicts entries for pages that no
	// longer exist (deletions always route through a full build today).
	cache.mu.Lock()
	cache.entries = next
	cache.mu.Unlock()

	byLang := make(map[string][]searchDocument)
	for _, d := range docs {
		lang := d.Lang
		if lang == "" {
			lang = "en"
		}
		byLang[lang] = append(byLang[lang], d)
	}

	total := 0
	for lang, langDocs := range byLang {
		data, err := json.Marshal(langDocs)
		if err != nil {
			return err
		}
		if err := ctx.WriteFile(fmt.Sprintf("search-index.%s.json", lang), data); err != nil {
			return err
		}
		total += len(langDocs)
	}

	// The embedded runtime assets never change between rebuilds of one
	// builder; the first build of a session is always a full Build() and
	// incremental rebuilds never prune output, so skipping the copy on
	// incremental rebuilds is safe.
	if !ctx.Incremental {
		if err := writeSearchAssets(ctx, byLang); err != nil {
			return err
		}
	}
	ctx.Log(fmt.Sprintf("Built search index (%d documents, %d languages)", total, len(byLang)))
	return nil
}

// buildSearchDocs extracts the searchable documents for one page: the page
// document itself followed by one document per heading. Heading documents
// are deduplicated within the page; cross-page dedup happens at the call
// site against the global seen set.
func buildSearchDocs(page *engine.Page, url string, maxLen int) []searchDocument {
	raw := TruncateRuneSafe(string(page.Content), maxLen*3)
	content := TruncateRuneSafe(ExtractSearchText(raw), maxLen)

	section := ""
	if page.Collection != nil {
		section = page.Collection.Name
	}

	breadcrumb := buildBreadcrumb(page)

	docs := []searchDocument{{
		ID:          url,
		Title:       page.Title,
		URL:         url,
		Description: page.Description,
		Content:     content,
		Section:     section,
		Tags:        page.Tags,
		Version:     page.Version,
		Breadcrumb:  breadcrumb,
		Kind:        "page",
		Lang:        page.Lang,
	}}

	seenHeadings := make(map[string]bool, len(page.Headings)+1)
	seenHeadings[url] = true
	for _, h := range page.Headings {
		hURL := url + "#" + h.ID
		if seenHeadings[hURL] {
			continue
		}
		seenHeadings[hURL] = true
		docs = append(docs, searchDocument{
			ID:         hURL,
			Title:      h.Text,
			URL:        hURL,
			Section:    section,
			Breadcrumb: breadcrumb + " > " + page.Title,
			Kind:       "heading",
			Anchor:     h.ID,
			Version:    page.Version,
			Lang:       page.Lang,
		})
	}
	return docs
}

func buildBreadcrumb(page *engine.Page) string {
	var parts []string
	if page.Collection != nil {
		parts = append(parts, page.Collection.Title)
	}
	var sectionParts []string
	// Depth cap guards against a (mis-assembled) circular section tree.
	for s, depth := page.Section, 0; s != nil && depth < 100; s, depth = s.Parent, depth+1 {
		title := s.Title
		if title == "" && s.IndexPage != nil {
			title = s.IndexPage.Title
		}
		if title != "" {
			sectionParts = append(sectionParts, title)
		}
	}
	for i := len(sectionParts) - 1; i >= 0; i-- {
		parts = append(parts, sectionParts[i])
	}
	return strings.Join(parts, " > ")
}

func writeSearchAssets(ctx *BuildDoneContext, byLang map[string][]searchDocument) error {
	err := fs.WalkDir(searchAssetsFS, "search_assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Stemmer modules are copied selectively below, per index lane.
			if path == "search_assets/stemmers" {
				return fs.SkipDir
			}
			return nil
		}
		data, err := fs.ReadFile(searchAssetsFS, path)
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(path, "search_assets/")
		var dest string
		switch rel {
		case "orama.esm.js":
			dest = "assets/vendor/orama/orama.esm.js"
		case "static-search.js":
			dest = "assets/js/static-search.js"
		default:
			dest = "assets/js/" + rel
		}
		return ctx.WriteFile(dest, data)
	})
	if err != nil {
		return err
	}

	// Copy only the stemmer modules matching the emitted index lanes. A
	// language without a vendored module simply gets none; the client falls
	// back to default tokenization.
	copied := make(map[string]bool, len(byLang))
	for lang := range byLang {
		base := searchLangBase(lang)
		if base == "" || copied[base] {
			continue
		}
		data, err := fs.ReadFile(searchAssetsFS, "search_assets/stemmers/"+base+".js")
		if err != nil {
			continue
		}
		copied[base] = true
		if err := ctx.WriteFile("assets/vendor/orama/stemmers/"+base+".js", data); err != nil {
			return err
		}
	}
	return nil
}

// searchLangBase normalizes a site language code to its base subtag
// ("pt-BR" and "pt_br" both map to "pt"), matching the client's stemmer
// module lookup.
func searchLangBase(lang string) string {
	lang = strings.ToLower(lang)
	if i := strings.IndexAny(lang, "-_"); i >= 0 {
		lang = lang[:i]
	}
	return lang
}

func appendUniqueScript(list []string, item string) []string {
	for _, existing := range list {
		if existing == item {
			return list
		}
	}
	return append(list, item)
}
