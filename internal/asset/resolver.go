package asset

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Resolver implements 3-layer asset lookup: user assets/ → theme assets/ → embedded.
type Resolver struct {
	ProjectDir string
	ThemeName  string
	EmbeddedFS fs.FS
}

// Resolve returns the absolute filesystem path for a named asset.
// Lookup order:
//  1. {ProjectDir}/assets/{path}
//  2. {ProjectDir}/themes/{ThemeName}/assets/{path}
//  3. embedded FS assets/{path}
//
// For embedded assets, returns "embedded:{path}" since there is no filesystem path.
func (r *Resolver) Resolve(assetPath string) (string, error) {
	// Layer 1: user assets/
	userPath := filepath.Join(r.ProjectDir, "assets", filepath.FromSlash(assetPath))
	if _, err := os.Stat(userPath); err == nil {
		return userPath, nil
	}

	// Layer 2: theme assets/
	if r.ThemeName != "" {
		themePath := filepath.Join(r.ProjectDir, "themes", r.ThemeName, "assets", filepath.FromSlash(assetPath))
		if _, err := os.Stat(themePath); err == nil {
			return themePath, nil
		}
	}

	// Layer 3: embedded FS
	if r.EmbeddedFS != nil {
		fsPath := "assets/" + assetPath
		if _, err := fs.Stat(r.EmbeddedFS, fsPath); err == nil {
			return "embedded:" + assetPath, nil
		}
	}

	return "", fmt.Errorf("asset not found: %s", assetPath)
}

// ResolveContent returns the file contents for a named asset.
// Uses the same 3-layer lookup as Resolve.
func (r *Resolver) ResolveContent(assetPath string) ([]byte, error) {
	// Layer 1: user assets/
	userPath := filepath.Join(r.ProjectDir, "assets", filepath.FromSlash(assetPath))
	if data, err := os.ReadFile(userPath); err == nil {
		return data, nil
	}

	// Layer 2: theme assets/
	if r.ThemeName != "" {
		themePath := filepath.Join(r.ProjectDir, "themes", r.ThemeName, "assets", filepath.FromSlash(assetPath))
		if data, err := os.ReadFile(themePath); err == nil {
			return data, nil
		}
	}

	// Layer 3: embedded FS
	if r.EmbeddedFS != nil {
		fsPath := "assets/" + assetPath
		if data, err := fs.ReadFile(r.EmbeddedFS, fsPath); err == nil {
			return data, nil
		}
	}

	return nil, fmt.Errorf("asset not found: %s", assetPath)
}

// MediaTypeFromExt returns the MIME type for a file extension.
// The extension should include the leading dot (e.g., ".jpg").
func MediaTypeFromExt(ext string) string {
	switch strings.ToLower(ext) {
	// Images
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".avif":
		return "image/avif"
	case ".ico":
		return "image/x-icon"
	case ".bmp":
		return "image/bmp"
	// CSS/JS
	case ".css":
		return "text/css"
	case ".js", ".mjs":
		return "application/javascript"
	// Fonts
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".otf":
		return "font/otf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	// Documents
	case ".pdf":
		return "application/pdf"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".yaml", ".yml":
		return "application/yaml"
	// Video/Audio
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".ogg":
		return "audio/ogg"
	default:
		return "application/octet-stream"
	}
}

// IsImage returns true if the file extension indicates an image format.
func IsImage(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".svg", ".webp", ".avif", ".bmp", ".ico":
		return true
	}
	return false
}
