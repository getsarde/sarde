package plugin

import (
	"encoding/json"
	"regexp"
	"strings"
)

func newSearchPlugin(cfg map[string]any) *Plugin {
	return &Plugin{
		Name: "search",
		Hooks: PluginHooks{
			BuildDone: func(ctx *BuildDoneContext) error {
				return searchBuildDone(ctx, cfg)
			},
		},
	}
}

type searchDocument struct {
	Title   string   `json:"title"`
	URL     string   `json:"url"`
	Content string   `json:"content,omitempty"`
	Section string   `json:"section,omitempty"`
	Tags    []string `json:"tags,omitempty"`
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
			Title:   page.Title,
			URL:     page.RelPermalink,
			Content: content,
			Section: section,
			Tags:    page.Tags,
		})
	}

	data, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return err
	}

	return ctx.WriteFile("search-index.json", data)
}

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// stripHTML removes HTML tags and collapses whitespace.
func stripHTML(s string) string {
	s = htmlTagRegex.ReplaceAllString(s, " ")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}
