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
	LayoutLabs         LayoutType = "labs"         // lab reader; sidebar + ToC + progress bar
)

var validLayouts = map[LayoutType]bool{
	LayoutDefault: true, LayoutDocs: true, LayoutSplash: true,
	LayoutWide: true, LayoutFull: true, LayoutCentered: true,
	LayoutSplit: true, LayoutPresentation: true, LayoutLabs: true,
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
	return layout == LayoutDocs || layout == LayoutWide || layout == LayoutLabs
}

// LayoutHasTOC returns true if the layout includes a table of contents.
func LayoutHasTOC(layout LayoutType) bool {
	return layout == LayoutDocs || layout == LayoutLabs
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
	Lang     string
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
	PublicFiles     int
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
	Title            string
	BaseURL          string
	BasePath         string // normalized: "/docs/" or "/"
	SiteID           string
	Language         string
	Generator        string
	Favicon          string
	FaviconType      string
	Logo             LogoContext
	SitemapEnabled   bool
	Config           any // *config.SiteConfig at runtime; any to avoid circular imports
	Collections      map[string]*Collection
	Taxonomies       map[string]*Taxonomy
	TaxonomiesByLang map[string]map[string]*Taxonomy
	Pages            []*Page
	Data             map[string]any
	BuildTime        time.Time
	Languages        []Language
	DefaultLang      string
	EditURL          string        // base URL for "edit this page" links (e.g. https://github.com/user/repo/edit/main/content)
	KazariScriptURL  string        // URL of the Kazari interaction JS file served globally on every page
	IconLicenses     []IconLicense // license metadata for loaded icon sets (for an attribution/credits page)
}

// LogoImage is one resolved logo variant. Width and Height are 0 for SVG logos
// and whenever the dimensions could not be probed, in which case the template
// omits the corresponding attributes.
type LogoImage struct {
	URL    string
	Width  int
	Height int
}

// LogoContext carries the resolved site logo into templates as .Site.Logo.
type LogoContext struct {
	Light         LogoImage
	Dark          LogoImage
	Alt           string
	ReplacesTitle bool
	// Single reports that one image serves both themes, so the template renders
	// a single <img> with no light/dark toggle classes.
	Single bool
}

// IconLicense is the license metadata of a loaded icon set, exposed to
// templates as .Site.IconLicenses so a theme/author can render a credits page.
type IconLicense struct {
	Prefix string
	Title  string
	SPDX   string
	URL    string
}

// Language represents a configured language for i18n.
type Language struct {
	Code   string
	Name   string
	Dir    string // "ltr" or "rtl"
	Weight int
}

// ---------------------------------------------------------------------------
// Versioning
// ---------------------------------------------------------------------------

// VersionConfig holds versioning settings for a collection (engine-level mirror
// of config.VersioningConfig to avoid import cycles).
type VersionConfig struct {
	Enabled                   bool
	LastVersion               string // version ID that serves the root URL (no prefix)
	PublishLatestAtVersionURL bool
	Versions                  []VersionDef
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
	StyleTag    template.HTML // pre-rendered <style> block with :root/:root[data-theme="dark"] tokens
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

// ThemeResolver handles template/asset overlay resolution. Priority order:
// user → theme → plugin → embedded.
type ThemeResolver struct {
	ProjectDir string   // root of user project (contains layouts/, themes/)
	ThemeName  string   // active theme name (for themes/<name>/layouts/)
	EmbeddedFS fs.FS    // compiled-in embedded/theme/ filesystem
	PluginDirs []string // templates/ dirs of active external plugins, sorted by slug
}
