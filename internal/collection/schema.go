package collection

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
)

// Field provenance markers for ResolvedField.Source.
const (
	SchemaSourceCatalog = "catalog"
	SchemaSourceCustom  = "custom"
)

// ResolvedField is one field of a collection's resolved frontmatter schema:
// a catalog field's shape plus provenance. Custom frontmatter_schema fields
// carry no Description/Unrendered/children — those knobs are catalog-only.
type ResolvedField struct {
	Type        string                   `json:"type"`
	Label       string                   `json:"label,omitempty"`
	Description string                   `json:"description,omitempty"`
	Required    bool                     `json:"required,omitempty"`
	Unrendered  bool                     `json:"unrendered,omitempty"`
	Default     any                      `json:"default,omitempty"`
	Min         *float64                 `json:"min,omitempty"`
	Max         *float64                 `json:"max,omitempty"`
	MaxLength   *int                     `json:"maxLength,omitempty"`
	Options     []string                 `json:"options,omitempty"`
	Fields      map[string]ResolvedField `json:"fields,omitempty"`
	Items       map[string]ResolvedField `json:"items,omitempty"`
	Source      string                   `json:"source"` // "catalog" | "custom"
}

// ResolvedSchema is the full frontmatter schema for one collection: every
// catalog field applicable to its effective layout and inferred type, merged
// with the collection's custom frontmatter_schema (custom wins on key
// collision). Consumed by `sarde schema` and Sarde Studio's schema editor.
type ResolvedSchema struct {
	Collection string                   `json:"collection"`
	Layout     string                   `json:"layout"`
	Type       string                   `json:"type"`
	Fields     map[string]ResolvedField `json:"fields"`
}

// UnknownCollectionError reports a collection that is neither a content/
// subdirectory nor configured in sarde.yaml.
type UnknownCollectionError struct{ Name string }

func (e *UnknownCollectionError) Error() string {
	return fmt.Sprintf("unknown collection %q: no content/%s directory and no sarde.yaml entry", e.Name, e.Name)
}

// ResolveSchema computes the resolved frontmatter schema for one collection.
// Like BuildEffectiveConfig it re-resolves sarde.yaml fresh on every call so
// callers always see the file's current state.
func ResolveSchema(projectDir, name string) (*ResolvedSchema, error) {
	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath: filepath.Join(projectDir, consts.FileSiteConfig),
		EnvPrefix:  "SARDE",
	})
	if err != nil {
		return nil, err
	}

	collectionDir := filepath.Join(projectDir, consts.DirContent, name)
	if info, statErr := os.Stat(collectionDir); statErr != nil || !info.IsDir() {
		if _, configured := cfg.Collections[name]; !configured {
			return nil, &UnknownCollectionError{Name: name}
		}
	}

	merged := MergeCollectionConfig(InferCollection(name), cfg.Collections[name])
	bucket := inferredBucket(name)

	cat, err := engine.LoadFrontmatterCatalog()
	if err != nil {
		return nil, err
	}

	rs := &ResolvedSchema{
		Collection: name,
		Layout:     string(merged.Layout),
		Type:       bucket,
		Fields:     make(map[string]ResolvedField),
	}

	for _, category := range applicableCategories(cat, string(merged.Layout), bucket) {
		if category.Nested && category.ParentKey != "" {
			rs.Fields[category.ParentKey] = ResolvedField{
				Type:   "object",
				Label:  category.Label,
				Fields: catalogFieldMap(category.Fields),
				Source: SchemaSourceCatalog,
			}
			continue
		}
		for _, f := range category.Fields {
			rs.Fields[f.Key] = catalogField(f)
		}
	}

	schema, err := content.LoadSchema(collectionDir)
	if err != nil {
		return nil, err
	}
	if schema != nil {
		for fieldName, def := range schema.Fields {
			rs.Fields[fieldName] = customField(def)
		}
	}

	return rs, nil
}

// applicableCategories filters the catalog's categories down to those active
// for the given layout, plus the inferred type's extra categories, in
// catalog order.
func applicableCategories(cat *engine.FrontmatterCatalog, layout, bucket string) []engine.CatalogCategory {
	active := make(map[string]bool)
	for _, n := range cat.Layouts[layout] {
		active[n] = true
	}
	if ct, ok := cat.CollectionTypes[bucket]; ok {
		for _, n := range ct.ExtraCategories {
			active[n] = true
		}
	}
	var out []engine.CatalogCategory
	for _, c := range cat.Categories {
		if active[c.Name] {
			out = append(out, c)
		}
	}
	return out
}

func catalogFieldMap(fields []engine.CatalogField) map[string]ResolvedField {
	if len(fields) == 0 {
		return nil
	}
	out := make(map[string]ResolvedField, len(fields))
	for _, f := range fields {
		out[f.Key] = catalogField(f)
	}
	return out
}

func catalogField(f engine.CatalogField) ResolvedField {
	return ResolvedField{
		Type:        f.Type,
		Label:       f.Label,
		Description: f.Description,
		Required:    f.Required,
		Unrendered:  f.Unrendered,
		Default:     f.Default,
		Min:         f.Min,
		Max:         f.Max,
		MaxLength:   f.MaxLength,
		Options:     f.Options,
		Fields:      catalogFieldMap(f.Fields),
		Items:       catalogFieldMap(f.Items),
		Source:      SchemaSourceCatalog,
	}
}

func customField(def engine.FieldDef) ResolvedField {
	return ResolvedField{
		Type:      def.Type,
		Label:     def.Label,
		Required:  def.Required,
		Default:   def.Default,
		Min:       def.Min,
		Max:       def.Max,
		MaxLength: def.MaxLength,
		Options:   def.Options,
		Source:    SchemaSourceCustom,
	}
}
