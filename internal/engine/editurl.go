package engine

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// EditURLValue represents the per-page edit_url frontmatter field.
//
// Supports three YAML forms:
//
//	edit_url: false            → suppress the edit link
//	edit_url: true             → use the site-wide edit URL (default)
//	edit_url: "https://..."    → custom URL for this page
type EditURLValue struct {
	Disabled  bool
	CustomURL string
}

func (e *EditURLValue) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		if value.Tag == "!!bool" {
			var b bool
			if err := value.Decode(&b); err != nil {
				return err
			}
			e.Disabled = !b
			return nil
		}
		e.CustomURL = value.Value
		return nil
	}
	return fmt.Errorf("edit_url: expected bool or string, got %v", value.Tag)
}
