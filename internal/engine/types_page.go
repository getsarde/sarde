package engine

import (
	"html/template"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Page — embedded sub-structs
// ---------------------------------------------------------------------------

// PageIdentity holds the core identity fields of a page.
type PageIdentity struct {
	Title        string
	Slug         string
	Date         time.Time
	Updated      time.Time
	PublishDate  time.Time
	ExpiryDate   time.Time
	Permalink    string
	RelPermalink string
	Kind         NodeKind
	FilePath     string
	RelPath      string
}

// URL returns the resolved Permalink if set, otherwise RelPermalink.
// In a fully built site, Permalink is always set. This accessor exists
// for robustness in tests and edge cases where Permalink may be empty.
func (p *PageIdentity) URL() string {
	if p.Permalink != "" {
		return p.Permalink
	}
	return p.RelPermalink
}

// PageContent holds rendered content and content-derived metadata.
type PageContent struct {
	Content           template.HTML
	Summary           template.HTML
	RawContent        string
	ContentDigest     string // hex digest of raw file bytes (for incremental rebuild skip)
	FrontmatterDigest string // hex digest of serialized frontmatter map (body-only change detection)
	WordCount         int
	ReadingTime       int
	Headings          []Heading
	HasCodeBlocks     bool
	HasImages         bool
	FrontmatterLines  int
}

// PageMeta holds editorial metadata.
type PageMeta struct {
	Draft       bool
	Description string
	Image       string
}

// PageRelationships holds graph connections to other pages and structures.
type PageRelationships struct {
	Collection *Collection
	Section    *Section
	PrevPage   *Page
	NextPage   *Page
	Siblings   []*Page
	Backlinks  []*Page
}

// PageTaxonomy holds taxonomy membership fields.
type PageTaxonomy struct {
	Tags       []string
	Categories []string
	Aliases    []string
	Extra      map[string][]string
}

// PageSidebar holds sidebar presentation fields.
type PageSidebar struct {
	Order  int
	Label  string
	Hidden bool
	Attrs  map[string]string
	Badge  Badge
	Icon   string
}

// PageTOC holds per-page table-of-contents override fields.
type PageTOC struct {
	Enabled  *bool
	MinLevel int
	MaxLevel int
}

// PageI18n holds language and translation fields.
type PageI18n struct {
	Lang            string
	LangRelPath     string
	Translations    []*Page
	AllTranslations []*Page
	IsFallback      bool
}

// PageVersioning holds version membership fields.
type PageVersioning struct {
	Version        string
	VersionRelPath string
	VersionPeers   []*Page
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

// Page represents a single content page. Sub-structs are embedded so all
// fields remain accessible as top-level names (e.g. page.Title, page.Tags).
type Page struct {
	PageIdentity
	PageContent
	PageMeta
	PageRelationships
	PageTaxonomy
	Sidebar PageSidebar
	TOC     PageTOC
	PageI18n
	PageVersioning

	ShowTags *bool

	NavNode   *NavNode
	Resources []Resource
	Params    map[string]any
}

// ---------------------------------------------------------------------------
// Frontmatter
// ---------------------------------------------------------------------------

// FrontmatterIdentity holds core identity fields parsed from frontmatter.
type FrontmatterIdentity struct {
	Title       string   `yaml:"title"`
	Slug        string   `yaml:"slug"`
	Date        FlexDate `yaml:"date"`
	Updated     FlexDate `yaml:"updated"`
	PublishDate FlexDate `yaml:"publish_date"`
	ExpiryDate  FlexDate `yaml:"expiry_date"`
	Aliases     []string `yaml:"aliases"`
	Layout      string   `yaml:"layout"`
	Type        string   `yaml:"type"`
	Template    string   `yaml:"template"`
}

// FrontmatterMeta holds editorial and behavioral override fields.
type FrontmatterMeta struct {
	Draft       bool          `yaml:"draft"`
	Description string        `yaml:"description"`
	Image       string        `yaml:"image"`
	Summary     string        `yaml:"summary"`
	Render      *bool         `yaml:"render"`
	Pagefind    *bool         `yaml:"pagefind"`
	ShowUpdated *bool         `yaml:"show_updated"`
	EditURL     *EditURLValue `yaml:"edit_url"`
}

// FrontmatterSidebar holds sidebar presentation fields.
type FrontmatterSidebar struct {
	Order  int               `yaml:"order"`
	Label  string            `yaml:"label"`
	Hidden bool              `yaml:"hidden"`
	Attrs  map[string]string `yaml:"attrs"`
	Badge  Badge             `yaml:"badge"`
	Icon   string            `yaml:"icon"`
}

// FrontmatterTOC holds table-of-contents override fields.
//
// Supports two YAML forms:
//
//	toc: false                      → FrontmatterTOC{Enabled: ptr(false)}
//	toc:
//	  enabled: true
//	  min_level: 2                  → FrontmatterTOC{Enabled: ptr(true), MinLevel: 2}
type FrontmatterTOC struct {
	Enabled  *bool `yaml:"enabled"`
	MinLevel int   `yaml:"min_level"`
	MaxLevel int   `yaml:"max_level"`
}

func (t *FrontmatterTOC) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		var b bool
		if err := value.Decode(&b); err != nil {
			return err
		}
		t.Enabled = &b
		return nil
	}
	type plain FrontmatterTOC
	return value.Decode((*plain)(t))
}

