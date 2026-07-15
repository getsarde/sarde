package engine

import (
	_ "embed"
	"fmt"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed frontmatter_catalog.yaml
var frontmatterCatalogYAML []byte

// FrontmatterCatalog is the canonical description of every frontmatter
// field Sarde recognizes, grouped into categories, with a mapping from
// each implemented layout to the categories available on it. It is the
// single source of truth consumed by `sarde catalog` and Sarde Studio,
// and parity with the validator's known-key sets is enforced by tests.
type FrontmatterCatalog struct {
	Layouts         map[string][]string              `yaml:"layouts" json:"layouts"`
	CollectionTypes map[string]CatalogCollectionType `yaml:"collection_types" json:"collectionTypes"`
	Categories      []CatalogCategory                `yaml:"categories" json:"categories"`
}

// CatalogCollectionType holds name-based inference hints for a collection
// type (mirroring internal/collection/infer.go): matching names, the
// layout that type infers to, and extra field categories granted to
// matching collections regardless of layout.
type CatalogCollectionType struct {
	Names           []string `yaml:"names" json:"names"`
	Layout          string   `yaml:"layout" json:"layout"`
	ExtraCategories []string `yaml:"extra_categories" json:"extraCategories,omitempty"`
}

// CatalogCategory groups related frontmatter fields. Nested categories
// (sidebar, toc) describe the children of a single top-level mapping key
// named by ParentKey.
type CatalogCategory struct {
	Name      string         `yaml:"name" json:"name"`
	Label     string         `yaml:"label" json:"label"`
	Nested    bool           `yaml:"nested" json:"nested,omitempty"`
	ParentKey string         `yaml:"parent_key" json:"parentKey,omitempty"`
	Fields    []CatalogField `yaml:"fields" json:"fields"`
}

// CatalogField describes one frontmatter field: its key, widget type,
// human-facing label/description, and constraints. Object fields list
// their children in Fields; list-of-object fields describe item shape
// in Items. Unrendered marks fields the engine parses but no template
// renders yet — UIs should hide these from field pickers.
type CatalogField struct {
	Key         string         `yaml:"key" json:"key"`
	Type        string         `yaml:"type" json:"type"`
	Label       string         `yaml:"label" json:"label,omitempty"`
	Description string         `yaml:"description" json:"description,omitempty"`
	Required    bool           `yaml:"required" json:"required,omitempty"`
	Unrendered  bool           `yaml:"unrendered" json:"unrendered,omitempty"`
	Default     any            `yaml:"default" json:"default,omitempty"`
	Min         *float64       `yaml:"min" json:"min,omitempty"`
	Max         *float64       `yaml:"max" json:"max,omitempty"`
	MaxLength   *int           `yaml:"max_length" json:"maxLength,omitempty"`
	Options     []string       `yaml:"options" json:"options,omitempty"`
	Fields      []CatalogField `yaml:"fields" json:"fields,omitempty"`
	Items       []CatalogField `yaml:"items" json:"items,omitempty"`
}

var (
	catalogOnce sync.Once
	catalog     *FrontmatterCatalog
	catalogErr  error
)

// LoadFrontmatterCatalog parses the embedded catalog. The result is
// memoized; the embedded YAML cannot change at runtime.
func LoadFrontmatterCatalog() (*FrontmatterCatalog, error) {
	catalogOnce.Do(func() {
		var c FrontmatterCatalog
		if err := yaml.Unmarshal(frontmatterCatalogYAML, &c); err != nil {
			catalogErr = fmt.Errorf("parsing frontmatter catalog: %w", err)
			return
		}
		catalog = &c
	})
	return catalog, catalogErr
}
