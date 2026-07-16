package engine

import (
	_ "embed"
	"fmt"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed directive_catalog.yaml
var directiveCatalogYAML []byte

// DirectiveCatalog is the canonical description of every ::: block
// directive Sarde recognizes, grouped into categories. It is the single
// source of truth consumed by `sarde directives` and Sarde Studio's
// directive picker, and is built from each extension's parser grammar
// (internal/content/markdown/extensions/<name>/parser.go), not the docs.
type DirectiveCatalog struct {
	Categories []DirectiveCategory `yaml:"categories" json:"categories"`
}

// DirectiveCategory groups related directives for display.
type DirectiveCategory struct {
	Name       string             `yaml:"name" json:"name"`
	Label      string             `yaml:"label" json:"label"`
	Directives []CatalogDirective `yaml:"directives" json:"directives"`
}

// CatalogDirective describes one ::: block directive's syntax: its exact
// fence name, the optional [bracket] group, the attr/flag fields on the
// opening fence, and either a literal body template or a repeatable child
// block template for container directives.
type CatalogDirective struct {
	Name        string `yaml:"name" json:"name"`
	Label       string `yaml:"label" json:"label"`
	Description string `yaml:"description" json:"description"`
	Kind        string `yaml:"kind" json:"kind"` // "callout" | "block"

	// Bracket describes [Title]/[Summary]/[Label] support. Nil when the
	// directive's grammar has no bracket group at all (e.g. badge, video).
	Bracket *CatalogDirectiveBracket `yaml:"bracket,omitempty" json:"bracket,omitempty"`

	// Fields are attr/flag fields only, never the bracket. Order is the
	// picker form's display order.
	Fields []CatalogDirectiveField `yaml:"fields,omitempty" json:"fields,omitempty"`

	// BodyTemplate is literal body text (may contain ${...} editor snippet
	// placeholders). Empty for leaf directives with no body (video,
	// link-button, link-card) and for container directives.
	BodyTemplate string `yaml:"body_template,omitempty" json:"bodyTemplate,omitempty"`

	// ChildTemplate describes a repeatable nested child block for container
	// directives; a literal "N" token is replaced with the 1-based index.
	// ChildCountField optionally names the field whose value drives the
	// repeat count (falling back to ChildDefaultCount).
	ChildTemplate     string `yaml:"child_template,omitempty" json:"childTemplate,omitempty"`
	ChildDefaultCount int    `yaml:"child_default_count,omitempty" json:"childDefaultCount,omitempty"`
	ChildCountField   string `yaml:"child_count_field,omitempty" json:"childCountField,omitempty"`
}

// CatalogDirectiveBracket describes the [bracket] group on an opening fence.
type CatalogDirectiveBracket struct {
	Label       string `yaml:"label" json:"label"`
	Required    bool   `yaml:"required" json:"required"` // figure only: brackets mandatory (may be empty)
	Placeholder string `yaml:"placeholder" json:"placeholder"`
}

// CatalogDirectiveField is one attr/flag on the opening fence. Placement
// controls the exact emitted syntax — see directive_catalog.yaml's header.
type CatalogDirectiveField struct {
	Name        string   `yaml:"name" json:"name"`
	Label       string   `yaml:"label" json:"label"`
	Type        string   `yaml:"type" json:"type"`           // string | enum | boolean | number | icon
	Placement   string   `yaml:"placement" json:"placement"` // attr | bare-flag | quoted-flag | bare-icon | paren-flag
	Options     []string `yaml:"options,omitempty" json:"options,omitempty"`
	Default     string   `yaml:"default,omitempty" json:"default,omitempty"`
	Required    bool     `yaml:"required,omitempty" json:"required,omitempty"`
	Placeholder string   `yaml:"placeholder,omitempty" json:"placeholder,omitempty"`
}

var (
	directiveCatalogOnce sync.Once
	directiveCatalog     *DirectiveCatalog
	directiveCatalogErr  error
)

// LoadDirectiveCatalog parses the embedded directive catalog. The result is
// memoized; the embedded YAML cannot change at runtime.
func LoadDirectiveCatalog() (*DirectiveCatalog, error) {
	directiveCatalogOnce.Do(func() {
		var c DirectiveCatalog
		if err := yaml.Unmarshal(directiveCatalogYAML, &c); err != nil {
			directiveCatalogErr = fmt.Errorf("parsing directive catalog: %w", err)
			return
		}
		directiveCatalog = &c
	})
	return directiveCatalog, directiveCatalogErr
}
