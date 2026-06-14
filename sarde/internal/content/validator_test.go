package content

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/frostybee/sarde/internal/engine"
)

func float64Ptr(v float64) *float64 { return &v }
func intPtr(v int) *int             { return &v }

func TestValidate_NilSchema(t *testing.T) {
	v := &Validator{}
	warnings := v.Validate(map[string]interface{}{"title": "Test"}, nil)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for nil schema, got %d", len(warnings))
	}
}

func TestValidate_RequiredFieldMissing(t *testing.T) {
	v := &Validator{}
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"title": {Type: "string", Required: true},
		},
	}
	warnings := v.Validate(map[string]interface{}{}, schema)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if warnings[0].Field != "title" {
		t.Errorf("Field = %q, want %q", warnings[0].Field, "title")
	}
}

func TestValidate_RequiredFieldPresent(t *testing.T) {
	v := &Validator{}
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"title": {Type: "string", Required: true},
		},
	}
	warnings := v.Validate(map[string]interface{}{"title": "Hello"}, schema)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestValidate_TypeString(t *testing.T) {
	v := &Validator{}
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"title": {Type: "string"},
		},
	}
	warnings := v.Validate(map[string]interface{}{"title": 42}, schema)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
}

func TestValidate_TypeInt(t *testing.T) {
	v := &Validator{}
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"order": {Type: "int"},
		},
	}
	// Valid
	warnings := v.Validate(map[string]interface{}{"order": 5}, schema)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for int value, got %d", len(warnings))
	}
	// Invalid
	warnings = v.Validate(map[string]interface{}{"order": "heavy"}, schema)
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for string value, got %d", len(warnings))
	}
}

func TestValidate_TypeBool(t *testing.T) {
	v := &Validator{}
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"draft": {Type: "bool"},
		},
	}
	warnings := v.Validate(map[string]interface{}{"draft": "true"}, schema)
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for string 'true', got %d", len(warnings))
	}
}

func TestValidate_TypeDateValid(t *testing.T) {
	v := &Validator{}
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"date": {Type: "date"},
		},
	}
	warnings := v.Validate(map[string]interface{}{"date": "2025-01-15"}, schema)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for valid date, got %d", len(warnings))
	}
}

func TestValidate_TypeDateInvalid(t *testing.T) {
	v := &Validator{}
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"date": {Type: "date"},
		},
	}
	warnings := v.Validate(map[string]interface{}{"date": "not-a-date"}, schema)
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for invalid date, got %d", len(warnings))
	}
}

func TestValidate_TypeEnumValid(t *testing.T) {
	v := &Validator{}
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"status": {Type: "enum", Options: []string{"draft", "published"}},
		},
	}
	warnings := v.Validate(map[string]interface{}{"status": "draft"}, schema)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestValidate_TypeEnumInvalid(t *testing.T) {
	v := &Validator{}
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"status": {Type: "enum", Options: []string{"draft", "published"}},
		},
	}
	warnings := v.Validate(map[string]interface{}{"status": "unknown"}, schema)
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
}

func TestValidate_MinMax(t *testing.T) {
	v := &Validator{}
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"duration": {Type: "int", Min: float64Ptr(1), Max: float64Ptr(480)},
		},
	}
	// Below min
	warnings := v.Validate(map[string]interface{}{"duration": 0}, schema)
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for below min, got %d", len(warnings))
	}
	// Above max
	warnings = v.Validate(map[string]interface{}{"duration": 999}, schema)
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for above max, got %d", len(warnings))
	}
	// In range
	warnings = v.Validate(map[string]interface{}{"duration": 60}, schema)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for in-range value, got %d", len(warnings))
	}
}

func TestValidate_MaxLength(t *testing.T) {
	v := &Validator{}
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"description": {Type: "string", MaxLength: intPtr(10)},
		},
	}
	warnings := v.Validate(map[string]interface{}{"description": "this is too long"}, schema)
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
	warnings = v.Validate(map[string]interface{}{"description": "short"}, schema)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %d", len(warnings))
	}
}

