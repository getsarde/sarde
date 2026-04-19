package collection

import (
	"strings"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

// BuildSectionTree constructs a tree of Section nodes from pages.
// Pages with Kind==KindSection become section index pages.
// Other pages are assigned to their deepest matching section.
// Returns top-level sections for the collection.
func BuildSectionTree(pages []*engine.Page, collectionName string) []*engine.Section {
	// Map of directory path (relative to collection) → section
	sectionMap := make(map[string]*engine.Section)

	// First pass: create sections from _index.md pages
	for _, page := range pages {
		if page.Kind != engine.KindSection {
			continue
		}
		dirPath := sectionDir(page.RelPermalink, collectionName)
		sec := &engine.Section{
			Title:     page.Title,
			Slug:      page.Slug,
			Permalink: page.RelPermalink,
			IndexPage: page,
			Render:    true,
		}
		if page.Params != nil {
			if t, ok := page.Params["transparent"].(bool); ok && t {
				sec.Transparent = true
			}
			if r, ok := page.Params["render"].(bool); ok && !r {
				sec.Render = false
			}
		}
		sectionMap[dirPath] = sec
	}

	// Wire parent-child relationships
	for path, sec := range sectionMap {
		if path == "" {
			continue // root section has no parent
		}
		parentPath := parentDir(path)
		if parent, ok := sectionMap[parentPath]; ok {
			sec.Parent = parent
			parent.Sections = append(parent.Sections, sec)
		}
	}

	// Second pass: assign non-section pages to their deepest matching section
	for _, page := range pages {
		if page.Kind == engine.KindSection {
			continue
		}
		// Strip collection prefix to get section-relative path
		dirPath := sectionDir(pageDir(page.RelPermalink)+"/", collectionName)
		sec := findDeepestSection(dirPath, sectionMap)
		if sec != nil {
			page.Section = sec
			sec.Pages = append(sec.Pages, page)
		}
	}

	// Handle transparent sections: hoist pages into parent
	for _, sec := range sectionMap {
		if sec.Transparent && sec.Parent != nil {
			sec.Parent.Pages = append(sec.Parent.Pages, sec.Pages...)
			sec.Pages = nil
		}
	}

	// Collect top-level sections (those with no parent)
	var roots []*engine.Section
	for _, sec := range sectionMap {
		if sec.Parent == nil {
			roots = append(roots, sec)
		}
	}

	return roots
}

// sectionDir extracts the directory path relative to collection from a permalink.
// "/docs/guides/" → "guides", "/docs/" → ""
func sectionDir(permalink, collectionName string) string {
	// Strip leading "/" and trailing "/"
	p := strings.Trim(permalink, "/")
	// Strip collection prefix
	prefix := collectionName
	if strings.HasPrefix(p, prefix) {
		p = strings.TrimPrefix(p, prefix)
		p = strings.TrimPrefix(p, "/")
	}
	return p
}

// pageDir extracts the parent directory from a permalink.
// "/docs/guides/auth/" → "docs/guides"
func pageDir(permalink string) string {
	p := strings.Trim(permalink, "/")
	idx := strings.LastIndex(p, "/")
	if idx < 0 {
		return ""
	}
	return p[:idx]
}

// parentDir returns the parent of a section directory path.
// "guides/advanced" → "guides", "guides" → "", "" → ""
func parentDir(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return ""
	}
	return path[:idx]
}

// findDeepestSection walks up from dirPath to find the deepest matching section.
func findDeepestSection(dirPath string, sectionMap map[string]*engine.Section) *engine.Section {
	path := dirPath
	for {
		if sec, ok := sectionMap[path]; ok {
			return sec
		}
		if path == "" {
			break
		}
		idx := strings.LastIndex(path, "/")
		if idx < 0 {
			path = ""
		} else {
			path = path[:idx]
		}
	}
	return nil
}
