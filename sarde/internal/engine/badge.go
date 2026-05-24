package engine

import "gopkg.in/yaml.v3"

// BadgeVariant is a semantic label for a sidebar badge's color scheme.
type BadgeVariant string

const (
	BadgeVariantDefault BadgeVariant = "default"
	BadgeVariantNote    BadgeVariant = "note"
	BadgeVariantTip     BadgeVariant = "tip"
	BadgeVariantSuccess BadgeVariant = "success"
	BadgeVariantCaution BadgeVariant = "caution"
	BadgeVariantDanger  BadgeVariant = "danger"
)

// VariantAliases maps legacy color names to semantic variants.
var VariantAliases = map[string]BadgeVariant{
	"green": BadgeVariantTip,
	"amber": BadgeVariantCaution,
	"red":   BadgeVariantDanger,
}

// Badge represents a sidebar badge with a display label and a semantic variant.
//
// Supports two YAML forms:
//
//	badge: "New"                    → Badge{Text: "New", Variant: "default"}
//	badge:
//	  text: "New"
//	  variant: "tip"                → Badge{Text: "New", Variant: "tip"}
type Badge struct {
	Text    string       `yaml:"text"`
	Variant BadgeVariant `yaml:"variant"`
}

func (b Badge) IsEmpty() bool { return b.Text == "" }

// CSSClass returns the CSS class for the badge variant (e.g. "sarde-badge-tip").
func (b Badge) CSSClass() string {
	if b.Variant == "" {
		return "sarde-badge-default"
	}
	return "sarde-badge-" + string(b.Variant)
}

func (b *Badge) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		b.Text = value.Value
		b.Variant = BadgeVariantDefault
		return nil
	}
	type plain Badge
	if err := value.Decode((*plain)(b)); err != nil {
		return err
	}
	if alias, ok := VariantAliases[string(b.Variant)]; ok {
		b.Variant = alias
	}
	if b.Variant == "" && b.Text != "" {
		b.Variant = BadgeVariantDefault
	}
	return nil
}
