package config

import (
	"fmt"
	"log"
	"sort"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Helpers for *bool fields
// ---------------------------------------------------------------------------

// BoolPtr returns a pointer to the given bool value.
func BoolPtr(v bool) *bool { return &v }

// BoolVal dereferences a *bool, returning fallback if nil.
func BoolVal(p *bool, fallback bool) bool {
	if p != nil {
		return *p
	}
	return fallback
}

// ---------------------------------------------------------------------------
// SiteConfig — top-level configuration matching site.yaml
// ---------------------------------------------------------------------------

// SiteConfig is the complete site configuration. Every field maps to a
// top-level key in site.yaml. Booleans use *bool so the merge layer can
// distinguish "not set" from "explicitly false".
type SiteConfig struct {
	Site           SiteIdentity                  `yaml:"site"`
	Social         []SocialLink                  `yaml:"social"`
	Theme          ThemeSettings                 `yaml:"theme"`
	TOC            TOCSettings                   `yaml:"toc"`
	Sidebar        SidebarSettings               `yaml:"sidebar"`
	Header         HeaderSettings                `yaml:"header"`
	Footer         FooterSettings                `yaml:"footer"`
	Head           HeadSettings                  `yaml:"head"`
	Build          BuildSettings                 `yaml:"build"`
	Markdown       MarkdownSettings              `yaml:"markdown"`
	Prefetch       PrefetchSettings              `yaml:"prefetch"`
	Images         ImageSettings                 `yaml:"images"`
	Search         SearchSettings                `yaml:"search"`
	LinkValidation LinkValidationSettings        `yaml:"link_validation"`
	ContentLint    ContentLintSettings           `yaml:"content_lint"`
	Analytics      AnalyticsSettings             `yaml:"analytics"`
	Deploy         DeployConfig                  `yaml:"deploy"`
	Redirects      map[string]string             `yaml:"redirects"`
	Collections    map[string]*CollectionSiteConfig `yaml:"collections"`
	Homepage       HomepageSettings              `yaml:"homepage"`
	Plugins        PluginSettings                `yaml:"plugins"`
	Taxonomies     map[string]TaxonomyConfig     `yaml:"taxonomies"`
	Server         ServerSettings                `yaml:"server"`
	Permalinks     map[string]string             `yaml:"permalinks"`
	I18n           I18nSettings                  `yaml:"i18n"`
	Content        ContentSettings               `yaml:"content"`
	LlmsTxt        LlmsTxtSettings              `yaml:"llms_txt"`
	Security       SecurityConfig               `yaml:"security"`
}

// ---------------------------------------------------------------------------
// Site Identity
// ---------------------------------------------------------------------------

type SiteIdentity struct {
	Title          string `yaml:"title"`
	Description    string `yaml:"description"`
	URL            string `yaml:"url"`
	Logo           Logo   `yaml:"logo"`
	Favicon        string `yaml:"favicon"`
	Language       string `yaml:"language"`
	EditURL        string `yaml:"edit_url"`
	TitleDelimiter string `yaml:"title_delimiter"`
	HeadingLinks   *bool  `yaml:"heading_links"`
	Custom404      string `yaml:"custom_404"`
}

// Logo supports both string and object forms in YAML:
//
//	logo: "/img/logo.svg"
//	logo:
//	  light: "/img/logo-light.svg"
//	  dark:  "/img/logo-dark.svg"
//	  alt:   "My Site"
type Logo struct {
	Light string `yaml:"light"`
	Dark  string `yaml:"dark"`
	Alt   string `yaml:"alt"`
}

func (l *Logo) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		l.Light = value.Value
		l.Dark = value.Value
		return nil
	}
	type plain Logo
	return value.Decode((*plain)(l))
}

// ---------------------------------------------------------------------------
// Social Links
// ---------------------------------------------------------------------------

type SocialLink struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
	Icon  string `yaml:"icon"`
}

// ---------------------------------------------------------------------------
// Theme & Appearance
// ---------------------------------------------------------------------------

type ThemeSettings struct {
	Name         string            `yaml:"name"`
	Preset       string            `yaml:"preset"`
	Dark         *bool             `yaml:"dark"`
	Overrides    map[string]string `yaml:"overrides"`
	PrimaryColor string            `yaml:"primary_color"`
	AccentColor  string            `yaml:"accent_color"`
	FontFamily   string            `yaml:"font_family"`
	FontMono     string            `yaml:"font_mono"`
	CodeLight    string            `yaml:"code_light"`
	CodeDark     string            `yaml:"code_dark"`
}

