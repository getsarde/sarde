package template

import (
	htmltemplate "html/template"

	"github.com/getsarde/sarde/internal/asset"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
)

// ShortcodeFuncMapConfig holds the dependencies needed to build a FuncMap
// for shortcode templates. Component registry, plugin funcs, and i18n are
// intentionally excluded to avoid concurrency issues during parallel
// markdown rendering.
type ShortcodeFuncMapConfig struct {
	Site           **engine.SiteContext
	Resolver       *engine.ThemeResolver
	AssetResolver  *asset.Resolver
	AssetManifest  *asset.Manifest
	ImageProcessor *asset.ImageProcessor
	PageIndex      **content.PageIndex
}

// BuildShortcodeFuncMap constructs a FuncMap suitable for shortcode templates.
// It creates a temporary Engine wired to the caller's pointers so closures
// see mutations made through those pointers during the build.
func BuildShortcodeFuncMap(cfg ShortcodeFuncMapConfig) htmltemplate.FuncMap {
	e := &Engine{
		resolver:       cfg.Resolver,
		site:           *cfg.Site,
		pageIndex:      *cfg.PageIndex,
		assetResolver:  cfg.AssetResolver,
		assetManifest:  cfg.AssetManifest,
		imageProcessor: cfg.ImageProcessor,
	}
	return e.buildFuncMap(nil, nil)
}
