package content

import (
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
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

// ---------------------------------------------------------------------------
// Layer 1: Unknown field detection
// ---------------------------------------------------------------------------

func TestDetectUnknownFields_UnknownTopLevel(t *testing.T) {
	fmMap := map[string]any{"titl": "oops", "title": "ok"}
	warnings := DetectUnknownFields(fmMap, nil, nil, "test.md")
	found := false
	for _, w := range warnings {
		if w.Field == "titl" && w.Level == "warning" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for unknown top-level key \"titl\"")
	}
}

func TestDetectUnknownFields_KnownKey(t *testing.T) {
	fmMap := map[string]any{"title": "hello", "draft": true, "tags": []string{"go"}}
	warnings := DetectUnknownFields(fmMap, nil, nil, "test.md")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for known keys, got %d: %v", len(warnings), warnings)
	}
}

func TestDetectUnknownFields_UnknownSidebarKey(t *testing.T) {
	fmMap := map[string]any{
		"sidebar": map[string]any{"labl": "typo", "label": "ok"},
	}
	warnings := DetectUnknownFields(fmMap, nil, nil, "test.md")
	found := false
	for _, w := range warnings {
		if w.Field == "sidebar.labl" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for unknown sidebar key \"labl\"")
	}
}

func TestDetectUnknownFields_UnknownTOCKey(t *testing.T) {
	fmMap := map[string]any{
		"toc": map[string]any{"min_lvl": 2},
	}
	warnings := DetectUnknownFields(fmMap, nil, nil, "test.md")
	found := false
	for _, w := range warnings {
		if w.Field == "toc.min_lvl" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for unknown toc key \"min_lvl\"")
	}
}

func TestDetectUnknownFields_ParamsAndCascadeSkipped(t *testing.T) {
	fmMap := map[string]any{
		"params":  map[string]any{"custom_field": "value"},
		"cascade": map[string]any{"draft": true},
	}
	warnings := DetectUnknownFields(fmMap, nil, nil, "test.md")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for params/cascade, got %d: %v", len(warnings), warnings)
	}
}

func TestDetectUnknownFields_SchemaDefinedField(t *testing.T) {
	fmMap := map[string]any{"difficulty": "advanced"}
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"difficulty": {Type: "string"},
		},
	}
	warnings := DetectUnknownFields(fmMap, schema, nil, "test.md")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for schema-defined field, got %d: %v", len(warnings), warnings)
	}
}

func TestDetectUnknownFields_TaxonomyKeySkipped(t *testing.T) {
	fmMap := map[string]any{"authors": []string{"Jane Doe"}, "title": "ok"}
	taxCfg := map[string]config.TaxonomyConfig{
		"authors": {Singular: "author"},
	}
	warnings := DetectUnknownFields(fmMap, nil, taxCfg, "test.md")
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for taxonomy key, got %d: %v", len(warnings), warnings)
	}
}

// ---------------------------------------------------------------------------
// Layer 2: Slug validation
// ---------------------------------------------------------------------------

func TestValidatePageFields_SlugWhitespace(t *testing.T) {
	page := &engine.Page{}
	fm := &engine.Frontmatter{}
	fm.Title = "Present"
	fm.Slug = "   "
	warnings := ValidatePageFields(page, fm)
	found := false
	for _, w := range warnings {
		if w.Field == "slug" && w.Level == "warning" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for whitespace-only slug")
	}
}

func TestValidatePageFields_SlugValid(t *testing.T) {
	page := &engine.Page{}
	fm := &engine.Frontmatter{}
	fm.Title = "Present"
	fm.Slug = "my-page"
	warnings := ValidatePageFields(page, fm)
	for _, w := range warnings {
		if w.Field == "slug" {
			t.Errorf("unexpected slug warning for valid slug: %v", w)
		}
	}
}

// ---------------------------------------------------------------------------
// Layer 2: Head tag validation
// ---------------------------------------------------------------------------

func TestValidatePageFields_HeadTagInvalid(t *testing.T) {
	page := &engine.Page{}
	fm := &engine.Frontmatter{}
	fm.Title = "Present"
	fm.Head = []engine.HeadTag{{Tag: "div", Attrs: map[string]string{"class": "x"}}}
	warnings := ValidatePageFields(page, fm)
	found := false
	for _, w := range warnings {
		if w.Field == "head[0].tag" && w.Level == "warning" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for invalid head tag \"div\"")
	}
}

func TestValidatePageFields_HeadTagValid(t *testing.T) {
	page := &engine.Page{}
	fm := &engine.Frontmatter{}
	fm.Title = "Present"
	fm.Head = []engine.HeadTag{
		{Tag: "meta", Attrs: map[string]string{"name": "robots"}},
		{Tag: "link", Attrs: map[string]string{"rel": "canonical"}},
	}
	warnings := ValidatePageFields(page, fm)
	for _, w := range warnings {
		if w.Field == "head[0].tag" || w.Field == "head[1].tag" {
			t.Errorf("unexpected head tag warning for valid tags: %v", w)
		}
	}
}
