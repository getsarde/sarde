package cfgutil

import "gopkg.in/yaml.v3"

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
