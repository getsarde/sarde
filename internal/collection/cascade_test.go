package collection

import (
	"testing"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
)

func TestApplyCascade_Basic(t *testing.T) {
	indexPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{consts.CascadeKey: map[string]any{"layout": "docs", "toc": map[string]any{"enabled": true}}},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: sec},
		Params:            map[string]any{},
	}

	ApplyCascade([]*engine.Page{child})

	if child.Params["layout"] != "docs" {
		t.Errorf("expected layout=docs, got %v", child.Params["layout"])
	}
	if child.TOC.Enabled == nil || *child.TOC.Enabled != true {
		t.Errorf("expected TOC.Enabled=true, got %v", child.TOC.Enabled)
	}
}

func TestApplyCascade_ChildOverrides(t *testing.T) {
	indexPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{consts.CascadeKey: map[string]any{"layout": "docs"}},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: sec},
		Params:            map[string]any{"layout": "wide"},
	}

	ApplyCascade([]*engine.Page{child})

	if child.Params["layout"] != "wide" {
		t.Errorf("child's own layout should win, got %v", child.Params["layout"])
	}
}

func TestApplyCascade_MultiLevel(t *testing.T) {
	grandparentIdx := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{consts.CascadeKey: map[string]any{"layout": "docs", "toc": map[string]any{"enabled": false}}},
	}
	parentIdx := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{consts.CascadeKey: map[string]any{"toc": map[string]any{"enabled": true}}},
	}
	grandparent := &engine.Section{IndexPage: grandparentIdx}
	parent := &engine.Section{IndexPage: parentIdx, Parent: grandparent}
	child := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: parent},
		Params:            map[string]any{},
	}

	ApplyCascade([]*engine.Page{child})

	if child.Params["layout"] != "docs" {
		t.Errorf("expected layout=docs from grandparent, got %v", child.Params["layout"])
	}
	if child.TOC.Enabled == nil || *child.TOC.Enabled != true {
		t.Errorf("expected TOC.Enabled=true from parent (overrides grandparent), got %v", child.TOC.Enabled)
	}
}

func TestApplyCascade_ParamsMerge(t *testing.T) {
	indexPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params: map[string]any{
			consts.CascadeKey: map[string]any{
				"params": map[string]any{"author": "Team", "color": "blue"},
			},
		},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: sec},
		Params:            map[string]any{"color": "red"},
	}

	ApplyCascade([]*engine.Page{child})

	if child.Params["author"] != "Team" {
		t.Errorf("expected author=Team from cascade params, got %v", child.Params["author"])
	}
	if child.Params["color"] != "red" {
		t.Errorf("child's own color should win, got %v", child.Params["color"])
	}
}

func TestApplyCascade_SectionPageFromParent(t *testing.T) {
	parentIdx := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{consts.CascadeKey: map[string]any{"toc": map[string]any{"enabled": true}}},
	}
	parent := &engine.Section{IndexPage: parentIdx}
	childIdx := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{},
	}
	childSec := &engine.Section{IndexPage: childIdx, Parent: parent}
	childIdx.PageRelationships.Section = childSec

	ApplyCascade([]*engine.Page{childIdx})

	if childIdx.TOC.Enabled == nil || *childIdx.TOC.Enabled != true {
		t.Errorf("section _index.md should receive TOC.Enabled=true from parent, got %v", childIdx.TOC.Enabled)
	}
}

func TestApplyCascade_NoCascade(t *testing.T) {
	indexPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: sec},
		Params:            map[string]any{},
	}

	ApplyCascade([]*engine.Page{child})

	if len(child.Params) != 0 {
		t.Errorf("expected no params change, got %v", child.Params)
	}
}

func TestApplyCascade_SidebarLabel(t *testing.T) {
	indexPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params:       map[string]any{consts.CascadeKey: map[string]any{"sidebar": map[string]any{"label": "Guides"}}},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: sec},
		Params:            map[string]any{},
	}

	ApplyCascade([]*engine.Page{child})

	if child.Sidebar.Label != "Guides" {
		t.Errorf("expected Sidebar.Label=Guides, got %q", child.Sidebar.Label)
	}
}

func TestApplyCascade_Banner(t *testing.T) {
	indexPage := &engine.Page{
		PageIdentity: engine.PageIdentity{Kind: engine.KindSection},
		Params: map[string]any{
			consts.CascadeKey: map[string]any{
				"banner": map[string]any{"content": "Under review", "variant": "caution"},
			},
		},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		PageIdentity:      engine.PageIdentity{Kind: engine.KindPage},
		PageRelationships: engine.PageRelationships{Section: sec},
		Params:            map[string]any{},
	}

	ApplyCascade([]*engine.Page{child})

	b, ok := child.Params["banner"].(*engine.PageBanner)
	if !ok || b == nil {
		t.Fatalf("expected *PageBanner, got %T", child.Params["banner"])
	}
	if b.Content != "Under review" {
		t.Errorf("expected content='Under review', got %q", b.Content)
	}
	if b.Variant != "caution" {
		t.Errorf("expected variant=caution, got %q", b.Variant)
	}
}
