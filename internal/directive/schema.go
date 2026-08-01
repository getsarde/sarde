package directive

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"regexp"

	"github.com/getsarde/sarde/internal/engine"
	"gopkg.in/yaml.v3"
)

// KindContainer directives have a markdown body rendered recursively;
// KindLeaf directives capture their body as raw text.
const (
	KindContainer = "container"
	KindLeaf      = "leaf"
)

// nameRegex constrains directive names (and therefore filenames).
var nameRegex = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ValidName reports whether name is a legal generic directive name.
func ValidName(name string) bool { return nameRegex.MatchString(name) }

// Def is one loaded generic directive definition.
type Def struct {
	Name        string
	Kind        string // KindContainer | KindLeaf
	Label       string
	Description string
	Category    string
	Source      string // "site" | "theme"
	Bracket     *engine.CatalogDirectiveBracket
	Fields      []engine.CatalogDirectiveField
	Template    *htmltemplate.Template
	CSS         []byte

	yamlRaw []byte
	htmlRaw []byte
}

// defFile mirrors the directives/<name>.yaml schema. Field placement is not
// authorable: every generic directive field is an opening-fence attr.
type defFile struct {
	Name        string                          `yaml:"name"`
	Kind        string                          `yaml:"kind"`
	Label       string                          `yaml:"label"`
	Description string                          `yaml:"description"`
	Category    string                          `yaml:"category"`
	Bracket     *engine.CatalogDirectiveBracket `yaml:"bracket"`
	Fields      []engine.CatalogDirectiveField  `yaml:"fields"`
}

// parseDef validates the YAML schema and template source for one directive.
// stem is the filename stem the name must match.
func parseDef(stem string, yamlRaw, htmlRaw []byte, funcMap htmltemplate.FuncMap) (*Def, error) {
	dec := yaml.NewDecoder(bytes.NewReader(yamlRaw))
	dec.KnownFields(true)
	var df defFile
	if err := dec.Decode(&df); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if df.Name == "" {
		return nil, fmt.Errorf("missing required key %q", "name")
	}
	if !ValidName(df.Name) {
		return nil, fmt.Errorf("invalid name %q (must match %s)", df.Name, nameRegex.String())
	}
	if df.Name != stem {
		return nil, fmt.Errorf("name %q does not match filename stem %q", df.Name, stem)
	}
	if df.Kind != KindContainer && df.Kind != KindLeaf {
		return nil, fmt.Errorf("invalid kind %q (expected %q or %q)", df.Kind, KindContainer, KindLeaf)
	}
	if df.Label == "" {
		return nil, fmt.Errorf("missing required key %q", "label")
	}
	if df.Description == "" {
		return nil, fmt.Errorf("missing required key %q", "description")
	}
	if df.Category == "" {
		df.Category = "custom"
	}

	seen := make(map[string]bool, len(df.Fields))
	for i := range df.Fields {
		f := &df.Fields[i]
		if f.Name == "" {
			return nil, fmt.Errorf("field %d is missing a name", i+1)
		}
		if seen[f.Name] {
			return nil, fmt.Errorf("duplicate field name %q", f.Name)
		}
		seen[f.Name] = true
		if !engine.ValidDirectiveFieldTypes[f.Type] {
			return nil, fmt.Errorf("field %q has invalid type %q", f.Name, f.Type)
		}
		if f.Type == "enum" && len(f.Options) == 0 {
			return nil, fmt.Errorf("enum field %q has no options", f.Name)
		}
		// Generic directives support exactly one placement.
		f.Placement = "attr"
	}

	tmpl, err := htmltemplate.New(df.Name).Funcs(funcMap).Parse(string(htmlRaw))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	return &Def{
		Name:        df.Name,
		Kind:        df.Kind,
		Label:       df.Label,
		Description: df.Description,
		Category:    df.Category,
		Bracket:     df.Bracket,
		Fields:      df.Fields,
		Template:    tmpl,
		yamlRaw:     yamlRaw,
		htmlRaw:     htmlRaw,
	}, nil
}
