package plugin

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed all:search_assets
var searchAssetsFS embed.FS

var searchRuntimeScripts = []string{
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
	Version     string   `json:"version,omitempty"`
	Lang        string   `json:"-"`
}

func searchBuildDone(ctx *BuildDoneContext, cfg map[string]any) error {
	maxLen := cfgInt(cfg, "max_content_length", 5000)
	excludePatterns := cfgStringSlice(cfg, "exclude")

	var docs []searchDocument
	seen := make(map[string]bool)
	for _, page := range ctx.Pages {
		if page.Draft {
			continue
		}
		if shouldExclude(page.Permalink, excludePatterns) {
			continue
		}
		url := page.URL()
		if seen[url] {
			continue
		}
		seen[url] = true

		raw := string(page.Content)
		if cutoff := maxLen * 3; len(raw) > cutoff {
			raw = raw[:cutoff]
		}
		content := stripHTML(raw)
		if len(content) > maxLen {
			content = content[:maxLen]
		}

		section := ""
		if page.Collection != nil {
			section = page.Collection.Name
		}

		docs = append(docs, searchDocument{
			ID:          url,
			Title:       page.Title,
			URL:         url,
			Description: page.Description,
			Content:     content,
			Section:     section,
			Tags:        page.Tags,
			Version:     page.Version,
			Lang:        page.Lang,
		})
	}

	byLang := make(map[string][]searchDocument)
	for _, d := range docs {
		lang := d.Lang
		if lang == "" {
			lang = "en"
		}
		byLang[lang] = append(byLang[lang], d)
	}

	total := 0
	for lang, langDocs := range byLang {
		data, err := json.Marshal(langDocs)
		if err != nil {
			return err
		}
		if err := ctx.WriteFile(fmt.Sprintf("search-index.%s.json", lang), data); err != nil {
			return err
		}
		total += len(langDocs)
	}

	if err := writeSearchAssets(ctx); err != nil {
		return err
	}
	ctx.Log(fmt.Sprintf("Built search index (%d pages, %d languages)", total, len(byLang)))
	return nil
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

// stripHTML removes HTML tags and collapses whitespace.
func stripHTML(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inTag := false
	lastSpace := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '<' {
			inTag = true
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
			continue
		}
		if c == '>' {
			inTag = false
			continue
		}
		if inTag {
			continue
		}
		if c <= ' ' {
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
			continue
		}
		b.WriteByte(c)
		lastSpace = false
	}
	return strings.TrimSpace(b.String())
}
