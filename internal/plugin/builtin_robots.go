package plugin

import (
	"fmt"
	"strings"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
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
	// Point at sitemap.xml only when the sitemap plugin actually writes it.
	includeSitemap := cfgutil.Bool(cfg, "sitemap", true) && sitemapEnabled(ctx.Config)

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

// sitemapEnabled reports whether the sitemap plugin runs in this build.
func sitemapEnabled(cfg *config.SiteConfig) bool {
	return cfg == nil || cfg.SitemapActive()
}
