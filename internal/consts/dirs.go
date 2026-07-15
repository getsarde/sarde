package consts

const (
	// Top-level project directories.
	DirContent = "content"
	DirLayouts = "layouts"
	DirStatic  = "static"
	DirThemes  = "themes"
	DirAssets  = "assets"
	DirData    = "data"

	// Layout subdirectory names (used in embedded theme and template overlay).
	DirDefault    = "_default"
	DirDocs       = "_docs"
	DirBlog       = "_blog"
	DirTaxonomy   = "_taxonomy"
	DirComponents = "components"
	DirPartials   = "partials"
	DirShortcodes = "shortcodes"

	// Internal Params keys shared between builder, template, and collection packages.
	PaginationCurrentKey = "__pagination_current"
	TaxonomyKey          = "__taxonomy"
	TaxonomyTermKey      = "__taxonomy_term"
	TermEntriesKey       = "__term_entries"
	CascadeKey           = "__cascade"

	// Pagination defaults.
	DefaultPaginateBy = 10

	// Collection type names.
	CollectionBlog    = "blog"
	CollectionDocs    = "docs"
	CollectionCourses = "courses"

	// Config filenames.
	FileSiteConfig    = "sarde.yaml"
	FileThemeConfig   = "theme.yaml"
	FileCollConfig    = "config.yaml"
	FileNavConfig     = "nav.yaml"
	FileSidebarConfig = "sidebar.yaml"

	// Template filenames.
	TemplateBaseOf = "baseof.html"
	Template404    = "404.html"

	// Server defaults.
	DefaultHost = "127.0.0.1"
	DefaultPort = 4727

	// Output lock file (see internal/buildlock).
	FileOutputLock = ".sarde.lock"
)
