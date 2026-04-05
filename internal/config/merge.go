package config

import "path/filepath"

// ResolveOptions provides inputs for the 5-layer config cascade.
type ResolveOptions struct {
	ConfigPath string         // path to site.yaml (default: "site.yaml")
	ThemeDir   string         // path to active theme dir (empty = skip theme layer)
	CLIFlags   map[string]any // flag overrides from Cobra
	EnvPrefix  string         // env var prefix (default: "CODEROO")
}

// Resolve loads and merges all config layers, returning the final SiteConfig.
//
// Layer precedence (last wins):
//  1. Embedded defaults (compiled into binary)
//  2. theme.yaml (from active theme directory)
//  3. site.yaml (user's project-level config)
//  4. CLI flags
//  5. Environment variables (CODEROO_ prefix)
func Resolve(opts ResolveOptions) (*SiteConfig, error) {
	// Layer 1: embedded defaults
	cfg := Defaults()

	// Layer 2: theme.yaml (if theme dir specified)
	if opts.ThemeDir != "" {
		themePath := filepath.Join(opts.ThemeDir, "theme.yaml")
		themeCfg, err := LoadFile(themePath)
		if err != nil {
			return nil, err
		}
		if themeCfg != nil {
			mergeConfig(cfg, themeCfg)
		}
	}

	// Layer 3: user's site.yaml
	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = "site.yaml"
	}
	userCfg, err := LoadFile(configPath)
	if err != nil {
		return nil, err
	}
	if userCfg != nil {
		mergeConfig(cfg, userCfg)
	}

	// Layer 4: CLI flags
	if len(opts.CLIFlags) > 0 {
		applyCLIFlagOverrides(cfg, opts.CLIFlags)
	}

	// Layer 5: environment variables
	applyEnvOverrides(cfg, opts.EnvPrefix)

	return cfg, nil
}

// ---------------------------------------------------------------------------
// CLI flag application
// ---------------------------------------------------------------------------

func applyCLIFlagOverrides(cfg *SiteConfig, flags map[string]any) {
	if v, ok := flags["site.url"].(string); ok && v != "" {
		cfg.Site.URL = v
	}
	if v, ok := flags["build.drafts"].(bool); ok {
		cfg.Build.Drafts = BoolPtr(v)
	}
}

// ---------------------------------------------------------------------------
// mergeConfig overlays non-zero values from over onto base.
// ---------------------------------------------------------------------------

func mergeConfig(base, over *SiteConfig) {
	mergeSiteIdentity(&base.Site, &over.Site)
	mergeSocial(&base.Social, over.Social)
	mergeTheme(&base.Theme, &over.Theme)
	mergeTOC(&base.TOC, &over.TOC)
	mergeSidebar(&base.Sidebar, &over.Sidebar)
	mergeHeader(&base.Header, &over.Header)
	mergeFooter(&base.Footer, &over.Footer)
	mergeHead(&base.Head, &over.Head)
	mergeBuild(&base.Build, &over.Build)
	mergeMarkdown(&base.Markdown, &over.Markdown)
	mergePrefetch(&base.Prefetch, &over.Prefetch)
	mergeImages(&base.Images, &over.Images)
	mergeSearch(&base.Search, &over.Search)
	mergeLinkValidation(&base.LinkValidation, &over.LinkValidation)
	mergeContentLint(&base.ContentLint, &over.ContentLint)
	mergeAnalytics(&base.Analytics, &over.Analytics)
	mergeStringMap(&base.Redirects, over.Redirects)
	mergeCollections(&base.Collections, over.Collections)
	mergeHomepage(&base.Homepage, &over.Homepage)
	mergePlugins(&base.Plugins, &over.Plugins)
	mergeStringMap(&base.Taxonomies, over.Taxonomies)
	mergeServer(&base.Server, &over.Server)
	mergeStringMap(&base.Permalinks, over.Permalinks)
	mergeI18n(&base.I18n, &over.I18n)
	mergeContent(&base.Content, &over.Content)
	mergeLlmsTxt(&base.LlmsTxt, &over.LlmsTxt)
}

// ---------------------------------------------------------------------------
// Per-section merge helpers
// ---------------------------------------------------------------------------

