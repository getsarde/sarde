package engine

import (
	"html/template"
	"io/fs"
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
)

// ---------------------------------------------------------------------------
// Layout Types
// ---------------------------------------------------------------------------

// LayoutType determines the page layout (column structure).
type LayoutType string

const (
	LayoutDefault  LayoutType = "default"  // single-column (blog, projects, standalone)
	LayoutDocs     LayoutType = "docs"     // three-column (sidebar | content | ToC)
	LayoutSplash   LayoutType = "splash"   // full-width, no sidebar or ToC (landing pages)
	LayoutWide     LayoutType = "wide"     // wider content with sidebar, no ToC
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
type Page struct {
	// Identity
	Title       string
	Slug        string
	Date        time.Time
	Updated     time.Time
	PublishDate time.Time
	Permalink    string
	RelPermalink string
	Kind         NodeKind
	FilePath     string
	RelPath      string // path relative to content dir (POSIX slashes); used by editURL template func

	// Content
	Content     template.HTML
	Summary     template.HTML
	RawContent  string
	WordCount   int
	ReadingTime int
	Headings    []Heading

	// Metadata
	Draft       bool
	Weight      int
	Description string
	Image       string

	// Relationships
	Collection *Collection
	Section    *Section
	PrevPage   *Page
	NextPage   *Page
	Siblings   []*Page
	Backlinks  []*Page

	// Navigation (docs-layout only)
	NavNode *NavNode

	// Taxonomy
	Tags       []string
	Categories []string
	Aliases    []string

	// Sidebar
	SidebarLabel  string
	SidebarHidden bool
	Badge         Badge

	// Bundle resources
	Resources []Resource

	// i18n
	Lang         string
	LangRelPath  string // relative path within language root (for translation matching)
	Translations []*Page
	IsFallback   bool

	// User data
	Params map[string]any
}

// ---------------------------------------------------------------------------
// Frontmatter
// ---------------------------------------------------------------------------

// Frontmatter represents parsed frontmatter fields from a content file.
type Frontmatter struct {
	Title         string            `yaml:"title"`
	Date          time.Time         `yaml:"date"`
	Updated       time.Time         `yaml:"updated"`
	Draft       bool      `yaml:"draft"`
	PublishDate time.Time `yaml:"publish_date"`
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
	Prev          string            `yaml:"prev"`
	Next          string            `yaml:"next"`
	EditURL       *bool             `yaml:"edit_url"`
	ShowUpdated   *bool             `yaml:"show_updated"`
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
	Text    string `yaml:"text"`
	Link    string `yaml:"link"`
	Variant string `yaml:"variant"`
	Icon    string `yaml:"icon"`
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
	Featured  []*Page             // subset of Pages with frontmatter `featured: true`
	Sections  []*Section
	NavTree   *NavTree            // default language nav tree (backward compat)
	NavTrees  map[string]*NavTree // per-language nav trees (i18n)
	IndexPage *Page
}

// CollectionConfig holds per-collection settings (auto-detected or explicit).
type CollectionConfig struct {
	SortBy    string
	SortOrder string
	Layout    LayoutType
	Permalink string
	Paginate  int
	Feed      bool
	Sidebar   *SidebarConfig
	TOC       *TOCConfig
	PrevNext  *PrevNextConfig
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
	Label     string
	URL       string
	Slug      string
	Weight    int
	Position  int
	Children  []*NavNode
	Parent    *NavNode
	Depth     int
	IsActive  bool
	IsOpen    bool
	HasActive bool
	Page      *Page
	Attrs     map[string]string
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
}

// ---------------------------------------------------------------------------
// Taxonomy
// ---------------------------------------------------------------------------

// Taxonomy represents a grouping dimension (tags, categories, authors, etc.).
type Taxonomy struct {
	Name      string
	Singular  string
	Terms     map[string]*TaxonomyTerm
	Permalink string
}

// TaxonomyTerm is a single term within a taxonomy with its associated pages.
type TaxonomyTerm struct {
	Name      string
	Slug      string
	Permalink string
	Pages     []*Page
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

// ---------------------------------------------------------------------------
// Build Result
// ---------------------------------------------------------------------------

// BuildResult holds the outcome of a site build.
type BuildResult struct {
	PageCount int
	Duration  time.Duration
	Warnings  []ValidationWarning
	OutputDir string
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
// RouteData
// ---------------------------------------------------------------------------

// RouteData is the unified context object passed to every template render.
type RouteData struct {
	Page         *Page
	Collection   *Collection
	GlobalNav    *GlobalNav
	Sidebar      *NavTree
	SidebarType  string
	Breadcrumbs  []BreadcrumbItem
	Pagination   *PaginationLinks
	Paginator    *Paginator // numbered list-page pagination (section/list pages only)
	HasSidebar   bool
	Section      *Section
	IsSection    bool
	Layout       LayoutType
	Template     string
	Site         *SiteContext
	Theme        *ThemeConfig
	Lang         string
	Dir          string
	Translations []TranslationLink
	Homepage     *HomepageData // only set for KindHome pages

	// Per-page asset injection (populated by plugins via BeforeRender).
	Scripts       []string        // root-relative script URLs (emitted as <script defer src>)
	Styles        []string        // root-relative stylesheet URLs (emitted as <link rel="stylesheet">)
	InlineScripts []template.HTML // inline <script> bodies (already-escaped template.HTML)
}

// HomepageData exposes homepage settings to templates.
type HomepageData struct {
	Template string
	Hero     HeroData
}

// HeroData holds hero section settings for the homepage.
type HeroData struct {
	Title      string
	Subtitle   string
	CTA        *HeroCTAData
	Background string
}

// HeroCTAData holds the call-to-action button settings.
type HeroCTAData struct {
	Label string
	URL   string
}

// TranslationLink points to the same page in another language.
type TranslationLink struct {
	Lang  string
	URL   string
	Title string
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
	Type      string   `yaml:"type"       json:"type"`       // "string", "int", "float", "bool", "date", "list", "enum"
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

