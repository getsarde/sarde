package collection

import (
	"github.com/frostybee/sarde/internal/consts"
	"github.com/frostybee/sarde/internal/engine"
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

	default:
		return &engine.CollectionConfig{
			SortBy:    "title",
			SortOrder: "asc",
			Layout:    engine.LayoutDefault,
		}
	}
}
