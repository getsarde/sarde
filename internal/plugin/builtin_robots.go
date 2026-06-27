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
		sb.WriteString(fmt.Sprintf("Sitemap: %s\n", ctx.AbsURL("/sitemap.xml", "", "")))
	}

	if err := ctx.WriteFile("robots.txt", []byte(sb.String())); err != nil {
		return err
	}
	ctx.Log("Generated robots.txt")
	return nil
}
