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
	DirComponents = "components"
	DirPartials   = "partials"

	// Collection type names.
	CollectionBlog    = "blog"
	CollectionDocs    = "docs"
	CollectionCourses = "courses"

	// Config filenames.
	FileSiteConfig  = "site.yaml"
	FileThemeConfig = "theme.yaml"
	FileCollConfig  = "config.yaml"
	FileNavConfig   = "nav.yaml"

	// Template filenames.
	TemplateBaseOf = "baseof.html"
	Template404    = "404.html"

	// Server defaults.
	DefaultHost = "127.0.0.1"
	DefaultPort = 4727
)
