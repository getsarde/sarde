package plugin

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
)

// feedBuilder renders a feed document for one collection, restricted to the
// given language's pages. lang is "" for single-language sites.
type feedBuilder func(col *engine.Collection, pages []*engine.Page, lang, baseURL string, limit int) ([]byte, error)

func writeFeedFiles(ctx *BuildDoneContext, cfg map[string]any, filename, label string, build feedBuilder) error {
	limit := cfgutil.Int(cfg, "limit", 20)
	baseURL := ctx.BaseURL()

	feedCollections := feedEnabledCollections(cfg, ctx.Collections)

	isMultiLang := ctx.Config != nil && ctx.Config.I18n.IsMultiLang()
	defaultLang := ""
	if ctx.Config != nil {
		defaultLang = ctx.Config.I18n.GetDefaultLanguage()
	}

	feedCount := 0
	for _, colName := range feedCollections {
		col, ok := ctx.Collections[colName]
		if !ok || col == nil {
			continue
		}

		if !isMultiLang {
			// Single-language site: one feed per collection, unchanged layout.
			if err := writeOneFeed(ctx, build, col, col.Pages, "", filename, label, baseURL, limit); err != nil {
				return err
			}
			feedCount++
			continue
		}

		// Multi-language: one feed per (collection, language). Grouping by the
		// languages actually present avoids emitting empty feeds. Pages carry
		// a lang-aware Permalink already, so item links are correct per page.
		byLang := make(map[string][]*engine.Page)
		var order []string
		for _, p := range col.Pages {
			lang := p.Lang
			if lang == "" {
				lang = defaultLang
			}
			if _, seen := byLang[lang]; !seen {
				order = append(order, lang)
			}
			byLang[lang] = append(byLang[lang], p)
		}
		for _, lang := range order {
			if err := writeOneFeed(ctx, build, col, byLang[lang], lang, filename, label, baseURL, limit); err != nil {
				return err
			}
			feedCount++
		}
	}

	if feedCount > 0 {
		ctx.Log(fmt.Sprintf("Generated %d %s feed(s)", feedCount, label))
	}
	return nil
}

// writeOneFeed builds and writes a single feed file for a collection/language.
func writeOneFeed(ctx *BuildDoneContext, build feedBuilder, col *engine.Collection, pages []*engine.Page, lang, filename, label, baseURL string, limit int) error {
	data, err := build(col, pages, lang, baseURL, limit)
	if err != nil {
		return fmt.Errorf("marshaling %s for %s (lang %q): %w", label, col.Name, lang, err)
	}
	output := []byte(xml.Header + string(data))
	if err := ctx.WriteFile(feedOutputPath(ctx, col.Name, filename, lang), output); err != nil {
		return err
	}
	return nil
}

// feedOutputPath returns the on-disk feed path, lang-prefixed to match the
// site's localized URL layout (e.g. "fr/blog/feed.xml"). The default language
// (and single-language sites) keep the plain "blog/feed.xml".
func feedOutputPath(ctx *BuildDoneContext, colName, filename, lang string) string {
	rel := colName + "/" + filename
	if lang != "" && ctx.Resolver != nil {
		rel = strings.TrimPrefix(ctx.Resolver.OutputRelPath("/"+rel, lang, ""), "/")
	}
	return rel
}