// ---------------------------------------------------------------------------
// Table of Contents
// ---------------------------------------------------------------------------

type TOCSettings struct {
	Enabled  *bool `yaml:"enabled"`
	MinLevel int   `yaml:"min_level"`
	MaxLevel int   `yaml:"max_level"`
}

// ---------------------------------------------------------------------------
// Sidebar
// ---------------------------------------------------------------------------

type SidebarSettings struct {
	Collapsed    *bool         `yaml:"collapsed"`
	Badges       *bool         `yaml:"badges"`
	Pagination   *bool         `yaml:"pagination"`
	AutoGenerate *bool         `yaml:"auto_generate"`
	Items        []SidebarItem `yaml:"items"`
}

// SidebarItem is a single entry in a manually-defined sidebar.
type SidebarItem struct {
	Label     string        `yaml:"label"`
	Link      string        `yaml:"link,omitempty"`
	Collapsed *bool         `yaml:"collapsed,omitempty"`
	Items     []SidebarItem `yaml:"items,omitempty"`
}

// ---------------------------------------------------------------------------
// Header & Footer
// ---------------------------------------------------------------------------

type HeaderSettings struct {
	Search      *bool     `yaml:"search"`
	ThemeToggle *bool     `yaml:"theme_toggle"`
	Social      *bool     `yaml:"social"`
	Links       []NavLink `yaml:"links"`
}

type FooterSettings struct {
	Text    string    `yaml:"text"`
	Links   []NavLink `yaml:"links"`
	Credits *bool     `yaml:"credits"`
}

// NavLink is a labeled URL used in header and footer navigation.
type NavLink struct {
	Label    string `yaml:"label"`
	URL      string `yaml:"url"`
	External bool   `yaml:"external"`
}

// ---------------------------------------------------------------------------
// Custom Head Tags
// ---------------------------------------------------------------------------

type HeadSettings struct {
	Tags      []HeadTag `yaml:"tags"`
	CustomCSS []string  `yaml:"custom_css"`
	CustomJS  []string  `yaml:"custom_js"`
}

type HeadTag struct {
	Tag     string            `yaml:"tag"`
	Attrs   map[string]string `yaml:"attrs"`
	Content string            `yaml:"content"`
}

// ---------------------------------------------------------------------------
// Build Settings
// ---------------------------------------------------------------------------

type BuildSettings struct {
	Output   string `yaml:"output"`
	BasePath string `yaml:"base_path"`
	Clean    *bool  `yaml:"clean"`
	Sitemap  *bool  `yaml:"sitemap"`
	Minify   *bool  `yaml:"minify"`
	// LastUpdated selects the strategy for page "last updated" timestamps:
	//   "git"   — `git log -1` for the file, fall back to mtime on error
	//   "mtime" — filesystem modification time (default)
	//   "false" / "off" — disabled; no timestamp rendered
	// Legacy YAML bool form is accepted: true → "mtime", false → "false" (with deprecation warning).
	LastUpdated LastUpdatedStrategy `yaml:"last_updated"`
	Feed        *bool               `yaml:"feed"`
	Drafts      *bool               `yaml:"drafts"`
	Future      *bool               `yaml:"future"`
	Parallel    *bool               `yaml:"parallel"`
	Cache       *bool               `yaml:"cache"`
}

// LastUpdatedStrategy is a string enum with back-compat for the legacy bool form.
type LastUpdatedStrategy string

// UnmarshalYAML accepts strings ("git", "mtime", "false") and legacy bools
// (true → "mtime", false → "false") with a deprecation warning.
func (l *LastUpdatedStrategy) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode && value.Tag == "!!bool" {
		switch value.Value {
		case "true":
			*l = "mtime"
		case "false":
			*l = "false"
		default:
			return fmt.Errorf("build.last_updated: invalid bool %q", value.Value)
		}
		log.Printf("config: build.last_updated as bool is deprecated; use \"git\", \"mtime\", or \"false\"")
		return nil
	}
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	*l = LastUpdatedStrategy(s)
	return nil
}

// ---------------------------------------------------------------------------
// Markdown Settings
// ---------------------------------------------------------------------------