// FrontmatterNav holds prev/next navigation override fields.
type FrontmatterNav struct {
	Prev *NavOverride `yaml:"prev"`
	Next *NavOverride `yaml:"next"`
}

// Frontmatter represents parsed frontmatter fields from a content file.
// Sub-structs are embedded so all fields remain accessible as top-level
// names (e.g. fm.Title, fm.Draft, fm.Sidebar.Label).
type Frontmatter struct {
	FrontmatterIdentity `yaml:",inline"`
	FrontmatterMeta     `yaml:",inline"`
	Sidebar             FrontmatterSidebar `yaml:"sidebar"`
	TOC                 FrontmatterTOC     `yaml:"toc"`
	FrontmatterNav      `yaml:",inline"`

	Tags               []string       `yaml:"tags"`
	Categories         []string       `yaml:"categories"`
	ShowTags           *bool          `yaml:"show_tags"`
	Transparent        bool           `yaml:"transparent"`
	Hero               *HeroConfig    `yaml:"hero"`
	Icon               string         `yaml:"icon"`
	Head               []HeadTag      `yaml:"head"`
	Banner             *PageBanner    `yaml:"banner"`
	Cascade            map[string]any `yaml:"cascade"`
	Params             map[string]any `yaml:"params"`
	LearningObjectives []string       `yaml:"learning_objectives"`
}

// HeroConfig defines hero section fields for splash layout pages.
type HeroConfig struct {
	Title   string       `yaml:"title"`
	Tagline string       `yaml:"tagline"`
	Image   *HeroImage   `yaml:"image"`
	Actions []HeroAction `yaml:"actions"`
}

// HeroImage defines the hero image with optional light/dark variants.
type HeroImage struct {
	Src   string `yaml:"src"`
	Light string `yaml:"light"`
	Dark  string `yaml:"dark"`
	Alt   string `yaml:"alt"`
}

// HeroAction defines a call-to-action button in the hero section.
type HeroAction struct {
	Text    string            `yaml:"text"`
	Link    string            `yaml:"link"`
	Variant string            `yaml:"variant"`
	Icon    string            `yaml:"icon"`
	Attrs   map[string]string `yaml:"attrs"`
}

// SanitizeAttrs strips event-handler attributes (on*) from all hero actions.
func (h *HeroConfig) SanitizeAttrs() {
	for i := range h.Actions {
		if len(h.Actions[i].Attrs) == 0 {
			continue
		}
		clean := make(map[string]string, len(h.Actions[i].Attrs))
		for k, v := range h.Actions[i].Attrs {
			if !strings.HasPrefix(strings.ToLower(k), "on") {
				clean[k] = v
			}
		}
		h.Actions[i].Attrs = clean
	}
}

// HeadTag defines a single injected <head> element from frontmatter.
type HeadTag struct {
	Tag     string            `yaml:"tag"`
	Attrs   map[string]string `yaml:"attrs"`
	Content string            `yaml:"content"`
}

// AllowedHeadTags lists the HTML tag names permitted in per-page head injection.
var AllowedHeadTags = map[string]bool{
	"meta": true, "link": true, "script": true,
	"style": true, "noscript": true, "base": true,
}
