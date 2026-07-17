package external

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
)

// BlueprintField describes one configurable field declared in blueprint.yaml.
// Consumed by the desktop app for a future plugin settings UI.
type BlueprintField struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Label   string   `json:"label"`
	Hint    string   `json:"hint"`
	Default any      `json:"default"`
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
}

// LoadBlueprint parses {dir}/blueprint.yaml into field metadata. A missing
// file yields (nil, nil).
func LoadBlueprint(dir string) ([]BlueprintField, error) {
	data, err := os.ReadFile(filepath.Join(dir, consts.FilePluginBlueprint))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc struct {
		Fields map[string]struct {
			Type    string   `yaml:"type"`
			Label   string   `yaml:"label"`
			Hint    string   `yaml:"hint"`
			Default any      `yaml:"default"`
			Min     *float64 `yaml:"min"`
			Max     *float64 `yaml:"max"`
		} `yaml:"fields"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", consts.FilePluginBlueprint, err)
	}
	fields := make([]BlueprintField, 0, len(doc.Fields))
	for name, f := range doc.Fields {
		fields = append(fields, BlueprintField{
			Name: name, Type: f.Type, Label: f.Label, Hint: f.Hint,
			Default: f.Default, Min: f.Min, Max: f.Max,
		})
	}
	return fields, nil
}

// LoadDefaults resolves a plugin's shipped default config: blueprint.yaml
// field defaults overlaid by the flat key/value map in config.yaml. Either
// file may be absent. User overrides from sarde.yaml plugins.config.{slug}
// are applied later, at hook time, by the plugin manager.
func LoadDefaults(dir string) (map[string]any, error) {
	defaults := map[string]any{}

	if data, err := os.ReadFile(filepath.Join(dir, consts.FilePluginBlueprint)); err == nil {
		fromFields, err := cfgutil.DefaultsFromFields(data)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", consts.FilePluginBlueprint, err)
		}
		for k, v := range fromFields {
			defaults[k] = v
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if data, err := os.ReadFile(filepath.Join(dir, consts.FilePluginConfig)); err == nil {
		var flat map[string]any
		if err := yaml.Unmarshal(data, &flat); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", consts.FilePluginConfig, err)
		}
		for k, v := range flat {
			defaults[k] = v
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	return defaults, nil
}
