package engine

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func unmarshalBadge(t *testing.T, input string) Badge {
	t.Helper()
	var b Badge
	if err := yaml.Unmarshal([]byte(input), &b); err != nil {
		t.Fatalf("unmarshal %q: %v", input, err)
	}
	return b
}

func TestBadge_UnmarshalYAML_Scalar(t *testing.T) {
	b := unmarshalBadge(t, `"New"`)
	if b.Text != "New" {
		t.Errorf("Text = %q, want %q", b.Text, "New")
	}
	if b.Variant != BadgeVariantDefault {
		t.Errorf("Variant = %q, want %q", b.Variant, BadgeVariantDefault)
	}
}

func TestBadge_UnmarshalYAML_Mapping(t *testing.T) {
	b := unmarshalBadge(t, "text: Beta\nvariant: tip")
	if b.Text != "Beta" {
		t.Errorf("Text = %q, want %q", b.Text, "Beta")
	}
	if b.Variant != BadgeVariantTip {
		t.Errorf("Variant = %q, want %q", b.Variant, BadgeVariantTip)
	}
}

func TestBadge_UnmarshalYAML_MappingNoVariant(t *testing.T) {
	b := unmarshalBadge(t, "text: WIP")
	if b.Text != "WIP" {
		t.Errorf("Text = %q, want %q", b.Text, "WIP")
	}
	if b.Variant != BadgeVariantDefault {
		t.Errorf("Variant = %q, want %q", b.Variant, BadgeVariantDefault)
	}
}

func TestBadge_UnmarshalYAML_LegacyAlias(t *testing.T) {
	tests := []struct {
		alias string
		want  BadgeVariant
	}{
		{"green", BadgeVariantTip},
		{"amber", BadgeVariantCaution},
		{"red", BadgeVariantDanger},
	}
	for _, tt := range tests {
		b := unmarshalBadge(t, "text: X\nvariant: "+tt.alias)
		if b.Variant != tt.want {
			t.Errorf("alias %q: Variant = %q, want %q", tt.alias, b.Variant, tt.want)
		}
	}
}

func TestBadge_IsEmpty(t *testing.T) {
	if !(Badge{}).IsEmpty() {
		t.Error("zero Badge should be empty")
	}
	if (Badge{Text: "New"}).IsEmpty() {
		t.Error("Badge with text should not be empty")
	}
}

func TestBadge_CSSClass(t *testing.T) {
	tests := []struct {
		variant BadgeVariant
		want    string
	}{
		{BadgeVariantDefault, "sarde-badge-default"},
		{BadgeVariantTip, "sarde-badge-tip"},
		{BadgeVariantDanger, "sarde-badge-danger"},
		{"", "sarde-badge-default"},
	}
	for _, tt := range tests {
		b := Badge{Text: "X", Variant: tt.variant}
		if got := b.CSSClass(); got != tt.want {
			t.Errorf("CSSClass(%q) = %q, want %q", tt.variant, got, tt.want)
		}
	}
}

func TestBadge_UnmarshalYAML_InFrontmatter(t *testing.T) {
	// Scalar badge in frontmatter context
	var fm struct {
		Badge Badge `yaml:"badge"`
	}
	if err := yaml.Unmarshal([]byte("badge: New"), &fm); err != nil {
		t.Fatal(err)
	}
	if fm.Badge.Text != "New" || fm.Badge.Variant != BadgeVariantDefault {
		t.Errorf("scalar in struct: got %+v", fm.Badge)
	}

	// Mapping badge in frontmatter context
	fm.Badge = Badge{}
	if err := yaml.Unmarshal([]byte("badge:\n  text: Beta\n  variant: caution"), &fm); err != nil {
		t.Fatal(err)
	}
	if fm.Badge.Text != "Beta" || fm.Badge.Variant != BadgeVariantCaution {
		t.Errorf("mapping in struct: got %+v", fm.Badge)
	}
}
