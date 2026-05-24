package plugin

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/frostybee/sarde/internal/engine"
)

func newAtomPlugin(cfg map[string]any) *Plugin {
	return &Plugin{
		Name: "atom",
		Hooks: PluginHooks{
			BuildDone: func(ctx *BuildDoneContext) error {
				return atomBuildDone(ctx, cfg)
			},
		},
	}
}

func atomBuildDone(ctx *BuildDoneContext, cfg map[string]any) error {
	limit := cfgInt(cfg, "limit", 20)
	baseURL := ""
	if ctx.Site != nil {
		baseURL = strings.TrimRight(ctx.Site.BaseURL, "/")
	}

	feedCollections := cfgStringSlice(cfg, "collections")
	if len(feedCollections) == 0 {
		for name, col := range ctx.Collections {
			if col.Config != nil && col.Config.Feed {
				feedCollections = append(feedCollections, name)
			}
		}
	}

	feedCount := 0
	for _, colName := range feedCollections {
		col, ok := ctx.Collections[colName]
		if !ok || col == nil {
			continue
		}

		entries := buildAtomEntries(col.Pages, baseURL, limit)

		updated := time.Now().UTC().Format(time.RFC3339)
		if len(entries) > 0 {
			updated = entries[0].Updated
		}

		colURL := baseURL + "/" + colName + "/"
		feed := atomFeed{
			XMLNS:   "http://www.w3.org/2005/Atom",
			Title:   col.Title,
			Updated: updated,
			ID:      colURL,
			Links: []atomLink{
				{Href: colURL, Rel: "alternate", Type: "text/html"},
				{Href: colURL + "atom.xml", Rel: "self", Type: "application/atom+xml"},
			},
			Entries: entries,
		}

		data, err := xml.MarshalIndent(feed, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling Atom for %s: %w", colName, err)
		}

		output := []byte(xml.Header + string(data))
		path := colName + "/atom.xml"
		if err := ctx.WriteFile(path, output); err != nil {
			return err
		}
		feedCount++
	}

	if feedCount > 0 {
		ctx.Log(fmt.Sprintf("Generated %d Atom feed(s)", feedCount))
	}
	return nil
}

func buildAtomEntries(pages []*engine.Page, baseURL string, limit int) []atomEntry {
	var entries []atomEntry
	for _, page := range pages {
		if page.Draft {
			continue
		}
		if len(entries) >= limit {
			break
		}

		updated := page.Date
		if !page.Updated.IsZero() {
			updated = page.Updated
		}
		updatedStr := time.Now().UTC().Format(time.RFC3339)
		if !updated.IsZero() {
			updatedStr = updated.UTC().Format(time.RFC3339)
		}

		summary := page.Description
		if summary == "" {
			summary = string(page.Summary)
		}

		entryURL := baseURL + page.RelPermalink
		entry := atomEntry{
			Title:   page.Title,
			Link:    atomLink{Href: entryURL},
			ID:      entryURL,
			Updated: updatedStr,
			Summary: summary,
		}
		if author, ok := page.Params["author"].(string); ok && author != "" {
			entry.Author = &atomAuthor{Name: author}
		}
		entries = append(entries, entry)
	}
	return entries
}

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	XMLNS   string      `xml:"xmlns,attr"`
	Title   string      `xml:"title"`
	Links   []atomLink  `xml:"link"`
	Updated string      `xml:"updated"`
	ID      string      `xml:"id"`
	Entries []atomEntry `xml:"entry"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr,omitempty"`
	Type string `xml:"type,attr,omitempty"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

type atomEntry struct {
	Title   string      `xml:"title"`
	Link    atomLink    `xml:"link"`
	ID      string      `xml:"id"`
	Updated string      `xml:"updated"`
	Summary string      `xml:"summary,omitempty"`
	Author  *atomAuthor `xml:"author,omitempty"`
}
