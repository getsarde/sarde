package engine

import "gopkg.in/yaml.v3"

// NavOverride represents a per-page prev/next frontmatter override.
//
// Supports three YAML forms:
//
//	prev: false                             → suppress the nav link
//	prev: "some-slug"                       → use page with this slug
//	prev: { link: "/url/", label: "..." }   → explicit URL + label
type NavOverride struct {
	Disabled bool   `yaml:"-"`
	Slug     string `yaml:"-"`
	Link     string `yaml:"link"`
	Label    string `yaml:"label"`
}

func (n *NavOverride) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		if value.Tag == "!!bool" {
			var b bool
			if err := value.Decode(&b); err != nil {
				return err
			}
			n.Disabled = !b
			return nil
		}
		n.Slug = value.Value
		return nil
	case yaml.MappingNode:
		type plain NavOverride
		return value.Decode((*plain)(n))
	}
	return nil
}
