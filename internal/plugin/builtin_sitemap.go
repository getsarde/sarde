package plugin

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/getsarde/sarde/internal/plugin/cfgutil"
)

func newSitemapPlugin(cfg map[string]any) *Plugin {
	return &Plugin{
		Name: "sitemap",
		Hooks: PluginHooks{
			BuildDone: func(ctx *BuildDoneContext) error {
				return sitemapBuildDone(ctx, cfg)
			},
		},
	}
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

func sitemapBuildDone(ctx *BuildDoneContext, cfg map[string]any) error {
	changefreq := cfgutil.String(cfg, "changefreq", "weekly")
	priority := cfgutil.String(cfg, "priority", "0.5")
	excludePatterns := cfgutil.StringSlice(cfg, "exclude")

	baseURL := ctx.BaseURL()

	var urls []sitemapURL
	for _, page := range ctx.Pages {
		if page.Draft {
			continue
		}
		if strings.Contains(page.RelPermalink, "/page/") {
			continue
		}
		if shouldExclude(page.RelPermalink, excludePatterns) {
			continue
		}

		loc := baseURL + page.URL()
		lastmod := ""
		if !page.Updated.IsZero() {
			lastmod = page.Updated.Format(time.RFC3339)
		} else if !page.Date.IsZero() {
			lastmod = page.Date.Format(time.RFC3339)
		}

		urls = append(urls, sitemapURL{
			Loc:        loc,
			LastMod:    lastmod,
			ChangeFreq: changefreq,
			Priority:   priority,
		})
	}

	urlset := sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	data, err := xml.MarshalIndent(urlset, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling sitemap: %w", err)
	}

	output := []byte(xml.Header + string(data))
	if err := ctx.WriteFile("sitemap.xml", output); err != nil {
		return err
	}
	ctx.Log("Generated sitemap.xml")
	return nil
}
