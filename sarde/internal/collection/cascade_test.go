package collection

import (
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

func TestApplyCascade_Basic(t *testing.T) {
	indexPage := &engine.Page{
		Kind:   engine.KindSection,
		Params: map[string]any{"__cascade": map[string]any{"layout": "docs", "toc": true}},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		Kind:    engine.KindPage,
		Section: sec,
		Params:  map[string]any{},
	}

	ApplyCascade([]*engine.Page{child})

	if child.Params["layout"] != "docs" {
		t.Errorf("expected layout=docs, got %v", child.Params["layout"])
	}
	if child.Params["toc"] != true {
		t.Errorf("expected toc=true, got %v", child.Params["toc"])
	}
}

func TestApplyCascade_ChildOverrides(t *testing.T) {
	indexPage := &engine.Page{
		Kind:   engine.KindSection,
		Params: map[string]any{"__cascade": map[string]any{"layout": "docs"}},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		Kind:    engine.KindPage,
		Section: sec,
		Params:  map[string]any{"layout": "wide"},
	}

	ApplyCascade([]*engine.Page{child})

	if child.Params["layout"] != "wide" {
		t.Errorf("child's own layout should win, got %v", child.Params["layout"])
	}
}

func TestApplyCascade_MultiLevel(t *testing.T) {
	grandparentIdx := &engine.Page{
		Kind:   engine.KindSection,
		Params: map[string]any{"__cascade": map[string]any{"layout": "docs", "toc": false}},
	}
	parentIdx := &engine.Page{
		Kind:   engine.KindSection,
		Params: map[string]any{"__cascade": map[string]any{"toc": true}},
	}
	grandparent := &engine.Section{IndexPage: grandparentIdx}
	parent := &engine.Section{IndexPage: parentIdx, Parent: grandparent}
	child := &engine.Page{
		Kind:    engine.KindPage,
		Section: parent,
		Params:  map[string]any{},
	}

	ApplyCascade([]*engine.Page{child})

	if child.Params["layout"] != "docs" {
		t.Errorf("expected layout=docs from grandparent, got %v", child.Params["layout"])
	}
	if child.Params["toc"] != true {
		t.Errorf("expected toc=true from parent (overrides grandparent), got %v", child.Params["toc"])
	}
}

func TestApplyCascade_ParamsMerge(t *testing.T) {
	indexPage := &engine.Page{
		Kind: engine.KindSection,
		Params: map[string]any{
			"__cascade": map[string]any{
				"params": map[string]any{"author": "Team", "color": "blue"},
			},
		},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		Kind:    engine.KindPage,
		Section: sec,
		Params:  map[string]any{"color": "red"},
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
		Kind:   engine.KindSection,
		Params: map[string]any{"__cascade": map[string]any{"toc": true}},
	}
	parent := &engine.Section{IndexPage: parentIdx}
	childIdx := &engine.Page{
		Kind:    engine.KindSection,
		Params:  map[string]any{},
	}
	childSec := &engine.Section{IndexPage: childIdx, Parent: parent}
	childIdx.Section = childSec

	ApplyCascade([]*engine.Page{childIdx})

	if childIdx.Params["toc"] != true {
		t.Errorf("section _index.md should receive cascade from parent, got %v", childIdx.Params["toc"])
	}
}

func TestApplyCascade_NoCascade(t *testing.T) {
	indexPage := &engine.Page{
		Kind:   engine.KindSection,
		Params: map[string]any{},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		Kind:    engine.KindPage,
		Section: sec,
		Params:  map[string]any{},
	}

	ApplyCascade([]*engine.Page{child})

	if len(child.Params) != 0 {
		t.Errorf("expected no params change, got %v", child.Params)
	}
}

func TestApplyCascade_SidebarLabel(t *testing.T) {
	indexPage := &engine.Page{
		Kind:   engine.KindSection,
		Params: map[string]any{"__cascade": map[string]any{"sidebar_label": "Guides"}},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		Kind:    engine.KindPage,
		Section: sec,
		Params:  map[string]any{},
	}

	ApplyCascade([]*engine.Page{child})

	if child.SidebarLabel != "Guides" {
		t.Errorf("expected SidebarLabel=Guides, got %q", child.SidebarLabel)
	}
}

func TestApplyCascade_Banner(t *testing.T) {
	indexPage := &engine.Page{
		Kind: engine.KindSection,
		Params: map[string]any{
			"__cascade": map[string]any{
				"banner": map[string]any{"content": "Under review", "variant": "caution"},
			},
		},
	}
	sec := &engine.Section{IndexPage: indexPage}
	child := &engine.Page{
		Kind:    engine.KindPage,
		Section: sec,
		Params:  map[string]any{},
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
