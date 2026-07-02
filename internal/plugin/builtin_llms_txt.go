package plugin

import (
	"fmt"
	"strings"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

func newLlmsTxtPlugin(cfg map[string]any) *Plugin {
	return &Plugin{
		Name: "llms_txt",
		Hooks: PluginHooks{
			BuildDone: func(ctx *BuildDoneContext) error {
				return llmsTxtBuildDone(ctx)
			},
		},
	}
}

// blogCollectionNames lists collection names that are considered "blog" content.
var blogCollectionNames = map[string]bool{
	"blog": true, "posts": true, "articles": true, "news": true,
}

func llmsTxtBuildDone(ctx *BuildDoneContext) error {
	settings := ctx.Config.LlmsTxt
	if settings.Enabled != nil && !*settings.Enabled {
		return nil
	}

	includeBlog := config.BoolVal(settings.IncludeBlog, true)

	baseURL := ctx.BaseURL()
	title := "Site"
	description := ""
	if ctx.Site != nil && ctx.Site.Title != "" {
		title = ctx.Site.Title
	}
	if ctx.Config.Site.Description != "" {
		description = ctx.Config.Site.Description
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s\n", title)
	if description != "" {
		fmt.Fprintf(&sb, "\n> %s\n", description)
	}
	sb.WriteString("\n## Pages\n\n")

	for _, page := range ctx.Pages {
		if page.Draft {
			continue
		}
		if page.Kind == engine.KindSection || page.Kind == engine.KindHome {
			continue
		}

		// Filter blog pages if include_blog is false.
		if !includeBlog && page.Collection != nil && blogCollectionNames[page.Collection.Name] {
			continue
		}

		url := baseURL + page.URL()
		fmt.Fprintf(&sb, "- [%s](%s)\n", page.Title, url)
	}

	if err := ctx.WriteFile("llms.txt", []byte(sb.String())); err != nil {
		return err
	}
	ctx.Log("Generated llms.txt")
	return nil
}
