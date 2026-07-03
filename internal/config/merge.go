package config

import (
	"fmt"
	"path/filepath"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/validate"
)

// ResolveOptions provides inputs for the 5-layer config cascade.
type ResolveOptions struct {
	ConfigPath   string         // path to sarde.yaml (default: "sarde.yaml")
	ThemeDir     string         // path to active theme dir (empty = skip theme layer)
	CLIFlags     map[string]any // flag overrides from Cobra
	EnvPrefix    string         // env var prefix (default: "SARDE")
	Strict       bool           // reject unknown fields in user sarde.yaml
	KnownPlugins []string       // valid plugin names (collected by build layer from registries)
}

// Resolve loads and merges all config layers, returning the final SiteConfig.
//
// Layer precedence (last wins):
//  1. Embedded defaults (compiled into binary)
//  2. theme.yaml (from active theme directory)
//  3. sarde.yaml (user's project-level config)
//  4. CLI flags
//  5. Environment variables (SARDE_ prefix)
func Resolve(opts ResolveOptions) (*SiteConfig, error) {
	// Layer 1: embedded defaults
	cfg := Defaults()

	// Layer 2: theme.yaml (if theme dir specified)
	if opts.ThemeDir != "" {
		themePath := filepath.Join(opts.ThemeDir, consts.FileThemeConfig)
		themeCfg, err := LoadFile(themePath)
		if err != nil {
			return nil, err
		}
		if themeCfg != nil {
			mergeConfig(cfg, themeCfg)
		}
	}

	// Layer 3: user's sarde.yaml
	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = consts.FileSiteConfig
	}
	var (
		userCfg *SiteConfig
		err     error
	)
	if opts.Strict {
		userCfg, err = LoadFileStrict(configPath)
	} else {
		userCfg, err = LoadFile(configPath)
	}
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

	// Normalize fields that accept multiple input forms.
	cfg.Build.BasePath = NormalizeBasePath(cfg.Build.BasePath)

	// Apply i18n defaults and validation.
	if err := normalizeI18n(&cfg.I18n); err != nil {
		return nil, err
	}

	// Validate the merged config.
	errs, warns := Validate(cfg, opts.KnownPlugins)
	for _, w := range warns {
		devlog.Warn("config", "%s", w.Error())
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("config validation failed:\n%s", validate.FormatErrors(errs))
	}

	return cfg, nil
}

