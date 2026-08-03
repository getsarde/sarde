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
