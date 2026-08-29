---
title: Route Data
description: "Every field available to templates as the dot context, with the conditions under which each is populated"
sidebar:
  order: 6
---

Every template, component, and partial receives one `*RouteData` value as its dot context (`.`). This page lists every field on it and on the types reachable from it, in the order the Go structs declare them. The source of truth is `internal/engine/types_route.go` and `types_page.go`; a test in `internal/engine` fails when a field is added to the structs without an entry here.

`RouteData` and `Page` are built from embedded groups, and `html/template` promotes embedded fields, so address them flat: `.Lang`, not `.RouteI18n.Lang`; `.Page.Title`, not `.Page.PageIdentity.Title`.

## Reading this page

The `Set when` column on the `RouteData` tables says when a field holds a value. Everywhere else, assume the field is populated whenever its parent is. The conditions come from `BuildRouteData` in `internal/template/routedata.go`.

| Value | Meaning |
|-------|---------|
| always | Set on every render |
| collection | The page belongs to a collection (`.Collection` is non-nil) |
| sidebar layout | The layout is `docs`, `wide`, or `labs` |
| list page | The page is a collection `_index.md` |
| versioned | The collection has `versioning.enabled: true` |
| tabbed | Sidebar layout and the collection uses docs tabs |
| labs | The collection is a labs collection |
| plugins | Never set by the engine; plugins fill it in `BeforeRender` |

A nil pointer is false in `{{ if }}` but panics on field access, so guard optional pointers with `with`:

```html
{{ with .Paginator }}Page {{ .Current }} of {{ .Total }}{{ end }}
```

## `RouteData`

### Page and site