func normalizeI18n(i *I18nSettings) error {
	if !i.IsMultiLang() {
		return nil
	}

	// Defaults
	if i.Strategy == "" {
		i.Strategy = "prefix-except-default"
	}
	if i.Fallback == "" {
		i.Fallback = "default"
	}
	for code, lc := range i.Languages {
		if lc.Dir == "" {
			lc.Dir = "ltr"
			i.Languages[code] = lc
		}
	}

	// Validation
	if i.Strategy != "prefix-except-default" {
		return fmt.Errorf("i18n: unsupported strategy %q (only \"prefix-except-default\" is supported)", i.Strategy)
	}
	if i.Fallback != "default" && i.Fallback != "omit" {
		return fmt.Errorf("i18n: unsupported fallback %q (must be \"default\" or \"omit\")", i.Fallback)
	}
	defLang := i.GetDefaultLanguage()
	if _, ok := i.Languages[defLang]; !ok {
		return fmt.Errorf("i18n: default_language %q is not listed in languages", defLang)
	}

	return nil
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
	if v, ok := flags["build.future"].(bool); ok {
		cfg.Build.Future = BoolPtr(v)
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
	mergeIcons(&base.Icons, &over.Icons)
	mergeLinkValidation(&base.LinkValidation, &over.LinkValidation)
	mergeContentLint(&base.ContentLint, &over.ContentLint)
	mergeAnalytics(&base.Analytics, &over.Analytics)
	mergeDeploy(&base.Deploy, &over.Deploy)
	mergeStringMap(&base.Redirects, over.Redirects)
	mergeCollections(&base.Collections, over.Collections)
	mergeHomepage(&base.Homepage, &over.Homepage)
	mergePlugins(&base.Plugins, &over.Plugins)
	mergeTaxonomies(&base.Taxonomies, over.Taxonomies)
	mergeServer(&base.Server, &over.Server)
	mergeStringMap(&base.Permalinks, over.Permalinks)
	mergeI18n(&base.I18n, &over.I18n)
	mergeContent(&base.Content, &over.Content)
	mergeLlmsTxt(&base.LlmsTxt, &over.LlmsTxt)
	mergeSecurity(&base.Security, &over.Security)
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
	mergeStr(&base.TitleDelimiter, over.TitleDelimiter)
	mergeBoolP(&base.HeadingLinks, over.HeadingLinks)
	mergeStr(&base.Custom404, over.Custom404)
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
	if len(over.DarkOverrides) > 0 {
		base.DarkOverrides = over.DarkOverrides
	}
	mergeStr(&base.PrimaryColor, over.PrimaryColor)
	mergeStr(&base.AccentColor, over.AccentColor)
	mergeStr(&base.FontFamily, over.FontFamily)
	mergeStr(&base.FontMono, over.FontMono)
	mergeStr(&base.CodeLight, over.CodeLight)
	mergeStr(&base.CodeDark, over.CodeDark)
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
	mergeBoolP(&base.AutoGenerate, over.AutoGenerate)
	if len(over.Items) > 0 {
		base.Items = over.Items
	}
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
	mergeBoolP(&base.Credits, over.Credits)
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
	if over.LastUpdated != "" {
		base.LastUpdated = over.LastUpdated
	}
	mergeBoolP(&base.Feed, over.Feed)
	mergeBoolP(&base.Drafts, over.Drafts)
	mergeBoolP(&base.Future, over.Future)
	mergeBoolP(&base.Expired, over.Expired)
	mergeBoolP(&base.Parallel, over.Parallel)
	mergeBoolP(&base.Cache, over.Cache)
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
	mergeStr(&base.Codeblocks.Engine, over.Codeblocks.Engine)
	mergeStr(&base.Codeblocks.Style, over.Codeblocks.Style)
	mergeStr(&base.Codeblocks.LightTheme, over.Codeblocks.LightTheme)
	mergeStr(&base.Codeblocks.DarkTheme, over.Codeblocks.DarkTheme)
	mergeStr(&base.Codeblocks.Theme, over.Codeblocks.Theme)
	mergeStr(&base.Codeblocks.DarkModeSelector, over.Codeblocks.DarkModeSelector)
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

func mergeIcons(base, over *IconSettings) {
	mergeStr(&base.DefaultPrefix, over.DefaultPrefix)
	if len(over.Sets) > 0 {
		base.Sets = over.Sets
	}
	mergeStr(&base.SetsDir, over.SetsDir)
	mergeStr(&base.LocalDir, over.LocalDir)
	mergeStr(&base.Attribution, over.Attribution)
	mergeStr(&base.Render, over.Render)
}

func mergeLinkValidation(base, over *LinkValidationSettings) {
	mergeBoolP(&base.Enabled, over.Enabled)
	mergeStr(&base.Level, over.Level)
	mergeStr(&base.OnBroken, over.OnBroken)
	mergeStr(&base.OnBrokenAnchor, over.OnBrokenAnchor)
	mergeStr(&base.Report, over.Report)
	mergeStr(&base.OnRelativeLinks, over.OnRelativeLinks)
	mergeStr(&base.OnLocalLinks, over.OnLocalLinks)
	mergeStr(&base.OnUnverifiedInternal, over.OnUnverifiedInternal)
	mergeBoolP(&base.CheckAnchors, over.CheckAnchors)
	mergeBoolP(&base.CheckImages, over.CheckImages)
	mergeStr(&base.SameSitePolicy, over.SameSitePolicy)
	mergeStr(&base.SiteRootEscapePrefix, over.SiteRootEscapePrefix)
	mergeBoolP(&base.FailBuild, over.FailBuild)
	if len(over.Exclude) > 0 {
		base.Exclude = over.Exclude
	}
	if len(over.Ignore) > 0 {
		base.Ignore = over.Ignore
	}
	mergeExternalCheck(&base.External, &over.External)
}

func mergeExternalCheck(base, over *ExternalCheckSettings) {
	mergeBoolP(&base.Check, over.Check)
	mergeInt(&base.Concurrency, over.Concurrency)
	mergeStr(&base.Timeout, over.Timeout)
	mergeStr(&base.Cache, over.Cache)
	mergeStr(&base.CacheTTL, over.CacheTTL)
	mergeStr(&base.OnBroken, over.OnBroken)
	mergeStr(&base.Method, over.Method)
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
	mergeStr(&base.Script, over.Script)
}

func mergeDeploy(base, over *DeployConfig) {
	mergeStr(&base.Provider, over.Provider)
	mergeStr(&base.Branch, over.Branch)
	mergeStr(&base.SiteID, over.SiteID)
	mergeStr(&base.ProjectName, over.ProjectName)
	mergeStr(&base.ProjectID, over.ProjectID)
	mergeStr(&base.Command, over.Command)
	mergeStr(&base.RedirectFormat, over.RedirectFormat)
}

func mergeCollections(base *map[string]*CollectionSiteConfig, over map[string]*CollectionSiteConfig) {
	if len(over) == 0 {
		return
	}
	if *base == nil {
		*base = make(map[string]*CollectionSiteConfig)
	}
	for k, v := range over {
		existing, ok := (*base)[k]
		if !ok || existing == nil {
			(*base)[k] = v
			continue
		}
		mergeStr(&existing.Path, v.Path)
		mergeStr(&existing.URLPrefix, v.URLPrefix)
		mergeStr(&existing.Sort, v.Sort)
		mergeStr(&existing.Layout, v.Layout)
		mergeStr(&existing.Permalink, v.Permalink)
		mergeInt(&existing.Paginate, v.Paginate)
		mergeBoolP(&existing.Enabled, v.Enabled)
		mergeBoolP(&existing.Feed, v.Feed)
		mergeBoolP(&existing.Tabs, v.Tabs)
		if v.Sidebar != nil {
			existing.Sidebar = v.Sidebar
		}
		if v.TOC != nil {
			existing.TOC = v.TOC
		}
		if v.PrevNext != nil {
			existing.PrevNext = v.PrevNext
		}
		if v.Versioning != nil {
			existing.Versioning = v.Versioning
		}
		mergeStr(&existing.I18nFallback, v.I18nFallback)
		(*base)[k] = existing
	}
}

func mergeHomepage(base, over *HomepageSettings) {
	mergeStr(&base.Template, over.Template)
	mergeStr(&base.Hero.Eyebrow, over.Hero.Eyebrow)
	mergeStr(&base.Hero.Title, over.Hero.Title)
	mergeStr(&base.Hero.Subtitle, over.Hero.Subtitle)
	mergeStr(&base.Hero.Background, over.Hero.Background)
	if over.Hero.CTA != nil {
		base.Hero.CTA = over.Hero.CTA
	}
	if over.Hero.SecondaryCTA != nil {
		base.Hero.SecondaryCTA = over.Hero.SecondaryCTA
	}
	if len(over.Hero.Stats) > 0 {
		base.Hero.Stats = over.Hero.Stats
	}
	if over.Hero.Code != nil {
		base.Hero.Code = over.Hero.Code
	}
	if over.Hero.Image != nil {
		base.Hero.Image = over.Hero.Image
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
	mergeStr(&base.Host, over.Host)
	mergeInt(&base.Port, over.Port)
	mergeBoolP(&base.LiveReload, over.LiveReload)
}

func mergeI18n(base, over *I18nSettings) {
	mergeStr(&base.DefaultLanguage, over.DefaultLanguage)
	mergeStr(&base.Strategy, over.Strategy)
	mergeStr(&base.Fallback, over.Fallback)
	// Strict is a plain bool: an overriding layer can enable it but not
	// disable one set by a lower layer. Converting to *bool would allow
	// explicit disabling but is a wider change than needed here.
	if over.Strict {
		base.Strict = true
	}
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

func mergeSecurity(base, over *SecurityConfig) {
	if len(over.BlockedHrefSchemes) > 0 {
		base.BlockedHrefSchemes = over.BlockedHrefSchemes
	}
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

// mergeStringMap merges over into base per-key (later layer wins per key),
// rather than replacing the whole map. This preserves earlier-layer entries
// (e.g. theme-provided redirects/permalinks) when a later layer adds its own.
func mergeStringMap(base *map[string]string, over map[string]string) {
	if len(over) == 0 {
		return
	}
	if *base == nil {
		*base = make(map[string]string, len(over))
	}
	for k, v := range over {
		(*base)[k] = v
	}
}

func mergeTaxonomies(base *map[string]TaxonomyConfig, over map[string]TaxonomyConfig) {
	if len(over) == 0 {
		return
	}
	if *base == nil {
		*base = make(map[string]TaxonomyConfig)
	}
	for k, v := range over {
		existing, ok := (*base)[k]
		if !ok {
			(*base)[k] = v
			continue
		}
		if v.Singular != "" {
			existing.Singular = v.Singular
		}
		if v.PaginateBy != 0 {
			existing.PaginateBy = v.PaginateBy
		}
		if v.Render != nil {
			existing.Render = v.Render
		}
		if v.UndefinedTags != "" {
			existing.UndefinedTags = v.UndefinedTags
		}
		(*base)[k] = existing
	}
}
