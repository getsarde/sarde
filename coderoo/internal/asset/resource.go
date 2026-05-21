package asset

import (
	"image"
	// Register standard image formats for DecodeConfig.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

// ResourceEnhancer populates Resource metadata for page bundles.
type ResourceEnhancer struct {
	DevMode bool
}

// EnhancePageResources populates MediaType, Title, RelPermalink, Width, and Height
// for all resources attached to a page bundle.
func (e *ResourceEnhancer) EnhancePageResources(page *engine.Page) error {
	if len(page.Resources) == 0 {
		return nil
	}

	pageDir := filepath.Dir(page.FilePath)

	for i := range page.Resources {
		res := &page.Resources[i]

		// Set Title from filename (without extension).
		ext := filepath.Ext(res.Name)
		if res.Title == "" {
			res.Title = strings.TrimSuffix(res.Name, ext)
		}

		// Set MediaType from extension.
		if res.MediaType == "" {
			res.MediaType = MediaTypeFromExt(ext)
		}

		// Set RelPermalink: page's permalink + resource name.
		if res.RelPermalink == "" {
			pageURL := page.RelPermalink
			if !strings.HasSuffix(pageURL, "/") {
				pageURL += "/"
			}
			res.RelPermalink = pageURL + res.Name
		}

		// Set absolute source path for image processing.
		if res.SrcPath == "" {
			res.SrcPath = filepath.Join(pageDir, res.Name)
		}

		// Read image dimensions (cheap — only decodes header).
		if IsImage(res.Name) && res.Width == 0 {
			w, h, err := imageSize(res.SrcPath)
			if err == nil {
				res.Width = w
				res.Height = h
			}
		}
	}

	return nil
}

// imageSize reads image dimensions without decoding the full image.
func imageSize(path string) (width, height int, err error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// GetResource returns the first resource matching the given name (exact match).
func GetResource(resources []engine.Resource, name string) *engine.Resource {
	for i := range resources {
		if resources[i].Name == name {
			return &resources[i]
		}
	}
	return nil
}

// MatchResources returns all resources whose Name matches the glob pattern.
func MatchResources(resources []engine.Resource, pattern string) []engine.Resource {
	var matched []engine.Resource
	for _, r := range resources {
		if ok, _ := filepath.Match(pattern, r.Name); ok {
			matched = append(matched, r)
		}
	}
	return matched
}

// ResourcesByType returns all resources whose MediaType starts with the given prefix.
// For example, ResourcesByType(resources, "image") matches "image/jpeg", "image/png", etc.
func ResourcesByType(resources []engine.Resource, mediaType string) []engine.Resource {
	var matched []engine.Resource
	prefix := mediaType
	if !strings.Contains(prefix, "/") {
		prefix += "/"
	}
	for _, r := range resources {
		if strings.HasPrefix(r.MediaType, prefix) {
			matched = append(matched, r)
		}
	}
	return matched
}
