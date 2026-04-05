package engine

// SiteBuilder coordinates the full six-phase build pipeline.
// Each pipeline stage is an injected dependency (interface), making the
// builder testable and each stage independently swappable.
type SiteBuilder struct {
	config   *SiteConfig
	theme    *ThemeConfig
	inferrer *DefaultsInferrer

	// Pipeline stages
	discoverer  ContentDiscoverer
	parser      FrontmatterParser
	validator   SchemaValidator
	transformer ContentTransformer
	renderer    MarkdownRenderer
	tmplEngine  TemplateEngine

	// Assembled data
	collections map[string]*Collection
	taxonomies  map[string]*Taxonomy
	pages       []*Page
}

// NewSiteBuilder creates a new SiteBuilder with initialized internal maps.
func NewSiteBuilder() *SiteBuilder {
	return &SiteBuilder{
		collections: make(map[string]*Collection),
		taxonomies:  make(map[string]*Taxonomy),
	}
}

// Build executes the full six-phase build pipeline:
//
//	Phase 1: INITIALIZE — load config, theme, plugins
//	Phase 2: DISCOVER   — walk content/, classify files
//	Phase 3: PARSE      — frontmatter + markdown (parallel)
//	Phase 4: ASSEMBLE   — collections, nav trees, taxonomies
//	Phase 5: RENDER     — execute templates (parallel)
//	Phase 6: WRITE      — output HTML + assets to public/
func (b *SiteBuilder) Build() (*BuildResult, error) {
	// Phase 1: INITIALIZE

	// Phase 2: DISCOVER

	// Phase 3: PARSE (parallel)

	// Phase 4: ASSEMBLE

	// Phase 5: RENDER (parallel)

	// Phase 6: WRITE

	return &BuildResult{}, nil
}
