package build

import (
	"cmp"
	"slices"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
)

func populatePageIndexHeadings(idx *content.PageIndex, pages []*engine.Page) {
	for _, page := range pages {
		setPageIndexHeadings(idx, page)
	}
}

func setPageIndexHeadings(idx *content.PageIndex, page *engine.Page) {
	if idx == nil || page == nil || len(page.Headings) == 0 {
		return
	}
	if existing := idx.HeadingsFor(page.Permalink); existing != nil {
		return
	}
	ids := make([]string, len(page.Headings))
	for i, h := range page.Headings {
		ids[i] = h.ID
	}
	idx.SetHeadings(page.Permalink, ids)
}

func updateValidationEntry(data map[string]engine.ValidationEntry, page *engine.Page, links []engine.CollectedLink) {
	if len(links) == 0 {
		delete(data, page.Permalink)
		return
	}
	data[page.Permalink] = engine.ValidationEntry{Links: links, FilePath: page.FilePath, Lang: page.Lang}
}

func countMarkdownPages(pages []*engine.Page) int {
	count := 0
	for _, page := range pages {
		if page.RawContent != "" {
			count++
		}
	}
	return count
}

func sortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
