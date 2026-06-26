// Sidebar strategy dispatch moved from internal/sidebar.
package navigation

import (
	"github.com/getsarde/sarde/internal/engine"
)

// SidebarResult holds the output of a sidebar strategy build.
type SidebarResult struct {
	Enabled bool
	Tree    *engine.NavTree
	Type    string // "nav", "widget", "none"
}

// SidebarStrategy determines sidebar content for a given page/collection.
type SidebarStrategy interface {
	Build(page *engine.Page, collection *engine.Collection) *SidebarResult
}

// GetStrategy returns the appropriate sidebar strategy for a layout type.
func GetStrategy(layout engine.LayoutType) SidebarStrategy {
	switch layout {
	case engine.LayoutDocs, engine.LayoutWide:
		return &DocsSidebarStrategy{}
	default:
		return &DefaultSidebarStrategy{}
	}
}

// DocsSidebarStrategy returns the collection's pre-built NavTree for docs-layout.
type DocsSidebarStrategy struct{}

func (s *DocsSidebarStrategy) Build(page *engine.Page, collection *engine.Collection) *SidebarResult {
	if collection == nil || collection.NavTree == nil {
		return &SidebarResult{Type: "none"}
	}
	return &SidebarResult{
		Enabled: true,
		Tree:    collection.NavTree,
		Type:    "nav",
	}
}

// DefaultSidebarStrategy returns no sidebar for default-layout collections.
type DefaultSidebarStrategy struct{}

func (s *DefaultSidebarStrategy) Build(page *engine.Page, collection *engine.Collection) *SidebarResult {
	return &SidebarResult{Type: "none"}
}
