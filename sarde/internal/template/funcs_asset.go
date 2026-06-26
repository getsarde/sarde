package template

import (
	htmltemplate "html/template"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/asset"
	"github.com/getsarde/sarde/internal/engine"
)

func currentAssetResolver(ptr **asset.Resolver) *asset.Resolver {
	if ptr == nil {
		return nil
	}
	return *ptr
}

func currentAssetManifest(ptr **asset.Manifest) *asset.Manifest {
	if ptr == nil {
		return nil
	}
	return *ptr
}

func currentImageProcessor(ptr **asset.ImageProcessor) *asset.ImageProcessor {
	if ptr == nil {
		return nil
	}
	return *ptr
}

func buildAssetFuncs(
	assetResolverPtr **asset.Resolver,
	assetManifestPtr **asset.Manifest,
	imageProcessorPtr **asset.ImageProcessor,
	urlResolverPtr **engine.URLResolver,
) htmltemplate.FuncMap {
	return htmltemplate.FuncMap{
		"asset": func(path string) string {
			assetResolver := currentAssetResolver(assetResolverPtr)
			if assetResolver == nil {
				return path
			}
			resolved, err := assetResolver.Resolve(path)
			if err != nil {
				return path
			}
			url := "/assets/" + path
			_ = resolved
			if r := *urlResolverPtr; r != nil {
				url = r.URL(url, "", "")
			}
			return url
		},
		"fingerprint": func(path string) string {
			assetManifest := currentAssetManifest(assetManifestPtr)
			if assetManifest == nil {
				return path
			}
			entry, ok := assetManifest.Lookup(path)
			if !ok {
				return path
			}
			url := entry.OutputURL
			if r := *urlResolverPtr; r != nil {
				url = r.URL(url, "", "")
			}
			return url
		},
		"inline": func(path string) htmltemplate.HTML {
			assetResolver := currentAssetResolver(assetResolverPtr)
			if assetResolver == nil {
				return ""
			}
			data, err := assetResolver.ResolveContent(path)
			if err != nil {
				return ""
			}
			ext := strings.ToLower(filepath.Ext(path))
			switch ext {
			case ".css":
				return htmltemplate.HTML("<style>" + string(data) + "</style>")
			case ".js":
				return htmltemplate.HTML("<script>" + string(data) + "</script>")
			default:
				return htmltemplate.HTML(string(data))
			}
		},
		"getResource": func(resources []engine.Resource, name string) *engine.Resource {
			return asset.GetResource(resources, name)
		},
		"matchResources": func(resources []engine.Resource, pattern string) []engine.Resource {
			return asset.MatchResources(resources, pattern)
		},
		"resourcesByType": func(resources []engine.Resource, mediaType string) []engine.Resource {
			return asset.ResourcesByType(resources, mediaType)
		},
		"image": func(res engine.Resource) htmltemplate.HTML {
			return htmltemplate.HTML(asset.RenderPicture(
				res.RelPermalink, res.Title,
				res.Width, res.Height,
				nil, "", true,
			))
		},
		"resize_image": func(res engine.Resource, params string) htmltemplate.HTML {
			imageProcessor := currentImageProcessor(imageProcessorPtr)
			if imageProcessor == nil || res.SrcPath == "" {
				return htmltemplate.HTML(asset.RenderPicture(
					res.RelPermalink, res.Title,
					res.Width, res.Height,
					nil, "", true,
				))
			}
			opts := asset.ParseImageOptionsFromQuery(params)
			variants, lqip, err := imageProcessor.ProcessImage(res.SrcPath, opts)
			if err != nil {
				return htmltemplate.HTML(asset.RenderPicture(
					res.RelPermalink, res.Title,
					res.Width, res.Height,
					nil, "", true,
				))
			}
			return htmltemplate.HTML(asset.RenderPicture(
				res.RelPermalink, res.Title,
				res.Width, res.Height,
				variants, lqip, true,
			))
		},
	}
}