func TestValidate_TypeRadioValid(t *testing.T) {
	v := &Validator{}
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"color": {Type: "radio", Options: []string{"red", "green", "blue"}},
		},
	}
	if w := v.Validate(map[string]interface{}{"color": "red"}, schema); len(w) != 0 {
		t.Errorf("radio with valid option: got %v, want none", w)
	}
}

func TestValidate_TypeRadioInvalid(t *testing.T) {
	v := &Validator{}
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"color": {Type: "radio", Options: []string{"red", "green", "blue"}},
		},
	}
	if w := v.Validate(map[string]interface{}{"color": "purple"}, schema); len(w) != 1 {
		t.Errorf("radio with invalid option: got %d warnings, want 1", len(w))
	}
	if w := v.Validate(map[string]interface{}{"color": 42}, schema); len(w) != 1 {
		t.Errorf("radio with non-string: got %d warnings, want 1", len(w))
	}
}

func TestValidate_TypeGroup(t *testing.T) {
	v := &Validator{}
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"meta": {Type: "group"},
		},
	}
	if w := v.Validate(map[string]interface{}{"meta": map[string]interface{}{"a": 1}}, schema); len(w) != 0 {
		t.Errorf("group with map[string]interface{}: got %v, want none", w)
	}
	if w := v.Validate(map[string]interface{}{"meta": map[interface{}]interface{}{"a": 1}}, schema); len(w) != 0 {
		t.Errorf("group with map[interface{}]interface{}: got %v, want none", w)
	}
	if w := v.Validate(map[string]interface{}{"meta": "not-a-map"}, schema); len(w) != 1 {
		t.Errorf("group with string: got %d warnings, want 1", len(w))
	}
}

func TestValidate_EmptySchema(t *testing.T) {
	v := &Validator{}
	schema := &engine.FrontmatterSchema{Fields: map[string]engine.FieldDef{}}
	warnings := v.Validate(map[string]interface{}{"unknown": "value"}, schema)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for empty schema, got %d", len(warnings))
	}
}

func TestApplyDefaults_FillsMissing(t *testing.T) {
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"draft":  {Type: "bool", Default: false},
			"status": {Type: "string", Default: "draft"},
		},
	}
	fm := map[string]interface{}{"title": "Test"}
	result := ApplyDefaults(fm, schema)

	if result["draft"] != false {
		t.Errorf("draft = %v, want false", result["draft"])
	}
	if result["status"] != "draft" {
		t.Errorf("status = %v, want %q", result["status"], "draft")
	}
	if result["title"] != "Test" {
		t.Errorf("title = %v, want %q", result["title"], "Test")
	}
}

func TestApplyDefaults_DoesNotOverride(t *testing.T) {
	schema := &engine.FrontmatterSchema{
		Fields: map[string]engine.FieldDef{
			"status": {Type: "string", Default: "draft"},
		},
	}
	fm := map[string]interface{}{"status": "published"}
	result := ApplyDefaults(fm, schema)

	if result["status"] != "published" {
		t.Errorf("status = %v, want %q (should not override)", result["status"], "published")
	}
}

func TestLoadSchema_ValidFile(t *testing.T) {
	dir := t.TempDir()
	content := `frontmatter_schema:
  fields:
    title:
      type: string
      required: true
    weight:
      type: int
      min: 0
`
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0644)

	schema, err := LoadSchema(dir)
	if err != nil {
		t.Fatalf("LoadSchema error: %v", err)
	}
	if schema == nil {
		t.Fatal("schema should not be nil")
	}
	if len(schema.Fields) != 2 {
		t.Errorf("Fields len = %d, want 2", len(schema.Fields))
	}
	if !schema.Fields["title"].Required {
		t.Error("title should be required")
	}
}

func TestLoadSchema_MissingFile(t *testing.T) {
	schema, err := LoadSchema(t.TempDir())
	if err != nil {
		t.Fatalf("LoadSchema error: %v", err)
	}
	if schema != nil {
		t.Error("schema should be nil for missing file")
	}
}

func TestLoadSchema_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("{{invalid"), 0644)

	_, err := LoadSchema(dir)
	if err == nil {
		t.Error("LoadSchema should error for invalid YAML")
	}
}
