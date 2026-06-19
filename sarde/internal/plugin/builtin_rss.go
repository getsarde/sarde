package plugin

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/frostybee/sarde/internal/engine"
)

func newRSSPlugin(cfg map[string]any) *Plugin {
	return &Plugin{
		Name: "rss",
		Hooks: PluginHooks{
			BuildDone: func(ctx *BuildDoneContext) error {
				return rssBuildDone(ctx, cfg)
			},
		},
	}
}

func rssBuildDone(ctx *BuildDoneContext, cfg map[string]any) error {
	limit := cfgInt(cfg, "limit", 20)
	baseURL := ""
	if ctx.Site != nil {
		baseURL = strings.TrimRight(ctx.Site.BaseURL, "/")
	}

	feedCollections := feedEnabledCollections(cfg, ctx.Collections)

	feedCount := 0
	for _, colName := range feedCollections {
		col, ok := ctx.Collections[colName]
		if !ok || col == nil {
			continue
		}

		items := buildRSSItems(col.Pages, baseURL, limit)
		feed := rssChannel{
			Title:       col.Title,
			Link:        ctx.AbsURL("/"+colName+"/", "", ""),
			Description: fmt.Sprintf("Latest from %s", col.Title),
			Items:       items,
		}
		if len(items) > 0 {
			feed.LastBuildDate = items[0].PubDate
		}

		rss := rssFeed{
			Version: "2.0",
			Channel: feed,
		}

		data, err := xml.MarshalIndent(rss, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling RSS for %s: %w", colName, err)
		}

		output := []byte(xml.Header + string(data))
		path := colName + "/feed.xml"
		if err := ctx.WriteFile(path, output); err != nil {
			return err
		}
		feedCount++
	}

	if feedCount > 0 {
		ctx.Log(fmt.Sprintf("Generated %d RSS feed(s)", feedCount))
	}
	return nil
}

func buildRSSItems(pages []*engine.Page, baseURL string, limit int) []rssItem {
	var items []rssItem
	for _, page := range pages {
		if page.Draft {
			continue
		}
		if len(items) >= limit {
			break
		}

		pubDate := ""
		if !page.Date.IsZero() {
			pubDate = page.Date.Format(time.RFC1123Z)
		}

		desc := page.Description
		if desc == "" {
			desc = string(page.Summary)
		}

		items = append(items, rssItem{
			Title:       page.Title,
			Link:        baseURL + page.URL(),
			Description: desc,
			PubDate:     pubDate,
			GUID:        baseURL + page.URL(),
		})
	}
	return items
}

// RSS XML types.
type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	LastBuildDate string    `xml:"lastBuildDate,omitempty"`
	Items         []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description,omitempty"`
	PubDate     string `xml:"pubDate,omitempty"`
	GUID        string `xml:"guid"`
}