| Field | Type | Set when | Description |
|-------|------|----------|-------------|
| `.Page` | `*Page` | always | The page being rendered. See [Page](#page) |
| `.Collection` | `*Collection` | collection | The collection the page belongs to. See [Collection](#collection) |
| `.Site` | `*SiteContext` | always | Site-wide data. See [Site](#sitecontext-site) |
| `.Theme` | `*ThemeConfig` | always | Active theme metadata, tokens, and the pre-rendered style tag |
| `.Layout` | `LayoutType` | always | Resolved layout: the collection's layout, overridden by the page's `layout` frontmatter; forced to `default` on home, taxonomy, and term pages |
| `.Template` | `string` | always | Resolved template name. See [Template names](#template-names) |

### `RouteNav`

| Field | Type | Set when | Description |
|-------|------|----------|-------------|
| `.GlobalNav` | `*GlobalNav` | always | Header navigation built from collections and `header.links` |
| `.Sidebar` | `*NavTree` | sidebar layout | The sidebar tree for this page, with the active path marked. Tabbed collections get the active tab's tree; labs get the lab-scoped tree |
| `.SidebarType` | `string` | always | `"nav"` on sidebar layouts, `"none"` otherwise |
| `.Breadcrumbs` | `[]BreadcrumbItem` | sidebar layout | Trail from the collection root to the current page |
| `.Pagination` | `*PaginationLinks` | when a previous or next sibling exists | Prev/next links. Frontmatter `prev:` and `next:` can suppress, replace, or relabel either side |
| `.Paginator` | `*Paginator` | list page with `paginate > 0`; taxonomy term pages | Numbered pagination state |
| `.HasSidebar` | `bool` | sidebar layout | True on `docs`, `wide`, and `labs` |
| `.SidebarCollapsedByDefault` | `bool` | sidebar layout | From the collection's `sidebar.collapsed_by_default`; forced true when `collapse_level` is set |
| `.Section` | `*Section` | list page | The section this `_index.md` introduces |
| `.IsSection` | `bool` | list page | True for collection `_index.md` pages |

### `RouteI18n`

| Field | Type | Set when | Description |
|-------|------|----------|-------------|
| `.Lang` | `string` | always | Language code of this render: the page's language, else the site language, else `en` |
| `.Dir` | `string` | always | `ltr` or `rtl`, from the language's `dir` in `i18n.languages` |
| `.Translations` | `[]TranslationLink` | when non-empty | The same page in other languages |
| `.AllTranslations` | `[]TranslationLink` | when non-empty | Every configured language, with fallback entries for languages that lack this page. Feeds the language switcher |

### `RouteVersioning`

| Field | Type | Set when | Description |
|-------|------|----------|-------------|
| `.Version` | `string` | versioned | Version ID of the page |
| `.VersionLabel` | `string` | versioned | Display label of that version |
| `.Versions` | `[]VersionLink` | versioned | Every version, each pointing at the peer page or the version root |
| `.IsLatest` | `bool` | versioned | True when the page's version is the collection's `last_version` |
| `.VersionBanner` | `string` | versioned | `none`, `unmaintained`, or `unreleased` |

### `RouteTabs`

| Field | Type | Set when | Description |
|-------|------|----------|-------------|
| `.IsTabbed` | `bool` | tabbed | True when the collection renders docs tabs |
| `.DocsTabs` | `[]*DocsTab` | tabbed | Tabs in display order |
| `.ActiveTab` | `*DocsTab` | tabbed | The tab containing this page; falls back to the first tab |

### `RouteLabs`

| Field | Type | Set when | Description |
|-------|------|----------|-------------|
| `.LabNumber` | `int` | labs | Lab number, from the `labs_number` param |
| `.LabStepIndex` | `int` | labs | Position of this step within the lab |
| `.LabStepTotal` | `int` | labs | Number of steps in the lab |
| `.LabStepLabel` | `string` | labs with a `labs` config block | Word used for a step: `Lab` by default, or the configured `step_label` |
| `.LearningObjectives` | `[]string` | labs | From the `learning_objectives` frontmatter list |

### `RouteAssets`

| Field | Type | Set when | Description |
|-------|------|----------|-------------|
| `.Scripts` | `[]string` | plugins | Deferred external script URLs |
| `.Styles` | `[]string` | plugins | External stylesheet URLs |
| `.InlineScripts` | `[]template.JS` | plugins | Inline script bodies |
| `.ModuleScripts` | `[]string` | plugins | ES module script URLs |

### Extras

| Field | Type | Set when | Description |
|-------|------|----------|-------------|
| `.Homepage` | `*HomepageData` | home page, or a language-root `_index.md` | Homepage template and hero settings from `sarde.yaml` |
| `.Taxonomy` | `*Taxonomy` | taxonomy and term pages | The taxonomy being listed |
| `.TaxonomyTerm` | `*TaxonomyTerm` | term pages | The term being listed |
| `.TermEntries` | `[]*TermEntry` | taxonomy pages | Every term with counts, for tag clouds |
| `.PageBanner` | `*PageBanner` | when frontmatter has a `banner:` block | Per-page announcement banner |

### Template names

`.Template` follows this order: the page's `template` frontmatter wins when set; otherwise the value depends on the page kind.

| Page | `.Template` |
|------|-------------|
| Collection `_index.md` | `<collection>/list` (a labs section at lab depth gets `<collection>/single`) |
| Any other collection page | `<collection>/single` |
| Home page, or a language-root `_index.md` | `home` |
| Standalone section or page | `_default/single` |
| Taxonomy list page | `_taxonomy/list` |
| Taxonomy term page | `_taxonomy/term` |

The name is then resolved through the lookup chain in [Layouts and Templates](/customization/layouts-and-templates/#template-overlay-resolution).

## `Page`

`Page` embeds the groups below, so their fields are addressed as `.Page.<Field>`. `Sidebar` and `TOC` are named fields, addressed as `.Page.Sidebar.<Field>` and `.Page.TOC.<Field>`.

### `PageIdentity`

| Field | Type | Description |
|-------|------|-------------|
| `.Page.Title` | `string` | Page title, after inference from the first heading or filename |
| `.Page.Slug` | `string` | URL slug |
| `.Page.Date` | `time.Time` | Publish date. Zero when unknown; test with `.IsZero` |
| `.Page.Updated` | `time.Time` | Last-updated date |
| `.Page.PublishDate` | `time.Time` | Scheduled publish date |
| `.Page.ExpiryDate` | `time.Time` | Expiry date |
| `.Page.Permalink` | `string` | Final URL of the page; use this in links |
| `.Page.RelPermalink` | `string` | Root-relative form of the URL |
| `.Page.Kind` | `NodeKind` | `home`, `section`, `page`, `bundle`, `standalone`, `taxonomy`, or `term` |
| `.Page.FilePath` | `string` | Source file path |
| `.Page.RelPath` | `string` | Source path relative to `content/` |

A `time.Time` is never false in `{{ if }}`, so guard dates explicitly:

```html
{{ if not .Page.Date.IsZero }}<time datetime="{{ .Page.Date.Format "2006-01-02" }}">{{ .Page.Date.Format "January 2, 2006" }}</time>{{ end }}
```

### `PageContent`

| Field | Type | Description |
|-------|------|-------------|
| `.Page.Content` | `template.HTML` | Rendered body |
| `.Page.Summary` | `template.HTML` | Rendered summary |
| `.Page.RawContent` | `string` | Markdown source of the body |
| `.Page.WordCount` | `int` | Words in the body |
| `.Page.ReadingTime` | `int` | Estimated reading time in minutes |
| `.Page.Headings` | `[]Heading` | Headings extracted for the table of contents |
| `.Page.HasCodeBlocks` | `bool` | True when the body contains a fenced code block |
| `.Page.HasImages` | `bool` | True when the body contains an image |
| `.Page.FrontmatterLines` | `int` | Height of the frontmatter block, for line-number offsets |

Internal, not for templates: `ContentDigest` and `FrontmatterDigest` are hashes used by incremental rebuilds.

### `PageMeta`

| Field | Type | Description |
|-------|------|-------------|
| `.Page.Draft` | `bool` | True for `draft: true` pages |
| `.Page.Description` | `string` | Page description |
| `.Page.Image` | `string` | Cover image from the `image` frontmatter field |
| `.Page.DateExplicit` | `bool` | True when `Date` came from frontmatter or a `YYYY-MM-DD` filename prefix rather than file modification time. Check it before presenting a date as editorial content |

### `PageRelationships`

| Field | Type | Description |
|-------|------|-------------|
| `.Page.Collection` | `*Collection` | Owning collection |
| `.Page.Section` | `*Section` | Owning section |
| `.Page.PrevPage` | `*Page` | Previous page in collection order |
| `.Page.NextPage` | `*Page` | Next page in collection order |
| `.Page.Siblings` | `[]*Page` | Pages in the same section |
| `.Page.Backlinks` | `[]*Page` | Pages that link to this one |

### `PageTaxonomy`

| Field | Type | Description |
|-------|------|-------------|
| `.Page.Tags` | `[]string` | Tags |
| `.Page.Categories` | `[]string` | Categories |
| `.Page.Aliases` | `[]string` | Redirect aliases |
| `.Page.Extra` | `map[string][]string` | Custom taxonomies keyed by name, for example `authors` |

### `PageSidebar` (`.Page.Sidebar`)

Sidebar presentation settings for this page. This is not the navigation tree; that is `.Sidebar` on `RouteData`.

| Field | Type | Description |
|-------|------|-------------|
| `.Page.Sidebar.Order` | `int` | Sort order |
| `.Page.Sidebar.Label` | `string` | Label override |
| `.Page.Sidebar.Hidden` | `bool` | Hidden from the sidebar |
| `.Page.Sidebar.Attrs` | `map[string]string` | Extra attributes on the sidebar link |
| `.Page.Sidebar.Badge` | `Badge` | Badge next to the label |
| `.Page.Sidebar.Icon` | `string` | Icon name |

### `PageTOC` (`.Page.TOC`)

| Field | Type | Description |
|-------|------|-------------|
| `.Page.TOC.Enabled` | `*bool` | Per-page override; nil means inherit |
| `.Page.TOC.MinLevel` | `int` | Lowest heading level to include |
| `.Page.TOC.MaxLevel` | `int` | Highest heading level to include |

### `PageI18n`

| Field | Type | Description |
|-------|------|-------------|
| `.Page.Lang` | `string` | Language code |
| `.Page.LangRelPath` | `string` | Source path relative to the language root |
| `.Page.Translations` | `[]*Page` | The same page in other languages |
| `.Page.AllTranslations` | `[]*Page` | All language variants including fallbacks |
| `.Page.IsFallback` | `bool` | True when this render is a fallback copy from the default language |

### `PageVersioning`

| Field | Type | Description |
|-------|------|-------------|
| `.Page.Version` | `string` | Version ID |
| `.Page.VersionRelPath` | `string` | Source path relative to the version root |
| `.Page.VersionPeers` | `[]*Page` | The same page in other versions |

### Direct fields and methods

| Field | Type | Description |
|-------|------|-------------|
| `.Page.ShowTags` | `*bool` | Per-page `show_tags` override; nil means inherit |
| `.Page.NavNode` | `*NavNode` | This page's node in the sidebar tree |
| `.Page.Resources` | `[]Resource` | Files bundled with the page |
| `.Page.Params` | `map[string]any` | The `params` frontmatter block plus engine-set keys |

| Method | Returns | Description |
|--------|---------|-------------|
| `.Page.URL` | `string` | `Permalink` when set, otherwise `RelPermalink` |
| `.Page.ShowUpdated` | `bool` | Whether to show the updated date, from `show_updated` frontmatter (default true). Display only; `Updated` is always resolved |

## `SiteContext` (`.Site`)

| Field | Type | Description |
|-------|------|-------------|
| `.Site.Title` | `string` | Site title |
| `.Site.BaseURL` | `string` | Absolute base URL |
| `.Site.BasePath` | `string` | Normalized base path: `/docs/` or `/` |
| `.Site.SiteID` | `string` | Site identifier |
| `.Site.Language` | `string` | Default language code |
| `.Site.Generator` | `string` | Generator string for the `meta` tag |
| `.Site.Favicon` | `string` | Favicon URL |
| `.Site.FaviconType` | `string` | Favicon MIME type |
| `.Site.Logo` | `LogoContext` | Resolved site logo |
| `.Site.SitemapEnabled` | `bool` | True when the sitemap plugin is on |
| `.Site.Config` | `any` | The full `sarde.yaml`, a `*config.SiteConfig` at runtime. See below |
| `.Site.Collections` | `map[string]*Collection` | Collections by name |
| `.Site.Taxonomies` | `map[string]*Taxonomy` | Taxonomies by name |
| `.Site.TaxonomiesByLang` | `map[string]map[string]*Taxonomy` | Taxonomies per language |
| `.Site.Pages` | `[]*Page` | Every page in the site |
| `.Site.Data` | `map[string]any` | Contents of the `data/` directory |
| `.Site.BuildTime` | `time.Time` | Build timestamp |
| `.Site.Languages` | `[]Language` | Configured languages |
| `.Site.DefaultLang` | `string` | Default language code |
| `.Site.EditURL` | `string` | Base URL for "edit this page" links |
| `.Site.KazariScriptURL` | `string` | URL of the Kazari interaction script |
| `.Site.IconLicenses` | `[]IconLicense` | License metadata of loaded icon sets, for a credits page |

`.Site.Config` is typed `any` in Go to avoid an import cycle, and holds the parsed configuration at runtime. Templates read it directly, so `{{ .Site.Config.Footer.Text }}` works, but a misspelled section fails at render time rather than at config load. The sections the default theme reads:

| Path | Configuration section |
|------|-----------------------|
| `.Site.Config.Site` | [`site`](/reference/configuration/#site) |
| `.Site.Config.Social` | [`social`](/reference/configuration/#social) |
| `.Site.Config.Header` | [`header`](/reference/configuration/#header) |
| `.Site.Config.Footer` | [`footer`](/reference/configuration/#footer) |
| `.Site.Config.Theme` | [`theme`](/reference/configuration/#theme) |
| `.Site.Config.Search` | [`search`](/reference/configuration/#search) |
| `.Site.Config.Analytics` | [`analytics`](/reference/configuration/#analytics) |
| `.Site.Config.Homepage` | [`homepage`](/reference/configuration/#homepage) |

Every other top-level key in [Configuration](/reference/configuration/) is reachable the same way, by its section name in PascalCase.

### `LogoContext` and `LogoImage`

| Field | Type | Description |
|-------|------|-------------|
| `.Site.Logo.Light` | `LogoImage` | Light-mode image |
| `.Site.Logo.Dark` | `LogoImage` | Dark-mode image |
| `.Site.Logo.Alt` | `string` | Alt text |
| `.Site.Logo.ReplacesTitle` | `bool` | Hide the text title and show only the logo |
| `.Site.Logo.Single` | `bool` | One image serves both themes |

`LogoImage` has `URL`, `Width`, and `Height`. Width and height are 0 for SVG logos and whenever the dimensions could not be probed.

### `Language`

| Field | Type | Description |
|-------|------|-------------|
| `Code` | `string` | Language code |
| `Name` | `string` | Display name |
| `Dir` | `string` | `ltr` or `rtl` |
| `Weight` | `int` | Sort weight |

### `IconLicense`

| Field | Type | Description |
|-------|------|-------------|
| `Prefix` | `string` | Icon set prefix |
| `Title` | `string` | Icon set name |
| `SPDX` | `string` | SPDX license identifier |
| `URL` | `string` | License URL |

## `Collection`

| Field | Type | Description |
|-------|------|-------------|
| `.Collection.Name` | `string` | Directory name |
| `.Collection.Title` | `string` | Display title |
| `.Collection.Config` | `*CollectionConfig` | Resolved collection settings |
| `.Collection.Pages` | `[]*Page` | Every page, already sorted |
| `.Collection.Featured` | `[]*Page` | Pages with `featured: true` |
| `.Collection.Sections` | `[]*Section` | Top-level sections |
| `.Collection.NavTree` | `*NavTree` | Default-language sidebar tree |
| `.Collection.NavTrees` | `map[string]*NavTree` | Sidebar tree per language |
| `.Collection.IndexPage` | `*Page` | The collection's `_index.md` |
| `.Collection.IsTabbed` | `bool` | True when docs tabs are detected or forced |
| `.Collection.Tabs` | `[]*DocsTab` | Tabs ordered by weight, then title |
| `.Collection.Versioning` | `*VersionConfig` | Versioning settings; nil when off |
| `.Collection.LabNavTrees` | `map[string]*NavTree` | Lab-scoped trees keyed by lab section permalink |
| `.Collection.IsMultiCourse` | `bool` | True when a labs collection groups labs by course |

Internal, not for templates: `CompositeNavTrees` and `CompositeTabSets` are keyed by a language-and-version key that templates cannot compute.

### `CollectionConfig`

| Field | Type | Description |
|-------|------|-------------|
| `SortBy` | `string` | Sort field |
| `SortOrder` | `string` | `asc` or `desc` |
| `Layout` | `LayoutType` | Default layout for the collection |
| `Permalink` | `string` | Permalink pattern |
| `Paginate` | `int` | Items per list page; 0 disables |
| `Feed` | `bool` | Generate a feed |
| `Sidebar` | `*SidebarConfig` | Sidebar settings |
| `TOC` | `*TOCConfig` | Table of contents settings |
| `PrevNext` | `*PrevNextConfig` | Prev/next link settings |
| `Tabs` | `*bool` | nil auto-detects, true forces, false disables tabs |
| `Versioning` | `*VersionConfig` | nil when the collection is not versioned |
| `Labs` | `*LabsConfig` | nil unless this is a labs collection |

### `SidebarConfig`

| Field | Type | Description |
|-------|------|-------------|
| `Collapsible` | `bool` | Groups can be collapsed |
| `CollapsedByDefault` | `bool` | Groups start collapsed |
| `MaxDepth` | `int` | Deepest level shown |
| `Search` | `bool` | Show the sidebar search box |
| `CollapseLevel` | `int` | When above 0, groups at this depth or shallower start open and deeper ones collapsed; 0 leaves `CollapsedByDefault` in charge |
| `Overrides` | `map[string]*SidebarOverride` | `sidebar.yaml` node overrides by collection-relative path |
| `TabOverrides` | `map[string]*TabOverride` | `sidebar.yaml` tab overrides by slug |

### `TOCConfig`, `PrevNextConfig`, `LabsConfig`

| Type | Fields |
|------|--------|
| `TOCConfig` | `Enabled`, `MinLevel`, `MaxLevel`, `ScrollHighlight` |
| `PrevNextConfig` | `Enabled`, and `Labels`, a two-element array holding the previous and next labels |
| `LabsConfig` | `StepLabel`, the word used for a step (`Lab` by default) |

### `DocsTab`

| Field | Type | Description |
|-------|------|-------------|
| `Title` | `string` | Tab label |
| `Description` | `string` | Tab description |
| `Icon` | `string` | Emoji, icon name, or SVG path |
| `Slug` | `string` | Directory name, used for URL prefix matching |
| `Order` | `int` | Sort order |
| `Permalink` | `string` | URL of the tab's index page |
| `Section` | `*Section` | The top-level section backing this tab |
| `NavTree` | `*NavTree` | Default-language tree for the tab |
| `NavTrees` | `map[string]*NavTree` | Tree per language |
| `Pages` | `[]*Page` | Pages in the tab |
| `IndexPage` | `*Page` | The tab's `_index.md` |

### `Section`

| Field | Type | Description |
|-------|------|-------------|
| `Title` | `string` | Section title |
| `Slug` | `string` | Directory slug |
| `Permalink` | `string` | Section URL |
| `Pages` | `[]*Page` | Pages directly in the section |
| `Sections` | `[]*Section` | Child sections |
| `IndexPage` | `*Page` | The section's `_index.md`, if any |
| `Parent` | `*Section` | Parent section; nil at the root |
| `Collection` | `*Collection` | Owning collection |
| `Transparent` | `bool` | Children render as if they were in the parent |
| `Render` | `bool` | The section has its own page |

`Parent` and `Collection` point back up the tree. A template that recurses through them without a stopping condition never terminates.

### `VersionConfig` and `VersionDef`

| Field | Type | Description |
|-------|------|-------------|
| `Enabled` | `bool` | Versioning is on |
| `LastVersion` | `string` | Version ID served at the root URL |
| `PublishLatestAtVersionURL` | `bool` | Also publish the latest version under its own prefix |
| `Versions` | `[]VersionDef` | Declared versions |

`VersionDef` has `ID`, `Label`, `Path` (URL segment, defaults to `ID`), `Banner` (`none`, `unmaintained`, or `unreleased`), and `Redirect` (`same-page` or `root`).

## Navigation types

### `NavTree`

| Field | Type | Description |
|-------|------|-------------|
| `Root` | `*NavNode` | Invisible root whose `Children` are the top-level entries |
| `Flat` | `[]*NavNode` | Every node in document order |
| `TotalPages` | `int` | Number of page nodes |
| `MaxDepth` | `int` | Deepest nesting level |

Internal, not for templates: `Hash` is a cache key.

### `NavNode`

| Field | Type | Description |
|-------|------|-------------|
| `Label` | `string` | Link text |
| `URL` | `string` | Link target |
| `Slug` | `string` | Node slug |
| `Order` | `int` | Sort order |
| `Position` | `int` | Position among siblings |
| `Children` | `[]*NavNode` | Child nodes |
| `Parent` | `*NavNode` | Parent node; nil for the root |
| `Depth` | `int` | Nesting level, root is 0 |
| `IsActive` | `bool` | This node is the current page |
| `IsOpen` | `bool` | Render the group expanded |
| `HasActive` | `bool` | A descendant is the current page |
| `DefaultOpen` | `bool` | The group starts expanded regardless of the active path |
| `GroupIndex` | `int` | Index used to persist collapse state |
| `Page` | `*Page` | The page behind the node, if any |
| `Attrs` | `map[string]string` | Extra link attributes |
| `Icon` | `string` | Icon name |
| `Badge` | `Badge` | Badge next to the label |
| `Description` | `string` | Description text |

Walk `Children` downward only. `Parent` and `Page.NavNode` form cycles.

```html
{{ define "navlist" }}
<ul>
  {{ range .Children }}
  <li{{ if .IsActive }} class="active"{{ else if .HasActive }} class="open"{{ end }}>
    <a href="{{ .URL }}">{{ .Label }}</a>
    {{ if .Children }}{{ template "navlist" . }}{{ end }}
  </li>
  {{ end }}
</ul>
{{ end }}
{{ with .Sidebar }}{{ template "navlist" .Root }}{{ end }}
```

→ A nested list of every sidebar entry, with the current page marked `active` and its ancestors marked `open`.

### `GlobalNav` and `GlobalNavItem`

`GlobalNav` has one field, `Items`, a slice of `GlobalNavItem`.

| Field | Type | Description |
|-------|------|-------------|
| `Label` | `string` | Link text |
| `URL` | `string` | Link target |
| `Collection` | `string` | Collection name the item represents, if any |
| `IsActive` | `bool` | The current page is inside this item |
| `External` | `bool` | The target is off-site |

### `BreadcrumbItem`

| Field | Type | Description |
|-------|------|-------------|
| `Label` | `string` | Crumb text |
| `URL` | `string` | Crumb target |
| `Current` | `bool` | This crumb is the current page |

### `PaginationLinks` and `PaginationLink`

`PaginationLinks` has `Prev` and `Next`, each a `*PaginationLink` with `URL` and `Title`. Either side may be nil.

### `Paginator`

| Field | Type | Description |
|-------|------|-------------|
| `Pages` | `[]PaginationLink` | Numbered links, one per page of results |
| `CurrentPages` | `[]*Page` | Content pages visible on this pagination page |
| `Current` | `int` | 1-based index of the current page |
| `Total` | `int` | Total number of pagination pages |
| `HasPrev` | `bool` | A previous page exists |
| `HasNext` | `bool` | A next page exists |
| `PrevURL` | `string` | URL of the previous page |
| `NextURL` | `string` | URL of the next page |
| `TotalItems` | `int` | Content items across all pages |
| `BaseURL` | `string` | Collection base URL for building custom links |
| `FirstURL` | `string` | URL of the first page |
| `LastURL` | `string` | URL of the last page |

`Pages` holds the numbered links, not the content. The content for the current page is `CurrentPages`. Page 1 lives at the collection URL and page N at `<collection>/page/N/`.

```html
{{ $pages := .Collection.Pages }}
{{ with .Paginator }}{{ $pages = .CurrentPages }}{{ end }}
{{ range $pages }}
<article><a href="{{ .Permalink }}">{{ .Title }}</a></article>
{{ end }}
{{ with .Paginator }}
<nav>
  {{ if .HasPrev }}<a href="{{ .PrevURL }}">Newer</a>{{ end }}
  <span>{{ .Current }} / {{ .Total }}</span>
  {{ if .HasNext }}<a href="{{ .NextURL }}">Older</a>{{ end }}
</nav>
{{ end }}
```

→ On a paginated list, the current slice of posts and prev/next links; on an unpaginated list, every post and no nav.

## Taxonomy types

### `Taxonomy`

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Plural name, for example `tags` |
| `Singular` | `string` | Singular name |
| `Terms` | `map[string]*TaxonomyTerm` | Terms by slug |
| `Permalink` | `string` | URL of the taxonomy list page |
| `PaginateBy` | `int` | Items per term page; 0 means no pagination |

### `TaxonomyTerm`

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Term as written in frontmatter |
| `Slug` | `string` | URL slug |
| `CustomSlug` | `string` | Slug from the `permalink` field in `data/*.yml`, overriding the generated one |
| `Permalink` | `string` | URL of the term page |
| `Pages` | `[]*Page` | Pages carrying the term |
| `Label` | `string` | Display label from `data/*.yml` |
| `Description` | `string` | Description from `data/*.yml` |
| `Color` | `string` | Color from `data/*.yml` |
| `Icon` | `string` | Icon from `data/*.yml` |
| `Hidden` | `bool` | Hide from listings |
| `Priority` | `int` | Sort priority |
| `Difficulty` | `string` | `beginner`, `intermediate`, or `advanced` |
| `ContentType` | `string` | `lecture`, `lab`, `assignment`, `project`, `reference`, `tutorial`, or `assessment` |

### `TermEntry`

Embeds `*TaxonomyTerm`, so every field above (`Name`, `Slug`, `CustomSlug`, `Permalink`, `Pages`, `Label`, `Description`, `Color`, `Icon`, `Hidden`, `Priority`, `Difficulty`, `ContentType`) is available directly, plus:

| Field | Type | Description |
|-------|------|-------------|
| `Count` | `int` | Number of pages carrying the term |
| `PopTier` | `int` | Popularity quintile from 1 to 5, for tag-cloud sizing |

## Homepage types

### `HomepageData` and `HeroData`

`HomepageData` has `Template` (the homepage template name) and `Hero`.

| Field | Type | Description |
|-------|------|-------------|
| `.Homepage.Hero.Eyebrow` | `string` | Small text above the title |
| `.Homepage.Hero.Title` | `string` | Hero title |
| `.Homepage.Hero.Subtitle` | `string` | Hero subtitle |
| `.Homepage.Hero.CTA` | `*HeroCTAData` | Primary button |
| `.Homepage.Hero.SecondaryCTA` | `*HeroCTAData` | Secondary button |
| `.Homepage.Hero.Stats` | `[]HeroStatData` | Proof points |
| `.Homepage.Hero.Code` | `*HeroCodeData` | Code sample panel |
| `.Homepage.Hero.Image` | `*HeroImageData` | Image or SVG panel |
| `.Homepage.Hero.Background` | `string` | Background style |

### `HeroCTAData`, `HeroStatData`, `HeroCodeData`, `HeroImageData`

| Type | Fields |
|------|--------|
| `HeroCTAData` | `Label`, `URL`, `Icon` |
| `HeroStatData` | `Value`, `Label` |
| `HeroCodeData` | `Title`, `Language`, `Body` |
| `HeroImageData` | `Src`, `Light`, `Dark`, `Alt`, `HTML` (inline SVG as `template.HTML`) |

## Leaf types

### `TranslationLink`

| Field | Type | Description |
|-------|------|-------------|
| `Lang` | `string` | Language code |
| `Name` | `string` | Display name, for example `Français`; falls back to the code |
| `Dir` | `string` | `ltr` or `rtl` |
| `URL` | `string` | URL of the page in that language |
| `Title` | `string` | Title of the page in that language |
| `IsFallback` | `bool` | The page does not exist in that language; the link goes to the fallback |

```html
{{ if .AllTranslations }}
<ul>
  {{ range .AllTranslations }}
  <li><a href="{{ .URL }}" hreflang="{{ .Lang }}" dir="{{ .Dir }}"{{ if eq .Lang $.Lang }} aria-current="true"{{ end }}>{{ .Name }}</a></li>
  {{ end }}
</ul>
{{ end }}
```

→ One link per configured language, with the current one marked.

### `VersionLink`

| Field | Type | Description |
|-------|------|-------------|
| `ID` | `string` | Version ID |
| `Label` | `string` | Display label |
| `URL` | `string` | Peer page or version root, depending on the redirect strategy |
| `Title` | `string` | Title of the target page |
| `IsCurrent` | `bool` | This is the version being rendered |
| `IsLatest` | `bool` | This is the `last_version` |
| `Banner` | `string` | `none`, `unmaintained`, or `unreleased` |
| `Redirect` | `string` | `same-page` or `root` |

### `Heading`

| Field | Type | Description |
|-------|------|-------------|
| `Level` | `int` | Heading level, 1 to 6 |
| `ID` | `string` | Anchor ID |
| `Text` | `string` | Heading text |

### `Resource`

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | File name |
| `Title` | `string` | Title |
| `MediaType` | `string` | MIME type |
| `RelPermalink` | `string` | URL of the file |
| `Width` | `int` | Image width, 0 when unknown |
| `Height` | `int` | Image height, 0 when unknown |

Internal, not for templates: `SrcPath` is the absolute path on the build machine.

### `Badge`

| Field | Type | Description |
|-------|------|-------------|
| `Text` | `string` | Badge text |
| `Variant` | `BadgeVariant` | `default`, `note`, `tip`, `success`, `caution`, or `danger` |

| Method | Returns | Description |
|--------|---------|-------------|
| `IsEmpty` | `bool` | No text set |
| `CSSClass` | `string` | `sarde-badge-<variant>`, for example `sarde-badge-tip` |

### `PageBanner`

| Field | Type | Description |
|-------|------|-------------|
| `Content` | `string` | Banner text |
| `Variant` | `string` | `note`, `tip`, `caution`, or `danger`; defaults to `note` |
| `Icon` | `string` | Lucide icon name overriding the variant's default |

### `ThemeConfig` (`.Theme`)

| Field | Type | Description |
|-------|------|-------------|
| `Name` | `string` | Theme name |
| `Slug` | `string` | Theme slug |
| `Version` | `string` | Theme version |
| `Author` | `string` | Theme author |
| `Tokens` | `map[string]string` | Resolved light-mode tokens |
| `DarkTokens` | `map[string]string` | Resolved dark-mode tokens |
| `DarkEnabled` | `bool` | Dark mode is on |
| `StyleTag` | `template.HTML` | Pre-rendered `<style>` block with the token custom properties |

### `LayoutType` and `NodeKind`

`.Layout` is one of `default`, `docs`, `splash`, `wide`, `full`, `centered`, `split`, `presentation`, or `labs`. Sidebar layouts are `docs`, `wide`, and `labs`; table-of-contents layouts are `docs` and `labs`.

`.Page.Kind` is one of `home`, `section`, `page`, `bundle`, `standalone`, `taxonomy`, or `term`.
