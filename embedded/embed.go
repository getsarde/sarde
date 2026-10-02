package embedded

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"sort"
)

// DefaultsYAML contains the embedded default site configuration.
//
//go:embed defaults/sarde.yaml
var DefaultsYAML []byte

// LiveReloadJS contains the embedded live reload client script.
//
//go:embed livereload/livereload.js
var LiveReloadJS []byte

// i18nFS contains embedded default i18n translation strings.
//
//go:embed all:i18n
var i18nFS embed.FS

// themeFS contains all embedded theme templates and components.
//
//go:embed all:theme
var themeFS embed.FS

// ScaffoldHeroLight contains the light-mode hero SVG for sarde new site.
//
//go:embed scaffold/hero-light.svg
var ScaffoldHeroLight []byte

// ScaffoldHeroDark contains the dark-mode hero SVG for sarde new site.
//
//go:embed scaffold/hero-dark.svg
var ScaffoldHeroDark []byte

// templatesFS contains the site templates for sarde new site --template,
// one directory per template name.
//
//go:embed all:templates
var templatesFS embed.FS

// I18nFS returns the embedded i18n filesystem rooted at "i18n/".
func I18nFS() fs.FS {
	sub, _ := fs.Sub(i18nFS, "i18n")
	return sub
}

// ThemeFS returns the embedded theme filesystem rooted at "theme/".
func ThemeFS() fs.FS {
	sub, _ := fs.Sub(themeFS, "theme")
	return sub
}

// ThemeDirFS returns a live disk-backed fs.FS rooted at the given directory.
// Used by 'sarde dev --theme-dev' for live-reload of theme assets during
// framework development. In production, ThemeFS() is used instead.
func ThemeDirFS(dir string) (fs.FS, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("theme dev dir not accessible: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("theme dev path is not a directory: %s", dir)
	}
	if _, err := os.Stat(dir + "/css"); err != nil {
		return nil, fmt.Errorf("theme dev dir missing css/ subdirectory — did you point to embedded/theme/? path: %s", dir)
	}
	return os.DirFS(dir), nil
}

// SiteTemplate returns the files of the named site template, rooted at the
// template's directory, and false when no such template exists.
func SiteTemplate(name string) (fs.FS, bool) {
	if name == "" || !fs.ValidPath(name) {
		return nil, false
	}
	info, err := fs.Stat(templatesFS, "templates/"+name)
	if err != nil || !info.IsDir() {
		return nil, false
	}
	sub, err := fs.Sub(templatesFS, "templates/"+name)
	if err != nil {
		return nil, false
	}
	return sub, true
}

// SiteTemplateNames returns the names of the embedded site templates, sorted.
func SiteTemplateNames() []string {
	entries, _ := fs.ReadDir(templatesFS, "templates")
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}
