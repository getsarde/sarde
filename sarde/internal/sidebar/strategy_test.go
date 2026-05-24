package sidebar

import (
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

func TestGetStrategy_Docs(t *testing.T) {
	s := GetStrategy(engine.LayoutDocs)
	if _, ok := s.(*DocsSidebarStrategy); !ok {
		t.Error("expected DocsSidebarStrategy for docs layout")
	}
}

func TestGetStrategy_Default(t *testing.T) {
	s := GetStrategy(engine.LayoutDefault)
	if _, ok := s.(*DefaultSidebarStrategy); !ok {
		t.Error("expected DefaultSidebarStrategy for default layout")
	}
}

func TestDocsSidebarStrategy_WithNavTree(t *testing.T) {
	tree := &engine.NavTree{Root: &engine.NavNode{Label: "Root"}}
	col := &engine.Collection{NavTree: tree}

	result := (&DocsSidebarStrategy{}).Build(nil, col)

	if !result.Enabled {
		t.Error("expected enabled")
	}
	if result.Type != "nav" {
		t.Errorf("type: got %q", result.Type)
	}
	if result.Tree != tree {
		t.Error("expected same tree")
	}
}

func TestDocsSidebarStrategy_NilCollection(t *testing.T) {
	result := (&DocsSidebarStrategy{}).Build(nil, nil)
	if result.Type != "none" {
		t.Errorf("expected none, got %q", result.Type)
	}
}

func TestDefaultSidebarStrategy(t *testing.T) {
	result := (&DefaultSidebarStrategy{}).Build(nil, nil)
	if result.Type != "none" {
		t.Errorf("expected none, got %q", result.Type)
	}
}
