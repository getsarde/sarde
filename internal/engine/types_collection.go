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

	// Per-language views, keyed by language code. Pages, Featured,
	// IndexPage and Title hold every language together; on a multi-language
	// site collection.BuildLanguageViews fills these once i18n fallback
	// copies exist, and they stay nil on a single-language site. Read them
	// through PagesFor, ListPages, FeaturedFor, IndexFor, TitleFor and
	// LabNavTree, which fall back to the shared fields.
	LangPages         map[string][]*Page             // Pages in one language, fallback copies included, sorted
	LangIndex         map[string]*Page               // the collection's root index page in one language
	LangFeatured      map[string][]*Page             // Featured in one language
	LabNavTreesByLang map[string]map[string]*NavTree // LabNavTrees in one language
}

// PagesFor returns the collection's pages in lang, section pages included,
// in the collection's sort order. Without a per-language view for lang, such
// as on a single-language site, it returns Pages.
func (c *Collection) PagesFor(lang string) []*Page {
	if c == nil {
		return nil
	}
	if pages, ok := c.LangPages[lang]; ok {
		return pages
	}
	return c.Pages
}

// ListPages returns the pages a list of the collection shows in lang:
// PagesFor without section pages.
func (c *Collection) ListPages(lang string) []*Page {
	all := c.PagesFor(lang)
	out := make([]*Page, 0, len(all))
	for _, p := range all {
		if p.Kind != KindSection {
			out = append(out, p)
		}
	}
	return out
}

// FeaturedFor returns the featured pages in lang, or Featured without a
// per-language view.
func (c *Collection) FeaturedFor(lang string) []*Page {
	if c == nil {
		return nil
	}
	if pages, ok := c.LangFeatured[lang]; ok {
		return pages
	}
	return c.Featured
}

// IndexFor returns the collection's root index page in lang, or IndexPage
// without a per-language view. It is nil when lang has no index page, such
// as an untranslated collection under i18n fallback "omit".
func (c *Collection) IndexFor(lang string) *Page {
	if c == nil {
		return nil
	}
	if idx, ok := c.LangIndex[lang]; ok {
		return idx
	}
	return c.IndexPage
}

// TitleFor returns the collection's title in lang: the title of its index
// page in that language, or Title.
func (c *Collection) TitleFor(lang string) string {
	if c == nil {
		return ""
	}
	if idx := c.LangIndex[lang]; idx != nil && idx.Title != "" {
		return idx.Title
	}
	return c.Title
}

// LabNavTree returns the sidebar tree of the lab at labPermalink in lang.
func (c *Collection) LabNavTree(lang, labPermalink string) *NavTree {
	if c == nil {
		return nil
	}
	if trees, ok := c.LabNavTreesByLang[lang]; ok {
		return trees[labPermalink]
	}
	return c.LabNavTrees[labPermalink]
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