type MarkdownSettings struct {
	KaTeX               *bool                `yaml:"katex"`
	Mermaid             *bool                `yaml:"mermaid"`
	CDN                 *bool                `yaml:"cdn"`
	Unsafe              *bool                `yaml:"unsafe"`
	Typographer         *bool                `yaml:"typographer"`
	GithubAlerts        *bool                `yaml:"github_alerts"`
	TripleColonCallouts *bool                `yaml:"triple_colon_callouts"`
	TOC                 MarkdownTOCSettings  `yaml:"toc"`
	Highlighting        HighlightingSettings `yaml:"highlighting"`
}

type MarkdownTOCSettings struct {
	MinHeadingLevel int `yaml:"min_heading_level"`
	MaxHeadingLevel int `yaml:"max_heading_level"`
}

type HighlightingSettings struct {
	Style      string `yaml:"style"`
	LightTheme string `yaml:"light_theme"`
	DarkTheme  string `yaml:"dark_theme"`
	Theme      string `yaml:"theme"`
}

// ---------------------------------------------------------------------------
// Prefetch
// ---------------------------------------------------------------------------

type PrefetchSettings struct {
	Enabled  *bool  `yaml:"enabled"`
	Strategy string `yaml:"strategy"`
	Delay    int    `yaml:"delay"`
}

// ---------------------------------------------------------------------------
// Images
// ---------------------------------------------------------------------------

type ImageSettings struct {
	Widths      []int    `yaml:"widths"`
	Formats     []string `yaml:"formats"`
	Quality     int      `yaml:"quality"`
	Placeholder string   `yaml:"placeholder"`
	MaxWidth    int      `yaml:"max_width"`
	LazyLoading *bool    `yaml:"lazy_loading"`
	Dimensions  *bool    `yaml:"dimensions"`
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

type SearchSettings struct {
	Enabled  *bool  `yaml:"enabled"`
	Provider string `yaml:"provider"`
}

// ---------------------------------------------------------------------------
// Link Validation
// ---------------------------------------------------------------------------

type LinkValidationSettings struct {
	Enabled           *bool    `yaml:"enabled"`
	Level             string   `yaml:"level"`
	CheckAnchors      *bool    `yaml:"check_anchors"`
	CheckImages       *bool    `yaml:"check_images"`
	WarnRelativeLinks *bool    `yaml:"warn_relative_links"`
	CheckExternal     *bool    `yaml:"check_external"`
	Ignore            []string `yaml:"ignore"`
}

// ---------------------------------------------------------------------------
// Content Linting
// ---------------------------------------------------------------------------

type ContentLintSettings struct {
	Enabled *bool             `yaml:"enabled"`
	Rules   ContentLintRules  `yaml:"rules"`
}

type ContentLintRules struct {
	HeadingMaxLength     int      `yaml:"heading_max_length"`
	HeadingIncrement     *bool    `yaml:"heading_increment"`
	ImageAltRequired     *bool    `yaml:"image_alt_required"`
	NoEmptyLinks         *bool    `yaml:"no_empty_links"`
	FrontmatterRequired  []string `yaml:"frontmatter_required"`
}

// ---------------------------------------------------------------------------
// Analytics
// ---------------------------------------------------------------------------

type AnalyticsSettings struct {
	Provider string `yaml:"provider"`
	SiteID   string `yaml:"site_id"`
	Script   string `yaml:"script"`
}

// ---------------------------------------------------------------------------
// Deployment
// ---------------------------------------------------------------------------

type DeployConfig struct {
	Provider       string `yaml:"provider"`        // github, netlify, cloudflare, vercel, custom
	Branch         string `yaml:"branch"`          // GitHub Pages branch (default: gh-pages)
	SiteID         string `yaml:"site_id"`         // Netlify site ID
	ProjectName    string `yaml:"project_name"`    // Cloudflare Pages project name
	ProjectID      string `yaml:"project_id"`      // Vercel project ID
	Command        string `yaml:"command"`          // Custom deploy command
	RedirectFormat string `yaml:"redirect_format"` // html, netlify, vercel, all (default: all)
}

// ---------------------------------------------------------------------------
// Collections (site.yaml level)
// ---------------------------------------------------------------------------

type CollectionSiteConfig struct {
	Enabled   *bool                    `yaml:"enabled"`
	Path      string                   `yaml:"path"`
	URLPrefix string                   `yaml:"url_prefix"`
	Sort      string                   `yaml:"sort"`
	Layout    string                   `yaml:"layout"`
	Permalink string                   `yaml:"permalink"`
	Paginate  int                      `yaml:"paginate"`
	Feed      *bool                    `yaml:"feed"`
	Tabs      *bool                    `yaml:"tabs"`
	Sidebar   *CollectionSidebarConfig `yaml:"sidebar"`
	TOC       *CollectionTOCConfig     `yaml:"toc"`
	PrevNext  *CollectionPrevNextConfig `yaml:"prev_next"`
}

type CollectionSidebarConfig struct {
	Collapsible        *bool `yaml:"collapsible"`
	CollapsedByDefault *bool `yaml:"collapsed_by_default"`
	MaxDepth           int   `yaml:"max_depth"`
	Search             *bool `yaml:"search"`
}

type CollectionTOCConfig struct {
	Enabled         *bool `yaml:"enabled"`
	Depth           int   `yaml:"depth"`
	ScrollHighlight *bool `yaml:"scroll_highlight"`
}

type CollectionPrevNextConfig struct {
	Enabled *bool    `yaml:"enabled"`
	Labels  []string `yaml:"labels"`
}

// ---------------------------------------------------------------------------
// Taxonomy
// ---------------------------------------------------------------------------

// TaxonomyConfig holds per-taxonomy settings.
// Accepts both short form ("tag") and full form ({singular: "tag", paginate_by: 20}) in YAML.
type TaxonomyConfig struct {
	Singular   string `yaml:"singular"`
	PaginateBy int    `yaml:"paginate_by"`
}

func (tc *TaxonomyConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		tc.Singular = value.Value
		return nil
	}
	type raw TaxonomyConfig
	return value.Decode((*raw)(tc))
}

