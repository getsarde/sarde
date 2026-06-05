package embedded

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
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

// vendorFS contains vendored third-party ESM packages (e.g. Shiki).
//
//go:embed all:vendor
var vendorFS embed.FS

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

// VendorFS returns the embedded vendor filesystem rooted at "vendor/".
func VendorFS() fs.FS {
	sub, _ := fs.Sub(vendorFS, "vendor")
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