func mergeSiteIdentity(base, over *SiteIdentity) {
	mergeStr(&base.Title, over.Title)
	mergeStr(&base.Description, over.Description)
	mergeStr(&base.URL, over.URL)
	mergeLogo(&base.Logo, &over.Logo)
	mergeStr(&base.Favicon, over.Favicon)
	mergeStr(&base.Language, over.Language)
	mergeStr(&base.EditURL, over.EditURL)
}

func mergeLogo(base, over *Logo) {
	mergeStr(&base.Light, over.Light)
	mergeStr(&base.Dark, over.Dark)
	mergeStr(&base.Alt, over.Alt)
}

func mergeSocial(base *[]SocialLink, over []SocialLink) {
	if len(over) > 0 {
		*base = over
	}
}

func mergeTheme(base, over *ThemeSettings) {
	mergeStr(&base.Name, over.Name)
	mergeStr(&base.Preset, over.Preset)
	mergeBoolP(&base.Dark, over.Dark)
	if len(over.Overrides) > 0 {
		base.Overrides = over.Overrides
	}
}

func mergeTOC(base, over *TOCSettings) {
	mergeBoolP(&base.Enabled, over.Enabled)
	mergeInt(&base.MinLevel, over.MinLevel)
	mergeInt(&base.MaxLevel, over.MaxLevel)
}

func mergeSidebar(base, over *SidebarSettings) {
	mergeBoolP(&base.Collapsed, over.Collapsed)
	mergeBoolP(&base.Badges, over.Badges)
	mergeBoolP(&base.Pagination, over.Pagination)
}

func mergeHeader(base, over *HeaderSettings) {
	mergeBoolP(&base.Search, over.Search)
	mergeBoolP(&base.ThemeToggle, over.ThemeToggle)
	mergeBoolP(&base.Social, over.Social)
	if len(over.Links) > 0 {
		base.Links = over.Links
	}
}

func mergeFooter(base, over *FooterSettings) {
	mergeStr(&base.Text, over.Text)
	if len(over.Links) > 0 {
		base.Links = over.Links
	}
}

func mergeHead(base, over *HeadSettings) {
	if len(over.Tags) > 0 {
		base.Tags = over.Tags
	}
	if len(over.CustomCSS) > 0 {
		base.CustomCSS = over.CustomCSS
	}
	if len(over.CustomJS) > 0 {
		base.CustomJS = over.CustomJS
	}
}

func mergeBuild(base, over *BuildSettings) {
	mergeStr(&base.Output, over.Output)
	mergeStr(&base.BasePath, over.BasePath)
	mergeBoolP(&base.Clean, over.Clean)
	mergeBoolP(&base.Sitemap, over.Sitemap)
	mergeBoolP(&base.Minify, over.Minify)
	mergeBoolP(&base.LastUpdated, over.LastUpdated)
	mergeBoolP(&base.Feed, over.Feed)
	mergeBoolP(&base.Drafts, over.Drafts)
	mergeBoolP(&base.Future, over.Future)
	mergeBoolP(&base.Parallel, over.Parallel)
}

func mergeMarkdown(base, over *MarkdownSettings) {
	mergeBoolP(&base.KaTeX, over.KaTeX)
	mergeBoolP(&base.Mermaid, over.Mermaid)
	mergeBoolP(&base.CDN, over.CDN)
	mergeBoolP(&base.Unsafe, over.Unsafe)
	mergeBoolP(&base.Typographer, over.Typographer)
	mergeBoolP(&base.GithubAlerts, over.GithubAlerts)
	mergeBoolP(&base.TripleColonCallouts, over.TripleColonCallouts)
	mergeInt(&base.TOC.MinHeadingLevel, over.TOC.MinHeadingLevel)
	mergeInt(&base.TOC.MaxHeadingLevel, over.TOC.MaxHeadingLevel)
	mergeStr(&base.Highlighting.Style, over.Highlighting.Style)
	mergeStr(&base.Highlighting.LightTheme, over.Highlighting.LightTheme)
	mergeStr(&base.Highlighting.DarkTheme, over.Highlighting.DarkTheme)
	mergeStr(&base.Highlighting.Theme, over.Highlighting.Theme)
}

func mergePrefetch(base, over *PrefetchSettings) {
	mergeBoolP(&base.Enabled, over.Enabled)
	mergeStr(&base.Strategy, over.Strategy)
	mergeInt(&base.Delay, over.Delay)
}