// ---------------------------------------------------------------------------
// Homepage
// ---------------------------------------------------------------------------

type HomepageSettings struct {
	Template string       `yaml:"template"`
	Hero     HeroSettings `yaml:"hero"`
}

type HeroSettings struct {
	Title      string   `yaml:"title"`
	Subtitle   string   `yaml:"subtitle"`
	CTA        *HeroCTA `yaml:"cta"`
	Background string   `yaml:"background"`
}

type HeroCTA struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
}

// ---------------------------------------------------------------------------
// Plugins
// ---------------------------------------------------------------------------

type PluginSettings struct {
	Enabled []string                  `yaml:"enabled"`
	Config  map[string]map[string]any `yaml:"config"`
}

// ---------------------------------------------------------------------------
// Server
// ---------------------------------------------------------------------------

type ServerSettings struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	LiveReload *bool  `yaml:"live_reload"`
}

// ---------------------------------------------------------------------------
// i18n
// ---------------------------------------------------------------------------

type I18nSettings struct {
	DefaultLanguage string                    `yaml:"default_language"`
	Languages       map[string]LanguageConfig `yaml:"languages"`
}

type LanguageConfig struct {
	Name   string `yaml:"name"`
	Title  string `yaml:"title"`
	Weight int    `yaml:"weight"`
	Dir    string `yaml:"dir"` // "ltr" or "rtl"
}

// IsMultiLang returns true when the site has multiple languages configured.
func (s *I18nSettings) IsMultiLang() bool {
	return len(s.Languages) > 0
}

// GetDefaultLanguage returns the configured default language code, or "en" if unset.
func (s *I18nSettings) GetDefaultLanguage() string {
	if s.DefaultLanguage != "" {
		return s.DefaultLanguage
	}
	return "en"
}

// LanguageCodes returns all configured language codes sorted by weight then alphabetically.
func (s *I18nSettings) LanguageCodes() []string {
	codes := make([]string, 0, len(s.Languages))
	for code := range s.Languages {
		codes = append(codes, code)
	}
	sort.Slice(codes, func(i, j int) bool {
		wi, wj := s.Languages[codes[i]].Weight, s.Languages[codes[j]].Weight
		if wi != wj {
			return wi < wj
		}
		return codes[i] < codes[j]
	})
	return codes
}

// ---------------------------------------------------------------------------
// Content
// ---------------------------------------------------------------------------

type ContentSettings struct {
	Dir           string `yaml:"dir"`
	SummaryLength int    `yaml:"summary_length"`
}

// ---------------------------------------------------------------------------
// llms.txt
// ---------------------------------------------------------------------------

type LlmsTxtSettings struct {
	Enabled     *bool `yaml:"enabled"`
	IncludeBlog *bool `yaml:"include_blog"`
}

// ---------------------------------------------------------------------------
// Security
// ---------------------------------------------------------------------------

// SecurityConfig holds security-related settings for content rendering.
type SecurityConfig struct {
	BlockedHrefSchemes []string `yaml:"blocked_href_schemes"`
}
