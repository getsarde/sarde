package plugin

import (
	"fmt"
	"strings"
)

func newRobotsPlugin(cfg map[string]any) *Plugin {
	return &Plugin{
		Name: "robots",
		Hooks: PluginHooks{
			BuildDone: func(ctx *BuildDoneContext) error {
				return robotsBuildDone(ctx, cfg)
			},
		},
	}
}

func robotsBuildDone(ctx *BuildDoneContext, cfg map[string]any) error {
	includeSitemap := cfgBool(cfg, "sitemap", true)

	var sb strings.Builder
	sb.WriteString("User-agent: *\n")
	sb.WriteString("Allow: /\n")

	if includeSitemap && ctx.Site != nil && ctx.Site.BaseURL != "" {
		baseURL := strings.TrimRight(ctx.Site.BaseURL, "/")
		basePath := ctx.Site.BasePath
		if basePath == "" || basePath == "/" {
			basePath = "/"
		}
		sb.WriteString(fmt.Sprintf("Sitemap: %s%ssitemap.xml\n", baseURL, basePath))
	}

	if err := ctx.WriteFile("robots.txt", []byte(sb.String())); err != nil {
		return err
	}
	ctx.Log("Generated robots.txt")
	return nil
}
