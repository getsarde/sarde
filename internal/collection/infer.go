package collection

import (
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
)

var blogNames = map[string]bool{
	consts.CollectionBlog: true, "posts": true, "articles": true, "news": true,
}

// IsBlogName returns true if the directory name maps to a blog-type collection.
func IsBlogName(name string) bool {
	return blogNames[name]
}

var docsNames = map[string]bool{
	consts.CollectionDocs: true, "documentation": true, "guides": true, "reference": true,
	consts.CollectionCourses: true, "tutorials": true, "lessons": true, "workshops": true,
}

var labsNames = map[string]bool{
	consts.CollectionLabs: true,
}

var slidesNames = map[string]bool{
	consts.CollectionSlides: true, "presentations": true, "decks": true,
}

// IsLabsName returns true if the directory name maps to a labs-type collection.
func IsLabsName(name string) bool {
	return labsNames[name]
}

// IsSlidesName returns true if the directory name maps to a slides-type collection.
func IsSlidesName(name string) bool {
	return slidesNames[name]
}

// applyLabsLayoutDefault gives lab-level and deeper pages the labs layout.
// Landing pages (collection root, course level) keep LayoutDefault so they
// render as card grids. The depth threshold depends on whether the collection
// uses 2-level (labs at depth 1) or 3-level (labs at depth 2) structure.
func applyLabsLayoutDefault(pages []*engine.Page) {
	isMultiCourse := detectMultiCourseFromPages(pages)
	labDepth := 1
	if isMultiCourse {
		labDepth = 2
	}
	for _, p := range pages {
		if p.Section == nil {
			continue
		}
		depth := engine.SectionDepth(p.Section)
		if p.Kind == engine.KindSection && depth < labDepth {
			continue
		}
		if p.Kind != engine.KindSection && depth < labDepth {
			continue
		}
		if p.Params == nil {
			p.Params = make(map[string]any)
		}
		if _, exists := p.Params["layout"]; !exists {
			p.Params["layout"] = string(engine.LayoutLabs)
		}
	}
}

// detectMultiCourseFromPages checks if any section page at depth 1 has
// children that are also sections (depth 2), indicating 3-level structure.
func detectMultiCourseFromPages(pages []*engine.Page) bool {
	for _, p := range pages {
		if p.Kind == engine.KindSection && p.Section != nil && engine.SectionDepth(p.Section) >= 2 {
			return true
		}
	}
	return false
}

// applySlidesLayoutDefault gives every regular (non-section) page in a
// slides collection the presentation layout unless frontmatter or cascade
// already set one. Gallery/section pages keep the collection layout.
func applySlidesLayoutDefault(pages []*engine.Page) {
	for _, p := range pages {
		if p.Kind == engine.KindSection {
			continue
		}
		if p.Params == nil {
			p.Params = make(map[string]any)
		}
		if _, exists := p.Params["layout"]; !exists {
			p.Params["layout"] = string(engine.LayoutPresentation)
		}
	}
}

// InferCollection returns a CollectionConfig with sensible defaults
// based on the directory name convention.
func InferCollection(dirName string) *engine.CollectionConfig {
	switch {
	case blogNames[dirName]:
		return &engine.CollectionConfig{
			SortBy:    "date",
			SortOrder: "desc",
			Layout:    engine.LayoutDefault,
			Feed:      true,
			Paginate:  10,
			PrevNext: &engine.PrevNextConfig{
				Enabled: true,
				Labels:  [2]string{"Newer", "Older"},
			},
		}

	case docsNames[dirName]:
		return &engine.CollectionConfig{
			SortBy:    "order",
			SortOrder: "asc",
			Layout:    engine.LayoutDocs,
			Sidebar: &engine.SidebarConfig{
				Collapsible:        true,
				CollapsedByDefault: false,
				MaxDepth:           4,
				Search:             true,
			},
			TOC: &engine.TOCConfig{
				Enabled:         true,
				MinLevel:        2,
				MaxLevel:        4,
				ScrollHighlight: true,
			},
			PrevNext: &engine.PrevNextConfig{
				Enabled: true,
				Labels:  [2]string{"Previous", "Next"},
			},
		}

	case labsNames[dirName]:
		return &engine.CollectionConfig{
			SortBy:    "order",
			SortOrder: "asc",
			Layout:    engine.LayoutDefault,
			Sidebar: &engine.SidebarConfig{
				Collapsible:        true,
				CollapsedByDefault: false,
				MaxDepth:           4,
				Search:             true,
			},
			TOC: &engine.TOCConfig{
				Enabled:         true,
				MinLevel:        2,
				MaxLevel:        4,
				ScrollHighlight: true,
			},
			PrevNext: &engine.PrevNextConfig{
				Enabled: true,
				Labels:  [2]string{"Previous", "Next"},
			},
			Labs: &engine.LabsConfig{
				StepLabel: "Lab",
			},
		}

	case slidesNames[dirName]:
		// Gallery list pages keep normal site chrome (LayoutDefault);
		// individual deck pages default to layout: presentation via
		// applySlidesLayoutDefault after cascade resolution.
		return &engine.CollectionConfig{
			SortBy:    "date",
			SortOrder: "desc",
			Layout:    engine.LayoutDefault,
			PrevNext: &engine.PrevNextConfig{
				Enabled: false,
			},
		}

	default:
		return &engine.CollectionConfig{
			SortBy:    "title",
			SortOrder: "asc",
			Layout:    engine.LayoutDefault,
		}
	}
}
