package plugin

import (
	"encoding/xml"
	"html"
	"time"

	"github.com/getsarde/sarde/internal/engine"
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
	return writeFeedFiles(ctx, cfg, "feed.xml", "RSS", func(col *engine.Collection, pages []*engine.Page, lang, baseURL string, limit int) ([]byte, error) {
		items := buildRSSItems(pages, baseURL, limit)
		feed := rssChannel{
			Title:       col.Title,
			Link:        ctx.AbsURL("/"+col.Name+"/", lang, ""),
			Description: "Latest from " + col.Title,
			Items:       items,
		}
		if len(items) > 0 {
			feed.LastBuildDate = items[0].PubDate
		}
		return xml.MarshalIndent(rssFeed{Version: "2.0", Channel: feed}, "", "  ")
	})
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
		if desc == "" {
			// Rendered-text fallback for bodies that are entirely directive
			// blocks. Decoded once here; encoding/xml re-escapes on output.
			desc = html.UnescapeString(RenderedTextFallback(page.Content, RenderedFallbackMaxChars))
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
