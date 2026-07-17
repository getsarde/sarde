package engine

import "html/template"

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
	Lang            string
	Dir             string
	Translations    []TranslationLink
	AllTranslations []TranslationLink
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

// RouteLabs groups lab-collection fields (progress, numbering, objectives).
type RouteLabs struct {
	LabNumber          int
	LabStepIndex       int
	LabStepTotal       int
	LabStepLabel       string
	LearningObjectives []string
}

// RouteAssets holds per-page asset URLs injected by plugins via BeforeRender.
type RouteAssets struct {
	Scripts       []string
	Styles        []string
	InlineScripts []template.JS
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
	RouteLabs
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
	Image        *HeroImageData
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

// HeroImageData holds the optional hero image/SVG for the homepage hero panel.
type HeroImageData struct {
	Src   string
	Light string
	Dark  string
	Alt   string
	HTML  template.HTML
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
