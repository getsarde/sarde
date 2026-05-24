package content

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/frostybee/sarde/internal/engine"
	"gopkg.in/yaml.v3"
)

// Validator implements engine.SchemaValidator.
type Validator struct{}

// Validate checks frontmatter against a schema and returns warnings.
// A nil schema means no validation — returns nil.
// Never blocks the build; all issues are warnings.
func (v *Validator) Validate(fm map[string]interface{}, schema *engine.FrontmatterSchema) []engine.ValidationWarning {
	if schema == nil || len(schema.Fields) == 0 {
		return nil
	}

	var warnings []engine.ValidationWarning

	for name, def := range schema.Fields {
		val, exists := fm[name]

		if def.Required && (!exists || val == nil || val == "") {
			warnings = append(warnings, engine.ValidationWarning{
				Field:   name,
				Message: fmt.Sprintf("required field %q is missing", name),
				Level:   "error",
			})
			continue
		}

		if !exists || val == nil {
			continue
		}

		// Type checking
		if w := validateType(name, val, def); w != nil {
			warnings = append(warnings, *w)
			continue // skip constraint checks if type is wrong
		}

		// Constraint checking
		warnings = append(warnings, validateConstraints(name, val, def)...)
	}

	return warnings
}

func validateType(name string, val interface{}, def engine.FieldDef) *engine.ValidationWarning {
	switch def.Type {
	case "string", "text", "textarea", "color", "image", "url":
		if _, ok := val.(string); !ok {
			return &engine.ValidationWarning{
				Field:   name,
				Message: fmt.Sprintf("expected string, got %T", val),
				Level:   "error",
			}
		}
	case "int", "number":
		if !isNumeric(val) {
			return &engine.ValidationWarning{
				Field:   name,
				Message: fmt.Sprintf("expected number, got %T", val),
				Level:   "error",
			}
		}
	case "float":
		if !isNumeric(val) {
			return &engine.ValidationWarning{
				Field:   name,
				Message: fmt.Sprintf("expected float, got %T", val),
				Level:   "error",
			}
		}
	case "bool", "toggle":
		if _, ok := val.(bool); !ok {
			return &engine.ValidationWarning{
				Field:   name,
				Message: fmt.Sprintf("expected bool, got %T", val),
				Level:   "error",
			}
		}
	case "date":
		if !isDate(val) {
			return &engine.ValidationWarning{
				Field:   name,
				Message: fmt.Sprintf("expected date, got %T", val),
				Level:   "error",
			}
		}
	case "list", "tags":
		if _, ok := val.([]interface{}); !ok {
			return &engine.ValidationWarning{
				Field:   name,
				Message: fmt.Sprintf("expected list, got %T", val),
				Level:   "error",
			}
		}
	case "enum", "select", "radio":
		s, ok := val.(string)
		if !ok {
			return &engine.ValidationWarning{
				Field:   name,
				Message: fmt.Sprintf("expected string for %s, got %T", def.Type, val),
				Level:   "error",
			}
		}
		if len(def.Options) > 0 && !contains(def.Options, s) {
			return &engine.ValidationWarning{
				Field:   name,
				Message: fmt.Sprintf("value %q not in options %v", s, def.Options),
				Level:   "warning",
			}
		}
	case "group":
		if _, ok := val.(map[string]interface{}); !ok {
			if _, ok := val.(map[interface{}]interface{}); !ok {
				return &engine.ValidationWarning{
					Field:   name,
					Message: fmt.Sprintf("expected map for group, got %T", val),
					Level:   "error",
				}
			}
		}
	}
	return nil
}

func validateConstraints(name string, val interface{}, def engine.FieldDef) []engine.ValidationWarning {
	var warnings []engine.ValidationWarning

	// Numeric min/max
	if def.Min != nil || def.Max != nil {
		if n, ok := toFloat64(val); ok {
			if def.Min != nil && n < *def.Min {
				warnings = append(warnings, engine.ValidationWarning{
					Field:   name,
					Message: fmt.Sprintf("value %.0f is below minimum %.0f", n, *def.Min),
					Level:   "warning",
				})
			}
			if def.Max != nil && n > *def.Max {
				warnings = append(warnings, engine.ValidationWarning{
					Field:   name,
					Message: fmt.Sprintf("value %.0f exceeds maximum %.0f", n, *def.Max),
					Level:   "warning",
				})
			}
		}
	}

	// String max_length
	if def.MaxLength != nil {
		if s, ok := val.(string); ok && len(s) > *def.MaxLength {
			warnings = append(warnings, engine.ValidationWarning{
				Field:   name,
				Message: fmt.Sprintf("length %d exceeds max_length %d", len(s), *def.MaxLength),
				Level:   "warning",
			})
		}
	}

	return warnings
}

// ---------------------------------------------------------------------------
// Schema loading
// ---------------------------------------------------------------------------

// schemaFile is the YAML wrapper for loading config.yaml
type schemaFile struct {
	FrontmatterSchema *engine.FrontmatterSchema `yaml:"frontmatter_schema"`
}

// LoadSchema reads config.yaml from a collection directory and returns the schema.
// Returns (nil, nil) if no config file exists.
func LoadSchema(collectionDir string) (*engine.FrontmatterSchema, error) {
	for _, name := range []string{"config.yaml", "config.yml"} {
		path := filepath.Join(collectionDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}
		var sf schemaFile
		if err := yaml.Unmarshal(data, &sf); err != nil {
			return nil, err
		}
		return sf.FrontmatterSchema, nil
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Defaults application
// ---------------------------------------------------------------------------

// ApplyDefaults fills in missing frontmatter fields from schema defaults.
// Returns a new map (does not mutate the input).
func ApplyDefaults(fm map[string]interface{}, schema *engine.FrontmatterSchema) map[string]interface{} {
	if schema == nil || len(schema.Fields) == 0 {
		return fm
	}
	result := make(map[string]interface{}, len(fm))
	for k, v := range fm {
		result[k] = v
	}
	for name, def := range schema.Fields {
		if _, exists := result[name]; !exists && def.Default != nil {
			result[name] = def.Default
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func isNumeric(v interface{}) bool {
	switch v.(type) {
	case int, int64, float64:
		return true
	}
	return false
}

func isDate(v interface{}) bool {
	switch v.(type) {
	case time.Time:
		return true
	case string:
		s := v.(string)
		_, err1 := time.Parse(time.RFC3339, s)
		_, err2 := time.Parse("2006-01-02", s)
		return err1 == nil || err2 == nil
	}
	return false
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
