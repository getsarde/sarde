package config

import (
	"fmt"

	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/validate"
)

// Validate checks the merged SiteConfig for invalid enum values, out-of-range
// numbers, and cross-field constraint violations. Returns nil if everything is
// valid.
func Validate(cfg *SiteConfig) []validate.Error {
	var c validate.Checker
	validateRequired(&c, cfg)
	validateEnums(&c, cfg)
	validateRanges(&c, cfg)
	validateInterdependencies(&c, cfg)
	validateCollections(&c, cfg)
	return c.Errors()
}

func validateRequired(c *validate.Checker, cfg *SiteConfig) {
	c.Required("site.title", cfg.Site.Title)
	c.Required("site.language", cfg.Site.Language)
	c.Required("build.output", cfg.Build.Output)
	c.Required("content.dir", cfg.Content.Dir)
}

func validateEnums(c *validate.Checker, cfg *SiteConfig) {
	c.OneOf("icons.render", cfg.Icons.Render, []string{"inline", "sprite"})
	c.OneOf("link_validation.on_broken", cfg.LinkValidation.OnBroken, []string{"error", "warn", "ignore"})
	c.OneOf("link_validation.on_broken_anchor", cfg.LinkValidation.OnBrokenAnchor, []string{"error", "warn", "ignore"})
	c.OneOf("link_validation.on_relative_links", cfg.LinkValidation.OnRelativeLinks, []string{"error", "warn", "ignore"})
	c.OneOf("link_validation.on_local_links", cfg.LinkValidation.OnLocalLinks, []string{"error", "warn", "ignore"})
	c.OneOf("link_validation.on_unverified_internal", cfg.LinkValidation.OnUnverifiedInternal, []string{"error", "warn", "ignore"})
	c.OneOf("link_validation.report", cfg.LinkValidation.Report, []string{"pretty", "json", "github-actions"})
	c.OneOf("link_validation.external.on_broken", cfg.LinkValidation.External.OnBroken, []string{"error", "warn", "ignore"})
	c.OneOf("link_validation.external.method", cfg.LinkValidation.External.Method, []string{"head-then-get", "head", "get"})
	c.OneOf("deploy.provider", cfg.Deploy.Provider, []string{"github", "netlify", "cloudflare", "vercel", "custom"})
	c.OneOf("deploy.redirect_format", cfg.Deploy.RedirectFormat, []string{"html", "netlify", "vercel", "all"})
	c.OneOf("prefetch.strategy", cfg.Prefetch.Strategy, []string{"hover", "visible", "idle"})
	c.OneOf("images.placeholder", cfg.Images.Placeholder, []string{"lqip", "blur", "dominantColor", "none"})
}

func validateRanges(c *validate.Checker, cfg *SiteConfig) {
	c.IntRange("server.port", cfg.Server.Port, 1, 65535)
	c.IntRange("toc.min_level", cfg.TOC.MinLevel, 1, 6)
	c.IntRange("toc.max_level", cfg.TOC.MaxLevel, 1, 6)
	c.IntRange("markdown.toc.min_heading_level", cfg.Markdown.TOC.MinHeadingLevel, 1, 6)
	c.IntRange("markdown.toc.max_heading_level", cfg.Markdown.TOC.MaxHeadingLevel, 1, 6)
	c.IntRange("images.quality", cfg.Images.Quality, 1, 100)
	c.IntMin("images.max_width", cfg.Images.MaxWidth, 1)
	c.IntMin("link_validation.external.concurrency", cfg.LinkValidation.External.Concurrency, 1)
	c.IntMin("content.summary_length", cfg.Content.SummaryLength, 1)
}

func validateInterdependencies(c *validate.Checker, cfg *SiteConfig) {
	c.LessOrEqual("toc.min_level", cfg.TOC.MinLevel, "toc.max_level", cfg.TOC.MaxLevel)
	c.LessOrEqual("markdown.toc.min_heading_level", cfg.Markdown.TOC.MinHeadingLevel,
		"markdown.toc.max_heading_level", cfg.Markdown.TOC.MaxHeadingLevel)
}

func validateCollections(c *validate.Checker, cfg *SiteConfig) {
	for name, col := range cfg.Collections {
		prefix := "collections." + name
		if col.Layout != "" {
			c.Check(prefix+".layout", col.Layout, engine.ValidateLayout(engine.LayoutType(col.Layout)),
				"must be one of: default, docs, splash, wide, full, centered, split, presentation")
		}
		c.OneOf(prefix+".i18n_fallback", col.I18nFallback, []string{"default", "omit"})

		if col.Versioning != nil {
			for i, v := range col.Versioning.Versions {
				vp := fmt.Sprintf("%s.versioning.versions[%d]", prefix, i)
				c.OneOf(vp+".banner", string(v.Banner), []string{"none", "unmaintained", "unreleased"})
				c.OneOf(vp+".redirect", string(v.Redirect), []string{"same-page", "root"})
			}
		}
	}
	for name, tax := range cfg.Taxonomies {
		c.OneOf("taxonomies."+name+".undefined_tags", tax.UndefinedTags,
			[]string{"warn", "error", "ignore", "create"})
	}
	for code, lang := range cfg.I18n.Languages {
		c.OneOf("i18n.languages."+code+".dir", lang.Dir, []string{"ltr", "rtl"})
	}
}
