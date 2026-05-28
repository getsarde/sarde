package engine

import (
	"html/template"
	"io/fs"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Node Kinds
// ---------------------------------------------------------------------------

// NodeKind classifies content files discovered during the filesystem walk.
type NodeKind string

const (
	KindHome       NodeKind = "home"
	KindSection    NodeKind = "section"
	KindPage       NodeKind = "page"
	KindBundle     NodeKind = "bundle"
	KindStandalone NodeKind = "standalone"
	KindTaxonomy   NodeKind = "taxonomy"
	KindTerm       NodeKind = "term"
)

// ---------------------------------------------------------------------------
// Layout Types
// ---------------------------------------------------------------------------

// LayoutType determines the page layout (column structure).
type LayoutType string

const (
	LayoutDefault      LayoutType = "default"      // single-column (blog, projects, standalone)
	LayoutDocs         LayoutType = "docs"         // three-column (sidebar | content | ToC)
	LayoutSplash       LayoutType = "splash"       // full-width, no sidebar or ToC (landing pages)
	LayoutWide         LayoutType = "wide"         // wider content with sidebar, no ToC
	LayoutFull         LayoutType = "full"         // full-width, no sidebar or ToC
	LayoutCentered     LayoutType = "centered"     // narrow centered column, no sidebar
	LayoutSplit        LayoutType = "split"        // two equal columns; no sidebar, no ToC
	LayoutPresentation LayoutType = "presentation" // full-width slide-viewer; no sidebar, no ToC
)

var validLayouts = map[LayoutType]bool{
	LayoutDefault: true, LayoutDocs: true, LayoutSplash: true,
	LayoutWide: true, LayoutFull: true, LayoutCentered: true,
	LayoutSplit: true, LayoutPresentation: true,
}

// ValidateLayout returns true if the layout type is recognized.
func ValidateLayout(layout LayoutType) bool { return validLayouts[layout] }

// ResolveLayout converts a string to a validated LayoutType, falling back to LayoutDefault.
func ResolveLayout(s string) LayoutType {
	lt := LayoutType(s)
	if validLayouts[lt] {
		return lt
	}
	return LayoutDefault
}

// LayoutHasSidebar returns true if the layout includes a sidebar.
func LayoutHasSidebar(layout LayoutType) bool {
	return layout == LayoutDocs || layout == LayoutWide
}

// LayoutHasTOC returns true if the layout includes a table of contents.
func LayoutHasTOC(layout LayoutType) bool {
	return layout == LayoutDocs
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

// Page represents a single content page after parsing and transformation.
// ---------------------------------------------------------------------------
// Page — embedded sub-structs
// ---------------------------------------------------------------------------

// PageIdentity holds the core identity fields of a page.
type PageIdentity struct {
	Title        string
	Slug         string
	Date         time.Time
	Updated      time.Time
	PublishDate  time.Time
	ExpiryDate   time.Time
	Permalink    string
	RelPermalink string
	Kind         NodeKind
	FilePath     string
	RelPath      string
}

// PageContent holds rendered content and content-derived metadata.
type PageContent struct {
	Content       template.HTML
	Summary       template.HTML
	RawContent    string
	WordCount     int
	ReadingTime   int
	Headings      []Heading
	HasCodeBlocks bool
	HasImages     bool
}

// PageMeta holds editorial metadata.
type PageMeta struct {
	Draft       bool
	Weight      int
	Description string
	Image       string
}

// PageRelationships holds graph connections to other pages and structures.
type PageRelationships struct {
	Collection *Collection
	Section    *Section
	PrevPage   *Page
	NextPage   *Page
	Siblings   []*Page
	Backlinks  []*Page
}

// PageTaxonomy holds taxonomy membership fields.
type PageTaxonomy struct {
	Tags       []string
	Categories []string
	Aliases    []string
}

// PageSidebar holds sidebar presentation fields.
type PageSidebar struct {
	SidebarLabel  string
	SidebarHidden bool
	Badge         Badge
}

// PageI18n holds language and translation fields.
type PageI18n struct {
	Lang         string
	LangRelPath  string
	Translations []*Page
	IsFallback   bool
}

// PageVersioning holds version membership fields.
type PageVersioning struct {
	Version        string
	VersionRelPath string
	VersionPeers   []*Page
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

// Page represents a single content page. Sub-structs are embedded so all
// fields remain accessible as top-level names (e.g. page.Title, page.Tags).
type Page struct {
	PageIdentity
	PageContent
	PageMeta
	PageRelationships
	PageTaxonomy
	PageSidebar
	PageI18n
	PageVersioning

	NavNode   *NavNode
	Resources []Resource
	Params    map[string]any
}

// ---------------------------------------------------------------------------
// Frontmatter
// ---------------------------------------------------------------------------

// Frontmatter represents parsed frontmatter fields from a content file.
type Frontmatter struct {
	Title         string            `yaml:"title"`
	Date          time.Time         `yaml:"date"`
	Updated       time.Time         `yaml:"updated"`
	Draft         bool              `yaml:"draft"`
	PublishDate   time.Time         `yaml:"publish_date"`
	ExpiryDate    time.Time         `yaml:"expiry_date"`
	Slug          string            `yaml:"slug"`
	Summary       string            `yaml:"summary"`
	Template      string            `yaml:"template"`
	Tags          []string          `yaml:"tags"`
	Categories    []string          `yaml:"categories"`
	Layout        string            `yaml:"layout"`
	Type          string            `yaml:"type"`
	Weight        int               `yaml:"weight"`
	Description   string            `yaml:"description"`
	Image         string            `yaml:"image"`
	Aliases       []string          `yaml:"aliases"`
	Transparent   bool              `yaml:"transparent"`
	Render        *bool             `yaml:"render"`
	Hero          *HeroConfig       `yaml:"hero"`
	Pagefind      *bool             `yaml:"pagefind"`
	SidebarLabel  string            `yaml:"sidebar_label"`
	SidebarHidden bool              `yaml:"sidebar_hidden"`
	SidebarGroup  string            `yaml:"sidebar_group"`
	SidebarAttrs  map[string]string `yaml:"sidebar_attrs"`
	Badge         Badge             `yaml:"badge"`
	TOC           *bool             `yaml:"toc"`
	TOCMinLevel   int               `yaml:"toc_min_level"`
	TOCMaxLevel   int               `yaml:"toc_max_level"`
	Prev          *NavOverride      `yaml:"prev"`
	Next          *NavOverride      `yaml:"next"`
	EditURL       *EditURLValue     `yaml:"edit_url"`
	ShowUpdated   *bool             `yaml:"show_updated"`
	Icon          string            `yaml:"icon"`
	Head          []HeadTag         `yaml:"head"`
	Banner        *PageBanner       `yaml:"banner"`
	Cascade       map[string]any    `yaml:"cascade"`
	Params        map[string]any    `yaml:"params"`
}

// HeroConfig defines hero section fields for splash layout pages.
type HeroConfig struct {
	Title   string       `yaml:"title"`
	Tagline string       `yaml:"tagline"`
	Image   *HeroImage   `yaml:"image"`
	Actions []HeroAction `yaml:"actions"`
}

// HeroImage defines the hero image with optional light/dark variants.
type HeroImage struct {
	Src   string `yaml:"src"`
	Light string `yaml:"light"`
	Dark  string `yaml:"dark"`
	Alt   string `yaml:"alt"`
}

// HeroAction defines a call-to-action button in the hero section.
type HeroAction struct {
	Text    string            `yaml:"text"`
	Link    string            `yaml:"link"`
	Variant string            `yaml:"variant"`
	Icon    string            `yaml:"icon"`
	Attrs   map[string]string `yaml:"attrs"`
}

// SanitizeAttrs strips event-handler attributes (on*) from all hero actions.
func (h *HeroConfig) SanitizeAttrs() {
	for i := range h.Actions {
		if len(h.Actions[i].Attrs) == 0 {
			continue
		}
		clean := make(map[string]string, len(h.Actions[i].Attrs))
		for k, v := range h.Actions[i].Attrs {
			if !strings.HasPrefix(strings.ToLower(k), "on") {
				clean[k] = v
			}
		}
		h.Actions[i].Attrs = clean
	}
}

// HeadTag defines a single injected <head> element from frontmatter.
type HeadTag struct {
	Tag     string            `yaml:"tag"`
	Attrs   map[string]string `yaml:"attrs"`
	Content string            `yaml:"content"`
}

// AllowedHeadTags lists the HTML tag names permitted in per-page head injection.
var AllowedHeadTags = map[string]bool{
	"meta": true, "link": true, "script": true,
	"style": true, "noscript": true, "base": true,
}

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
	Versioning      *VersionConfig
	VersionNavTrees map[string]*NavTree // per-version nav trees, key = version ID ("" for unversioned)
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
}

// SidebarConfig controls sidebar behavior for docs-layout collections.
type SidebarConfig struct {
	Collapsible        bool
	CollapsedByDefault bool
	MaxDepth           int
	Search             bool
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
	Weight      int
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

// ---------------------------------------------------------------------------
// Navigation
// ---------------------------------------------------------------------------

// NavTree represents a complete sidebar navigation tree for a collection.
type NavTree struct {
	Root       *NavNode
	Flat       []*NavNode
	TotalPages int
	MaxDepth   int
}

// NavNode is a single entry in the sidebar navigation tree.
type NavNode struct {
	Label       string
	URL         string
	Slug        string
	Weight      int
	Position    int
	Children    []*NavNode
	Parent      *NavNode
	Depth       int
	IsActive    bool
	IsOpen      bool
	HasActive   bool
	DefaultOpen bool
	Page        *Page
	Attrs       map[string]string
}

// GlobalNav represents the top-level site navigation bar.
type GlobalNav struct {
	Items []GlobalNavItem
}

// GlobalNavItem is a single entry in the global navigation bar.
type GlobalNavItem struct {
	Label      string
	URL        string
	Collection string
	IsActive   bool
	External   bool
}

// BreadcrumbItem is a single entry in a breadcrumb trail.
type BreadcrumbItem struct {
	Label   string
	URL     string
	Current bool
}

// PaginationLinks holds prev/next page references.
type PaginationLinks struct {
	Prev *PaginationLink
	Next *PaginationLink
}

// PaginationLink is a reference to a prev or next page.
type PaginationLink struct {
	URL   string
	Title string
}

// Paginator holds numbered list-page pagination state for collection index pages.
type Paginator struct {
	Pages        []PaginationLink // Numbered links (one per page of results)
	CurrentPages []*Page          // Slice of content pages visible on this pagination page
	Current      int              // 1-based index of the current page
	Total        int              // Total number of pagination pages
	HasPrev      bool
	HasNext      bool
	PrevURL      string
	NextURL      string
	TotalItems   int    // Total content items across all pagers
	BaseURL      string // Collection base URL for constructing custom pagination links
	FirstURL     string // Permalink to the first pagination page
	LastURL      string // Permalink to the last pagination page
}

// ---------------------------------------------------------------------------
// Taxonomy
// ---------------------------------------------------------------------------

// Taxonomy represents a grouping dimension (tags, categories, authors, etc.).
type Taxonomy struct {
	Name       string
	Singular   string
	Terms      map[string]*TaxonomyTerm
	Permalink  string
	PaginateBy int // 0 = no pagination for term listing pages
}

// TaxonomyTerm is a single term within a taxonomy with its associated pages.
type TaxonomyTerm struct {
	Name        string
	Slug        string
	Permalink   string
	Pages       []*Page
	Label       string
	Description string
	Color       string
	Icon        string
	Hidden      bool
	Priority    int
}

// TermEntry wraps a TaxonomyTerm with computed tag-cloud data.
type TermEntry struct {
	*TaxonomyTerm
	Count   int
	PopTier int // 1-5 popularity quintile
}

// ---------------------------------------------------------------------------
// Resource
// ---------------------------------------------------------------------------

// Resource represents a page-bundled asset (image, file, etc.).
type Resource struct {
	Name         string
	Title        string
	MediaType    string
	RelPermalink string
	Width        int
	Height       int
	SrcPath      string // absolute filesystem path for image processing
}

// ---------------------------------------------------------------------------
// Heading
// ---------------------------------------------------------------------------

// Heading represents a heading extracted from content for ToC generation.
type Heading struct {
	Level int
	ID    string
	Text  string
}

// CollectedLink represents a link found during markdown rendering.
type CollectedLink struct {
	Href    string
	IsImage bool
}

// ValidationEntry holds collected links for a single page, used by the link validator.
type ValidationEntry struct {
	Links    []CollectedLink
	FilePath string
}

// RenderResult holds the output of a markdown-to-HTML conversion.
type RenderResult struct {
	HTML          string
	Headings      []Heading
	HasCodeBlocks bool
	HasImages     bool
	Links         []CollectedLink
}

// ---------------------------------------------------------------------------
// Build Result
// ---------------------------------------------------------------------------

// BuildResult holds the outcome of a site build.
type BuildResult struct {
	PageCount int
	Duration  time.Duration
	Warnings  []ValidationWarning
	OutputDir string

	// Summary stats (populated by full Build(), zero for incremental ContentRebuild).
	PaginatorPages  int
	Collections     int
	BundleAssets    int
	StaticFiles     int
	ProcessedImages int
	AliasCount      int
	SitemapCount    int

	// Build logging.
	LogMessages  []BuildLogEntry
	PhaseTimings []PhaseTiming
}

// PhaseTiming records the duration of a single build pipeline phase.
type PhaseTiming struct {
	Phase    string
	Duration time.Duration
}

// BuildLogEntry is a single log message emitted during the build.
type BuildLogEntry struct {
	Source  string // e.g. "sitemap", "search", "social-cards"
	Message string
}

// ValidationWarning represents a non-fatal issue found during frontmatter validation.
type ValidationWarning struct {
	File    string
	Field   string
	Message string
	Level   string
}

// ---------------------------------------------------------------------------
// Site Context
// ---------------------------------------------------------------------------

// SiteContext provides global site data accessible in every template.
type SiteContext struct {
	Title       string
	BaseURL     string
	Language    string
	Config      any // *config.SiteConfig at runtime; any to avoid circular imports
	Collections map[string]*Collection
	Taxonomies  map[string]*Taxonomy
	Pages       []*Page
	Data        map[string]any
	BuildTime   time.Time
	Languages   []Language
	DefaultLang string
	EditURL     string // base URL for "edit this page" links (e.g. https://github.com/user/repo/edit/main/content)
}

// Language represents a configured language for i18n.
type Language struct {
	Code   string
	Name   string
	Dir    string // "ltr" or "rtl"
	Weight int
}

// ---------------------------------------------------------------------------
// RouteData — embedded sub-structs
// ---------------------------------------------------------------------------

// RouteNav groups navigation-related fields for the current page render.
type RouteNav struct {
	GlobalNav                 *GlobalNav
	Sidebar                   *NavTree
	SidebarType               string
	Breadcrumbs               []BreadcrumbItem
	Pagination                *PaginationLinks
	Paginator                 *Paginator
	HasSidebar                bool
	SidebarCollapsedByDefault bool
	Section                   *Section
	IsSection                 bool
}

// RouteI18n groups language and translation fields.
type RouteI18n struct {
	Lang         string
	Dir          string
	Translations []TranslationLink
}

// RouteVersioning groups version-switcher fields for versioned collections.
type RouteVersioning struct {
	Version       string
	VersionLabel  string
	Versions      []VersionLink
	IsLatest      bool
	VersionBanner string
}

// RouteTabs groups docs-tab fields for tabbed collections.
type RouteTabs struct {
	IsTabbed  bool
	DocsTabs  []*DocsTab
	ActiveTab *DocsTab
}

// RouteAssets holds per-page asset URLs injected by plugins via BeforeRender.
type RouteAssets struct {
	Scripts       []string
	Styles        []string
	InlineScripts []template.HTML
	ModuleScripts []string
}

// ---------------------------------------------------------------------------
// RouteData
// ---------------------------------------------------------------------------

// RouteData is the unified context object passed to every template render.
// Sub-structs are embedded so all fields remain accessible as top-level
// names in both Go code and html/template (e.g. .Lang, .Scripts, .Version).
type RouteData struct {
	Page       *Page
	Collection *Collection
	Site       *SiteContext
	Theme      *ThemeConfig
	Layout     LayoutType
	Template   string

	RouteNav
	RouteI18n
	RouteVersioning
	RouteTabs
	RouteAssets

	Homepage     *HomepageData
	Taxonomy     *Taxonomy
	TaxonomyTerm *TaxonomyTerm
	TermEntries  []*TermEntry
	PageBanner   *PageBanner
}

// HomepageData exposes homepage settings to templates.
type HomepageData struct {
	Template string
	Hero     HeroData
}

// HeroData holds hero section settings for the homepage.
type HeroData struct {
	Eyebrow      string
	Title        string
	Subtitle     string
	CTA          *HeroCTAData
	SecondaryCTA *HeroCTAData
	Stats        []HeroStatData
	Code         *HeroCodeData
	Background   string
}

// HeroCTAData holds the call-to-action button settings.
type HeroCTAData struct {
	Label string
	URL   string
}

// HeroStatData holds a short proof point for the homepage hero.
type HeroStatData struct {
	Value string
	Label string
}

// HeroCodeData holds the optional code sample shown in the homepage hero.
type HeroCodeData struct {
	Title    string
	Language string
	Body     string
}

// TranslationLink points to the same page in another language.
type TranslationLink struct {
	Lang       string
	Name       string // display name (e.g. "Français"), falls back to Lang code
	Dir        string // "ltr" or "rtl"
	URL        string
	Title      string
	IsFallback bool
}

// ---------------------------------------------------------------------------
// Versioning
// ---------------------------------------------------------------------------

// VersionConfig holds versioning settings for a collection (engine-level mirror
// of config.VersioningConfig to avoid import cycles).
type VersionConfig struct {
	Enabled     bool
	LastVersion string // version ID that serves the root URL (no prefix)
	Versions    []VersionDef
}

// VersionDef describes one version of a versioned docs collection.
type VersionDef struct {
	ID       string
	Label    string
	Path     string // URL path segment (defaults to ID)
	Banner   string // "none" / "unmaintained" / "unreleased"
	Redirect string // "same-page" / "root"
}

// VersionLink points to the same page in another version (mirrors TranslationLink).
type VersionLink struct {
	ID        string
	Label     string
	URL       string // target URL (peer page or version root, based on redirect strategy)
	Title     string
	IsCurrent bool
	IsLatest  bool   // true if this is the last_version
	Banner    string // "none" / "unmaintained" / "unreleased"
	Redirect  string // "same-page" / "root"
}

// ThemeConfig holds metadata, token values, and pre-rendered CSS for the active theme.
type ThemeConfig struct {
	Name        string
	Slug        string
	Version     string
	Author      string
	Tokens      map[string]string
	DarkTokens  map[string]string
	DarkEnabled bool
	StyleTag    template.HTML // pre-rendered <style> block with :root/:root.dark tokens
}

// ---------------------------------------------------------------------------
// Placeholder types — fully defined in later phases
// ---------------------------------------------------------------------------

// FrontmatterSchema defines the expected frontmatter fields for a collection.
type FrontmatterSchema struct {
	Fields map[string]FieldDef `yaml:"fields" json:"fields"`
}

// FieldDef describes a single frontmatter field for validation and editor UI.
type FieldDef struct {
	Type      string   `yaml:"type"       json:"type"` // "string", "int", "float", "bool", "date", "list", "enum"
	Label     string   `yaml:"label"      json:"label,omitempty"`
	Required  bool     `yaml:"required"   json:"required,omitempty"`
	Default   any      `yaml:"default"    json:"default,omitempty"`
	Min       *float64 `yaml:"min"        json:"min,omitempty"`
	Max       *float64 `yaml:"max"        json:"max,omitempty"`
	MaxLength *int     `yaml:"max_length" json:"maxLength,omitempty"`
	Options   []string `yaml:"options"    json:"options,omitempty"` // for enum type
}

// ThemeResolver handles three-layer template/asset resolution (user → theme → embedded).
type ThemeResolver struct {
	ProjectDir string // root of user project (contains layouts/, themes/)
	ThemeName  string // active theme name (for themes/<name>/layouts/)
	EmbeddedFS fs.FS  // compiled-in embedded/theme/ filesystem
}
