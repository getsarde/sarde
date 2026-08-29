package cfgutil

import (
	"sort"

	"gopkg.in/yaml.v3"
)

// DefaultsFromFields extracts the default values from blueprint-style YAML of
// the shape fields: {name: {type, label, hint, default, min, max}}. Shared by
// the clientplugins manifest defaults and external plugin blueprints.
func DefaultsFromFields(raw []byte) (map[string]any, error) {
	var doc struct {
		Fields map[string]map[string]any `yaml:"fields"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	defaults := make(map[string]any, len(doc.Fields))
	for name, field := range doc.Fields {
		if def, ok := field["default"]; ok {
			defaults[name] = def
		}
	}
	return defaults, nil
}

// FieldNames extracts every declared field name from blueprint-style YAML,
// sorted. Unlike DefaultsFromFields it includes fields that declare no
// default, so key validation does not treat a default-less field as unknown.
func FieldNames(raw []byte) ([]string, error) {
	var doc struct {
		Fields map[string]map[string]any `yaml:"fields"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(doc.Fields))
	for name := range doc.Fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// Field is the typed form of one blueprint entry, preserving every key the
// YAML may declare so catalogs can publish it verbatim.
type Field struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Label   string   `json:"label"`
	Hint    string   `json:"hint,omitempty"`
	Default any      `json:"default"`
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Options []Option `json:"options,omitempty"`
	Fields  []Field  `json:"fields,omitempty"`
}

// Option is one choice of a select-typed field.
type Option struct {
	Value any    `json:"value" yaml:"value"`
	Label string `json:"label" yaml:"label"`
}

type rawField struct {
	Type    string              `yaml:"type"`
	Label   string              `yaml:"label"`
	Hint    string              `yaml:"hint"`
	Default any                 `yaml:"default"`
	Min     *float64            `yaml:"min"`
	Max     *float64            `yaml:"max"`
	Options []Option            `yaml:"options"`
	Fields  map[string]rawField `yaml:"fields"`
}

func (r rawField) toField(name string) Field {
	return Field{
		Name: name, Type: r.Type, Label: r.Label, Hint: r.Hint,
		Default: r.Default, Min: r.Min, Max: r.Max, Options: r.Options,
		Fields: fieldsFromRaw(r.Fields),
	}
}

func fieldsFromRaw(raw map[string]rawField) []Field {
	if len(raw) == 0 {
		return nil
	}
	names := make([]string, 0, len(raw))
	for name := range raw {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]Field, 0, len(names))
	for _, name := range names {
		out = append(out, raw[name].toField(name))
	}
	return out
}

// ParseFields decodes blueprint-style YAML into typed fields sorted by name.
// An empty document yields an empty, non-nil slice.
func ParseFields(raw []byte) ([]Field, error) {
	var doc struct {
		Fields map[string]rawField `yaml:"fields"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	fields := fieldsFromRaw(doc.Fields)
	if fields == nil {
		fields = []Field{}
	}
	return fields, nil
}
