package engine

// ContentDiscoverer walks the content directory and returns file paths grouped by collection.
type ContentDiscoverer interface {
	Discover(contentDir string) (map[string][]string, error)
}

// FrontmatterParser splits a raw .md file into a frontmatter map and Markdown body.
// Supports YAML (---), TOML (+++), and JSON ({}) delimiters.
type FrontmatterParser interface {
	Parse(raw []byte) (map[string]interface{}, string, error)
}

// SchemaValidator validates frontmatter against a collection's schema definition.
// Returns warnings (never errors) — builds always succeed.
// A nil schema means no validation (skip entirely).
type SchemaValidator interface {
	Validate(fm map[string]interface{}, schema *FrontmatterSchema) []ValidationWarning
}

// ContentTransformer enriches a page with computed fields (word count, reading time, excerpt, slug).
type ContentTransformer interface {
	Transform(page *Page) error
}

// MarkdownRenderer converts Markdown content to HTML via the Goldmark pipeline.
type MarkdownRenderer interface {
	Render(markdown string) (RenderResult, error)
}

// TemplateEngine loads templates from the resolved lookup chain and renders pages to HTML.
type TemplateEngine interface {
	Load(resolver *ThemeResolver) error
	Render(templateName string, data *RouteData) ([]byte, error)
}
