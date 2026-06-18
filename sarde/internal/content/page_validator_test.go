package content

import (
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

func TestValidatePageFields_MissingTitle(t *testing.T) {
	page := &engine.Page{}
	fm := &engine.Frontmatter{}
	warnings := ValidatePageFields(page, fm)
	found := false
	for _, w := range warnings {
		if w.Field == "title" && w.Level == "error" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error-level warning for missing title")
	}
}

func TestValidatePageFields_TitlePresent(t *testing.T) {
	page := &engine.Page{}
	fm := &engine.Frontmatter{}
	fm.Title = "Hello"
	warnings := ValidatePageFields(page, fm)
	for _, w := range warnings {
		if w.Field == "title" {
			t.Errorf("unexpected title warning: %v", w)
		}
	}
}

func TestValidatePageFields_ZeroValues(t *testing.T) {
	page := &engine.Page{}
	fm := &engine.Frontmatter{}
	fm.Title = "Present"
	warnings := ValidatePageFields(page, fm)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for zero-value optional fields, got %d: %v", len(warnings), warnings)
	}
}

func TestValidatePageFields_ValidValues(t *testing.T) {
	page := &engine.Page{}
	page.TOC.MinLevel = 2
	page.TOC.MaxLevel = 4
	page.Sidebar.Order = 5
	fm := &engine.Frontmatter{}
	fm.Title = "Present"
	warnings := ValidatePageFields(page, fm)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for valid values, got %d: %v", len(warnings), warnings)
	}
}

func TestValidatePageFields_TOCMinLevelOutOfRange(t *testing.T) {
	page := &engine.Page{}
	page.TOC.MinLevel = 99
	fm := &engine.Frontmatter{}
	fm.Title = "Present"
	warnings := ValidatePageFields(page, fm)
	found := false
	for _, w := range warnings {
		if w.Field == "toc.min_level" && w.Level == "warn" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning for toc.min_level: 99")
	}
}

func TestValidatePageFields_TOCMinGreaterThanMax(t *testing.T) {
	page := &engine.Page{}
	page.TOC.MinLevel = 4
	page.TOC.MaxLevel = 2
	fm := &engine.Frontmatter{}
	fm.Title = "Present"
	warnings := ValidatePageFields(page, fm)
	found := false
	for _, w := range warnings {
		if w.Field == "toc.min_level" && w.Level == "warn" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning for toc.min_level > toc.max_level")
	}
}

func TestValidatePageFields_SidebarOrderNegative(t *testing.T) {
	page := &engine.Page{}
	page.Sidebar.Order = -1
	fm := &engine.Frontmatter{}
	fm.Title = "Present"
	warnings := ValidatePageFields(page, fm)
	found := false
	for _, w := range warnings {
		if w.Field == "sidebar.order" && w.Level == "warn" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning for sidebar.order: -1")
	}
}

func TestValidatePageFields_LayoutBogus(t *testing.T) {
	page := &engine.Page{}
	fm := &engine.Frontmatter{}
	fm.Title = "Present"
	fm.Layout = "bogus"
	warnings := ValidatePageFields(page, fm)
	found := false
	for _, w := range warnings {
		if w.Field == "layout" && w.Level == "warn" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning for layout: bogus")
	}
}

func TestValidatePageFields_LayoutValid(t *testing.T) {
	page := &engine.Page{}
	fm := &engine.Frontmatter{}
	fm.Title = "Present"
	fm.Layout = "presentation"
	warnings := ValidatePageFields(page, fm)
	for _, w := range warnings {
		if w.Field == "layout" {
			t.Errorf("unexpected layout warning for valid layout: %v", w)
		}
	}
}

func TestValidatePageFields_BadgeVariantBogus(t *testing.T) {
	page := &engine.Page{}
	page.Sidebar.Badge.Variant = "bogus"
	fm := &engine.Frontmatter{}
	fm.Title = "Present"
	warnings := ValidatePageFields(page, fm)
	found := false
	for _, w := range warnings {
		if w.Field == "sidebar.badge.variant" && w.Level == "warn" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning for sidebar.badge.variant: bogus")
	}
}

func TestValidatePageFields_BadgeVariantValid(t *testing.T) {
	page := &engine.Page{}
	page.Sidebar.Badge.Variant = "tip"
	fm := &engine.Frontmatter{}
	fm.Title = "Present"
	warnings := ValidatePageFields(page, fm)
	for _, w := range warnings {
		if w.Field == "sidebar.badge.variant" {
			t.Errorf("unexpected badge variant warning for valid variant: %v", w)
		}
	}
}
