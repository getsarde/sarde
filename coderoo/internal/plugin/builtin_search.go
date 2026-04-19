package plugin

import (
	"embed"
	"encoding/json"
	"io/fs"
	"regexp"
	"strings"
)

//go:embed all:search_assets
var searchAssetsFS embed.FS

var searchRuntimeScripts = []string{
	"/assets/vendor/orama/orama.esm.js",
	"/assets/js/static-search.js",
}

func newSearchPlugin(cfg map[string]any) *Plugin {
	return &Plugin{
		Name: "search",
		Hooks: PluginHooks{
			BeforeRender: func(ctx *BeforeRenderContext) error {
				for _, s := range searchRuntimeScripts {
					ctx.RouteData.Scripts = appendUniqueScript(ctx.RouteData.Scripts, s)
				}
				return nil
			},
			BuildDone: func(ctx *BuildDoneContext) error {
				return searchBuildDone(ctx, cfg)
			},
		},
	}
}

type searchDocument struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Description string   `json:"description,omitempty"`
	Content     string   `json:"content,omitempty"`
	Section     string   `json:"section,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

func searchBuildDone(ctx *BuildDoneContext, cfg map[string]any) error {
	maxLen := cfgInt(cfg, "max_content_length", 5000)
	excludePatterns := cfgStringSlice(cfg, "exclude")

	var docs []searchDocument
	for _, page := range ctx.Pages {
		if page.Draft {
			continue
		}
		if shouldExclude(page.RelPermalink, excludePatterns) {
			continue
		}

		content := stripHTML(string(page.Content))
		if len(content) > maxLen {
			content = content[:maxLen]
		}

		section := ""
		if page.Collection != nil {
			section = page.Collection.Name
		}

		docs = append(docs, searchDocument{
			ID:          page.RelPermalink,
			Title:       page.Title,
			URL:         page.RelPermalink,
			Description: page.Description,
			Content:     content,
			Section:     section,
			Tags:        page.Tags,
		})
	}

	data, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return err
	}
	if err := ctx.WriteFile("search-index.json", data); err != nil {
		return err
	}

	return writeSearchAssets(ctx)
}

func writeSearchAssets(ctx *BuildDoneContext) error {
	return fs.WalkDir(searchAssetsFS, "search_assets", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(searchAssetsFS, path)
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(path, "search_assets/")
		var dest string
		switch rel {
		case "orama.esm.js":
			dest = "assets/vendor/orama/orama.esm.js"
		case "static-search.js":
			dest = "assets/js/static-search.js"
		default:
			dest = "assets/js/" + rel
		}
		return ctx.WriteFile(dest, data)
	})
}

func appendUniqueScript(list []string, item string) []string {
	for _, existing := range list {
		if existing == item {
			return list
		}
	}
	return append(list, item)
}

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// stripHTML removes HTML tags and collapses whitespace.
func stripHTML(s string) string {
	s = htmlTagRegex.ReplaceAllString(s, " ")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}
