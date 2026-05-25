package engine

// PageBanner defines a per-page announcement banner from frontmatter.
//
// Supports one YAML form:
//
//	banner:
//	  content: "This page is under construction"
//	  variant: "caution"
type PageBanner struct {
	Content string `yaml:"content"`
	Variant string `yaml:"variant"` // note | tip | caution | danger (defaults to "note")
}
