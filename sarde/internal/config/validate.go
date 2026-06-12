package config

import (
	"fmt"
	"strconv"

	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/validate"
)

// Validate checks the merged SiteConfig for invalid values.
// knownPlugins is the set of valid plugin names, collected by the build layer
// from the actual plugin registries. If nil, plugin name validation is skipped.
func Validate(cfg *SiteConfig, knownPlugins []string) (errs []validate.Error, warns []validate.Error) {
	var c validate.Checker  // hard errors
	var w validate.Checker  // warnings

	validateRequired(&c, cfg)
	validateRecommended(&w, cfg)
	validateEnums(&c, cfg)
	validateRanges(&c, cfg)
	validateInterdependencies(&c, cfg)
	validateCollections(&c, cfg)
	validateSliceElements(&c, cfg, knownPlugins)
	return c.Errors(), w.Errors()
}

// --- Hard-error required fields ---

func validateRequired(c *validate.Checker, cfg *SiteConfig) {
	c.Required("site.title", cfg.Site.Title)
	c.Required("site.language", cfg.Site.Language)
	c.Required("build.output", cfg.Build.Output)
	c.Required("content.dir", cfg.Content.Dir)
}

// --- Warning-level recommended fields ---

func validateRecommended(w *validate.Checker, cfg *SiteConfig) {
	w.Required("site.url", cfg.Site.URL)
	w.Required("site.description", cfg.Site.Description)
}

// --- Enum checks ---

func validateEnums(c *validate.Checker, cfg *SiteConfig) {
	c.OneOf("icons.render", cfg.Icons.Render, []string{"inline", "sprite"})
	c.OneOf("link_validation.on_broken", cfg.LinkValidation.OnBroken, []string{"error", "warn", "ignore"})
	c.OneOf("link_validation.on_broken_anchor", cfg.LinkValidation.OnBrokenAnchor, []string{"error", "warn", "ignore"})
	c.OneOf("link_validation.on_relative_links", cfg.LinkValidation.OnRelativeLinks, []string{"error", "warn", "ignore"})
	c.OneOf("link_validation.on_local_links", cfg.LinkValidation.OnLocalLinks, []string{"error", "warn", "ignore"})
	c.OneOf("link_validation.on_unverified_internal", cfg.LinkValidation.OnUnverifiedInternal, []string{"error", "warn", "ignore"})
	c.OneOf("link_validation.report", cfg.LinkValidation.Report, []string{"pretty", "json", "github-actions"})
	c.OneOf("link_validation.level", cfg.LinkValidation.Level, []string{"error", "warn", "ignore"})
	c.OneOf("link_validation.same_site_policy", cfg.LinkValidation.SameSitePolicy, []string{"error", "warn", "ignore"})
	c.OneOf("link_validation.external.on_broken", cfg.LinkValidation.External.OnBroken, []string{"error", "warn", "ignore"})
	c.OneOf("link_validation.external.method", cfg.LinkValidation.External.Method, []string{"head-then-get", "head", "get"})
	c.OneOf("deploy.provider", cfg.Deploy.Provider, []string{"github", "netlify", "cloudflare", "vercel", "custom"})
	c.OneOf("deploy.redirect_format", cfg.Deploy.RedirectFormat, []string{"html", "netlify", "vercel", "all"})
	c.OneOf("prefetch.strategy", cfg.Prefetch.Strategy, []string{"hover", "visible", "idle"})
	c.OneOf("images.placeholder", cfg.Images.Placeholder, []string{"lqip", "blur", "dominantColor", "none"})
	c.OneOf("search.provider", cfg.Search.Provider, []string{"orama"})
	c.OneOf("markdown.codeblocks.style", cfg.Markdown.Codeblocks.Style, []string{"class"})
	c.OneOf("build.last_updated", string(cfg.Build.LastUpdated), []string{"git", "mtime", "false", "off", "none"})
}

// --- Range checks ---

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
	c.IntMin("prefetch.delay", cfg.Prefetch.Delay, 0)
	c.IntMin("content_lint.rules.heading_max_length", cfg.ContentLint.Rules.HeadingMaxLength, 1)
}

// --- Cross-field constraints ---

func validateInterdependencies(c *validate.Checker, cfg *SiteConfig) {
	c.LessOrEqual("toc.min_level", cfg.TOC.MinLevel, "toc.max_level", cfg.TOC.MaxLevel)
	c.LessOrEqual("markdown.toc.min_heading_level", cfg.Markdown.TOC.MinHeadingLevel,
		"markdown.toc.max_heading_level", cfg.Markdown.TOC.MaxHeadingLevel)
}

// --- Per-collection, per-taxonomy, per-language checks ---

func validateCollections(c *validate.Checker, cfg *SiteConfig) {
	for name, col := range cfg.Collections {
		prefix := "collections." + name
		if col.Layout != "" {
			c.Check(prefix+".layout", col.Layout, engine.ValidateLayout(engine.LayoutType(col.Layout)),
				"must be one of: default, docs, splash, wide, full, centered, split, presentation")
		}
		c.OneOf(prefix+".i18n_fallback", col.I18nFallback, []string{"default", "omit"})
		c.IntMin(prefix+".paginate", col.Paginate, 1)

		if col.Sidebar != nil {
			c.IntRange(prefix+".sidebar.max_depth", col.Sidebar.MaxDepth, 1, 10)
		}
		if col.TOC != nil {
			c.IntRange(prefix+".toc.depth", col.TOC.Depth, 1, 6)
		}

		if col.Versioning != nil {
			c.OneOf(prefix+".versioning.fallback", col.Versioning.Fallback, []string{"default", "omit"})
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
		c.IntMin("taxonomies."+name+".paginate_by", tax.PaginateBy, 1)
	}
	for code, lang := range cfg.I18n.Languages {
		c.OneOf("i18n.languages."+code+".dir", lang.Dir, []string{"ltr", "rtl"})
	}
}

// --- Slice element validation ---

func validateSliceElements(c *validate.Checker, cfg *SiteConfig, knownPlugins []string) {
	validFormats := []string{"jpeg", "jpg", "png", "webp", "avif"}
	for i, f := range cfg.Images.Formats {
		c.OneOf(fmt.Sprintf("images.formats[%d]", i), f, validFormats)
	}
	for i, w := range cfg.Images.Widths {
		if w < 1 {
			c.Check(fmt.Sprintf("images.widths[%d]", i), strconv.Itoa(w), false,
				"must be at least 1")
		}
	}
	if len(knownPlugins) > 0 {
		for i, name := range cfg.Plugins.Enabled {
			c.OneOf(fmt.Sprintf("plugins.enabled[%d]", i), name, knownPlugins)
		}
	}
}
