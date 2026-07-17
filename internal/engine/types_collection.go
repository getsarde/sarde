package engine

// ---------------------------------------------------------------------------
// Collection
// ---------------------------------------------------------------------------

// Collection represents a group of pages (blog, docs, courses, etc.).
type Collection struct {
	Name      string
	Title     string
	Config    *CollectionConfig
	Pages     []*Page
	Featured  []*Page // subset of Pages with frontmatter `featured: true`
	Sections  []*Section
	NavTree   *NavTree            // default language nav tree (backward compat)
	NavTrees  map[string]*NavTree // per-language nav trees (i18n)
	IndexPage *Page
	IsTabbed  bool       // true when docs tabs are auto-detected or forced
	Tabs      []*DocsTab // ordered by weight, then title

	// Versioning
	Versioning        *VersionConfig
	CompositeNavTrees map[string]*NavTree   // keyed by langVersionKey(lang, ver) for versioned collections
	CompositeTabSets  map[string][]*DocsTab // keyed by langVersionKey(lang, ver) for versioned+tabbed collections

	// Labs
	LabNavTrees   map[string]*NavTree // keyed by lab section permalink, for lab-scoped sidebar
	IsMultiCourse bool                // true when the labs collection has a course grouping layer
}

// CollectionConfig holds per-collection settings (auto-detected or explicit).
type CollectionConfig struct {
	SortBy     string
	SortOrder  string
	Layout     LayoutType
	Permalink  string
	Paginate   int
	Feed       bool
	Sidebar    *SidebarConfig
	TOC        *TOCConfig
	PrevNext   *PrevNextConfig
	Tabs       *bool          // nil = auto-detect, true = force tabs, false = disable tabs
	Versioning *VersionConfig // nil = no versioning
	Labs       *LabsConfig    // nil = not a labs collection
}

// SidebarConfig controls sidebar behavior for docs-layout collections.
type SidebarConfig struct {
	Collapsible        bool
	CollapsedByDefault bool
	MaxDepth           int
	Search             bool

	// CollapseLevel, when > 0, expands groups at depth <= N by default and
	// collapses deeper groups. 0 = unset (CollapsedByDefault governs).
	CollapseLevel int

	// Overrides holds sidebar.yaml path-keyed node overrides
	// (collection-relative path -> override). Nil unless sidebar.yaml sets any.
	Overrides map[string]*SidebarOverride

	// TabOverrides holds sidebar.yaml tab-bar overrides (tab slug -> override).
	TabOverrides map[string]*TabOverride

	// Build-time bookkeeping for unmatched-key warnings; see sidebar_override.go.
	matchedOverrides map[string]bool
	matchedTabs      map[string]bool
}

// TOCConfig controls table of contents rendering.
type TOCConfig struct {
	Enabled         bool
	MinLevel        int
	MaxLevel        int
	ScrollHighlight bool
}

// PrevNextConfig controls prev/next navigation links.
type PrevNextConfig struct {
	Enabled bool
	Labels  [2]string
}

// ---------------------------------------------------------------------------
// DocsTab
// ---------------------------------------------------------------------------

// DocsTab represents one tab in a tabbed docs collection.
// Each tab corresponds to a top-level subdirectory with its own nav tree.
type DocsTab struct {
	Title       string
	Description string
	Icon        string // emoji, icon name, or SVG path
	Slug        string // directory name, used for URL prefix matching
	Order       int
	Permalink   string   // URL of the tab's index page
	Section     *Section // the top-level section backing this tab
	NavTree     *NavTree
	NavTrees    map[string]*NavTree // per-language (i18n)
	Pages       []*Page
	IndexPage   *Page
}

// ---------------------------------------------------------------------------
// Section
// ---------------------------------------------------------------------------

// LabsConfig holds labs-collection-specific settings.
type LabsConfig struct {
	StepLabel string // "Lab" (default), configurable to "Exercise", "Activity", etc.
}

// SectionDepth returns the nesting depth of a section (root = 0).
func SectionDepth(sec *Section) int {
	depth := 0
	for s := sec.Parent; s != nil; s = s.Parent {
		depth++
	}
	return depth
}

// Section represents a directory with child pages and sub-sections.
type Section struct {
	Title       string
	Slug        string
	Permalink   string
	Pages       []*Page
	Sections    []*Section
	IndexPage   *Page
	Parent      *Section
	Collection  *Collection
	Transparent bool
	Render      bool
}