func mergeImages(base, over *ImageSettings) {
	if len(over.Widths) > 0 {
		base.Widths = over.Widths
	}
	if len(over.Formats) > 0 {
		base.Formats = over.Formats
	}
	mergeInt(&base.Quality, over.Quality)
	mergeStr(&base.Placeholder, over.Placeholder)
	mergeInt(&base.MaxWidth, over.MaxWidth)
	mergeBoolP(&base.LazyLoading, over.LazyLoading)
	mergeBoolP(&base.Dimensions, over.Dimensions)
}

func mergeSearch(base, over *SearchSettings) {
	mergeBoolP(&base.Enabled, over.Enabled)
	mergeStr(&base.Provider, over.Provider)
}

func mergeLinkValidation(base, over *LinkValidationSettings) {
	mergeBoolP(&base.Enabled, over.Enabled)
	mergeStr(&base.Level, over.Level)
	mergeBoolP(&base.CheckAnchors, over.CheckAnchors)
	mergeBoolP(&base.CheckImages, over.CheckImages)
	mergeBoolP(&base.WarnRelativeLinks, over.WarnRelativeLinks)
	mergeBoolP(&base.CheckExternal, over.CheckExternal)
	if len(over.Ignore) > 0 {
		base.Ignore = over.Ignore
	}
}

func mergeContentLint(base, over *ContentLintSettings) {
	mergeBoolP(&base.Enabled, over.Enabled)
	mergeInt(&base.Rules.HeadingMaxLength, over.Rules.HeadingMaxLength)
	mergeBoolP(&base.Rules.HeadingIncrement, over.Rules.HeadingIncrement)
	mergeBoolP(&base.Rules.ImageAltRequired, over.Rules.ImageAltRequired)
	mergeBoolP(&base.Rules.NoEmptyLinks, over.Rules.NoEmptyLinks)
	if len(over.Rules.FrontmatterRequired) > 0 {
		base.Rules.FrontmatterRequired = over.Rules.FrontmatterRequired
	}
}

func mergeAnalytics(base, over *AnalyticsSettings) {
	mergeStr(&base.Provider, over.Provider)
	mergeStr(&base.SiteID, over.SiteID)
}

func mergeCollections(base *map[string]*CollectionSiteConfig, over map[string]*CollectionSiteConfig) {
	if len(over) > 0 {
		*base = over
	}
}

func mergeHomepage(base, over *HomepageSettings) {
	mergeStr(&base.Template, over.Template)
	mergeStr(&base.Hero.Title, over.Hero.Title)
	mergeStr(&base.Hero.Subtitle, over.Hero.Subtitle)
	mergeStr(&base.Hero.Background, over.Hero.Background)
	if over.Hero.CTA != nil {
		base.Hero.CTA = over.Hero.CTA
	}
}

func mergePlugins(base, over *PluginSettings) {
	if len(over.Enabled) > 0 {
		base.Enabled = over.Enabled
	}
	if len(over.Config) > 0 {
		base.Config = over.Config
	}
}

func mergeServer(base, over *ServerSettings) {
	mergeInt(&base.Port, over.Port)
	mergeBoolP(&base.LiveReload, over.LiveReload)
}

func mergeI18n(base, over *I18nSettings) {
	mergeStr(&base.DefaultLanguage, over.DefaultLanguage)
	if len(over.Languages) > 0 {
		base.Languages = over.Languages
	}
}

func mergeContent(base, over *ContentSettings) {
	mergeStr(&base.Dir, over.Dir)
	mergeInt(&base.SummaryLength, over.SummaryLength)
}

func mergeLlmsTxt(base, over *LlmsTxtSettings) {
	mergeBoolP(&base.Enabled, over.Enabled)
	mergeBoolP(&base.IncludeBlog, over.IncludeBlog)
}

// ---------------------------------------------------------------------------
// Primitive merge helpers
// ---------------------------------------------------------------------------

func mergeStr(base *string, over string) {
	if over != "" {
		*base = over
	}
}

func mergeInt(base *int, over int) {
	if over != 0 {
		*base = over
	}
}

func mergeBoolP(base **bool, over *bool) {
	if over != nil {
		*base = over
	}
}

func mergeStringMap(base *map[string]string, over map[string]string) {
	if len(over) > 0 {
		*base = over
	}
}
