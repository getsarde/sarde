---
title: UI Components
description: "Reference for named template components that render layout chrome, and how their three-layer overlay resolves"
sidebar:
  order: 7
---

Components are named template slots that render layout chrome (headers, sidebars, footers, toggles). Call them from layout templates with `{{ component "Name" . }}`. They are distinct from [shortcodes](/reference/shortcodes) (invoked from Markdown content) and [extensions](/extensions/using-extensions) (`:::` Markdown syntax).

```html
{{ component "Header" . }}
```

The `Header` component renders the site title, navigation links, search trigger, version switcher, language switcher, social links, center toggle, and theme toggle. Override any component by placing a same-named `.html` file in `layouts/components/`.

## Calling a component

Pass the name as a string and the current route data (`.`) as the second argument:

```html
{{ component "Footer" . }}
```

Each mechanism handles unknown names differently:

| Mechanism | Invoked from | Unknown name behavior |
|-----------|--------------|----------------------|
| `component` | Layout templates | Empty string, no error |
| `partial` | Layout templates | Error |
| Shortcode | Markdown content | Warning, raw text preserved |

A typo in a component name silently renders nothing. Use [`{{ dump (component "Name" .) }}`](/reference/template-functions#debug-and-data) during development to verify a component produces output.

## Overriding components

Components resolve through three layers. Each layer fully replaces same-named components from the layer below.

| Priority | Layer | Directory |
|----------|-------|-----------|
| 1 (lowest) | Embedded | Compiled into the binary (27 default components) |
| 2 | Theme | `themes/<name>/layouts/components/` |
| 3 (highest) | Project | `layouts/components/` |

Filenames are case-sensitive and use PascalCase: `Header.html`, not `header.html`. The filename minus `.html` must match the component name exactly.

:::caution
Overriding a component that calls sub-components (like Header) means those sub-component calls are lost unless the override re-calls them. For example, overriding `Header.html` without including `{{ component "Search" . }}` removes the search trigger from the header entirely.
:::

## RouteData

Every component receives the current `RouteData` as `.`. Fields are organized into embedded sub-structs in Go, but templates address all fields by flat name.

### Page and site

| Field | Type | Description |
|-------|------|-------------|
| `.Page` | `*Page` | Current page (Title, Slug, Content, Tags, Headings, etc.) |
| `.Collection` | `*Collection` | Collection this page belongs to (nil for standalone pages) |
| `.Site` | `*SiteContext` | Site-wide context (Title, BaseURL, Config, Collections, etc.) |
| `.Theme` | `*ThemeConfig` | Active theme (name, tokens, StyleTag) |
| `.Layout` | `LayoutType` | Layout type (default, docs, splash, wide, full, centered, split, presentation) |
| `.Template` | `string` | Resolved template name |

### Navigation

| Field | Type | Description |
|-------|------|-------------|
| `.GlobalNav` | `*GlobalNav` | Top navigation items from `header.links` config |
| `.Sidebar` | `*NavTree` | Docs sidebar navigation tree |
| `.SidebarType` | `string` | Sidebar strategy (none, auto, manual) |
| `.Breadcrumbs` | `[]BreadcrumbItem` | Breadcrumb trail items |
| `.Pagination` | `*PaginationLinks` | Prev/next page links |
| `.Paginator` | `*Paginator` | Numbered pagination for section list pages |
| `.HasSidebar` | `bool` | Whether the layout includes a sidebar |
| `.SidebarCollapsedByDefault` | `bool` | Whether sidebar sections start collapsed |
| `.Section` | `*Section` | Current section (for `_index.md` pages) |
| `.IsSection` | `bool` | Whether the current page is a section index |

### i18n

| Field | Type | Description |
|-------|------|-------------|
| `.Lang` | `string` | Current rendering language code |
| `.Dir` | `string` | Text direction (`ltr` or `rtl`) |
| `.Translations` | `[]TranslationLink` | Same-page translations |
| `.AllTranslations` | `[]TranslationLink` | All available language links (feeds LanguageSwitcher) |

### Versioning

| Field | Type | Description |
|-------|------|-------------|
| `.Version` | `string` | Current version ID |
| `.VersionLabel` | `string` | Current version display label |
| `.Versions` | `[]VersionLink` | All version links (feeds VersionSwitcher) |
| `.IsLatest` | `bool` | Whether the current version is the latest |
| `.VersionBanner` | `string` | Version notice type: `""`, `"unmaintained"`, or `"unreleased"` |

### Tabs and assets

| Field | Type | Description |
|-------|------|-------------|
| `.IsTabbed` | `bool` | Whether the collection uses tabs |
| `.DocsTabs` | `[]*DocsTab` | Tab definitions (Title, Slug, Icon, Permalink) |
| `.ActiveTab` | `*DocsTab` | Currently active tab |
| `.Scripts` | `[]string` | Deferred external script URLs |
| `.Styles` | `[]string` | External stylesheet URLs |
| `.InlineScripts` | `[]template.JS` | Raw inline script content |
| `.ModuleScripts` | `[]string` | ES module script URLs |

### Extras

| Field | Type | Description |
|-------|------|-------------|
| `.Homepage` | `*HomepageData` | Homepage template data (hero, catalog, etc.) |
| `.Taxonomy` | `*Taxonomy` | Current taxonomy (on taxonomy list pages) |
| `.TaxonomyTerm` | `*TaxonomyTerm` | Current term (on term pages) |
| `.TermEntries` | `[]*TermEntry` | Term page entries |
| `.PageBanner` | `*PageBanner` | Frontmatter-driven page banner (Content, Variant, Icon) |

:::note
These are Go embedded structs, but `html/template` promotes embedded fields automatically. Always use the flat name (`.GlobalNav`, `.Version`), not the struct prefix (`.RouteNav.GlobalNav`).
:::

## Component reference

All 27 built-in components, alphabetically.

| Name | Purpose | Key data | Called by |
|------|---------|----------|----------|
| Breadcrumbs | Breadcrumb trail with home icon | `.Breadcrumbs` | `_docs/baseof.html` |
| CenterToggle | Centered/wide content width toggle | Static | Header |
| ContentPanel | Content area wrapper | `.Page.Content` | *Not called by default* |
| DocsTabSwitcher | Mobile docs tab dropdown | `.IsTabbed`, `.DocsTabs`, `.ActiveTab` | Sidebar |
| EditLink | "Edit this page" link | `.Page.Params`, `.Site.EditURL` | `_docs/baseof.html`, blog singles, `_default/single.html` |
| FallbackNotice | i18n fallback content notice | `.Page.IsFallback` | Both baseof templates |
| Footer | Site footer with links and credits | `.Site.Config.Footer.*` | Both baseof templates |
| GlobalNav | Top navigation bar | `.GlobalNav.Items` | Header |
| Head | `<head>` content (meta, styles, scripts) | `.Site.*`, `.Page.*`, `.Styles` | Both baseof templates |
| Header | Header chrome, composes 8 sub-components | Multiple | Both baseof templates |
| LanguageSwitcher | Language dropdown | `.AllTranslations`, `.Lang` | Header |
| LastUpdated | "Last updated" byline | `.Page.Updated`, `theme.date_format` | docs, labs, blog singles, `_default/single.html` |
| MobileTableOfContents | Mobile collapsible ToC with progress ring | `.Page.Headings` | `_docs/baseof.html` |
| PageBanner | Frontmatter-driven page banner | `.PageBanner` | Both baseof templates |
| PageTags | Tag chips with taxonomy links | `.Page.Tags` | `_docs/baseof.html`, blog singles |
| PageTitle | `<h1>` with optional icon and description | `.Page.Title`, `.Page.Sidebar.Icon` | `_docs/baseof.html` |
| Pagination | Prev/next page links | `.Pagination` | `_docs/baseof.html` |
| Scripts | `<script>` tags (deferred, inline, module) | `.Scripts`, `.InlineScripts`, `.ModuleScripts` | Both baseof templates |
| Search | Search trigger button and dialog modal | `.Site.Config.Search.Provider` | Header |
| Sidebar | Docs nav tree (3 levels deep) | `.Sidebar`, `.SidebarCollapsedByDefault` | `_docs/baseof.html` |
| SiteTitle | Site name/logo link to `/` | `.Site.Title` | Header |
| Social | Social icon links row | `.Site.Config.Social` | Header, Footer |
| TableOfContents | Desktop sidebar ToC | `.Page.Headings` | `_docs/baseof.html` |
| TagSidebar | Popular tags widget (top 20) | `topTerms "tags" 20` | *Not called by default* |
| ThemeToggle | Light/system/dark toggle | Static + i18n labels | Header |
| VersionBanner | Unmaintained/unreleased version notice | `.VersionBanner`, `.Versions` | `_docs/baseof.html` |
| VersionSwitcher | Version dropdown | `.Versions`, `.VersionLabel` | Header |

Components marked *Not called by default* are registered and overridable but not wired into the default layout. Add them to a custom layout or component override to use them.

## Composition

The docs layout (`_docs/baseof.html`) calls components in this order:

```text
Head
Header
  ├── SiteTitle
  ├── GlobalNav
  ├── Search
  ├── VersionSwitcher
  ├── LanguageSwitcher
  ├── Social              (if header.social enabled)
  ├── CenterToggle
  └── ThemeToggle
MobileTableOfContents     (if page has headings)
Sidebar
  └── DocsTabSwitcher     (if collection is tabbed)
ThemeToggle               (in sidebar footer)
VersionSwitcher           (in sidebar footer)
LanguageSwitcher          (in sidebar footer)
TableOfContents           (if page has headings)
Breadcrumbs               (if breadcrumbs exist)
FallbackNotice
VersionBanner
PageBanner
PageTitle
PageTags
  [page content]
Pagination                (if pagination exists)
Footer
  └── Social
Scripts
```

The default layout (`_default/baseof.html`) is leaner: Head, Header, FallbackNotice, PageBanner, content block, Footer, Scripts. It does not call Breadcrumbs, PageTitle, PageTags, Sidebar, or any ToC component.

## Component details

### Head

Renders `<head>` content: charset, viewport, and generator meta tags, sitemap link (if [`search`](/reference/configuration#search) enables it), favicon, an inline `window.__SARDE__` config object, the `<title>` tag, meta description, SEO tags via `partial "seo.html"`, theme styles via [`themeStyles`](/reference/template-functions#templates), per-page stylesheets from `.Styles`, and per-page head tags from [`frontmatter head`](/reference/frontmatter#head).

### Header

Composes eight sub-components: SiteTitle, GlobalNav, Search, VersionSwitcher, LanguageSwitcher, Social, CenterToggle, ThemeToggle. Social is gated by [`header.social`](/reference/configuration#header) (default `true`). Override `Header.html` to reorganize or remove any of these elements.

### Sidebar

Renders the docs navigation tree up to 3 levels deep using collapsible `<details>`/`<summary>` elements. Reads `.Sidebar.Root.Children` and each node's URL, Label, Icon, IsActive, IsOpen, and HasActive fields. Renders [`sidebar.badge`](/reference/frontmatter#sidebar-badge) when present on a nav node's page. Calls DocsTabSwitcher at the top when `.IsTabbed` is true. Sidebar section open/closed state persists via `sessionStorage`.

### Footer

Renders navigation links from [`footer.links`](/reference/configuration#footer), a Social component (if [`social`](/reference/configuration#social) is configured), a copyright line with the site title and current year, optional custom text from [`footer.text`](/reference/configuration#footer), and a "Made with Sarde" credit (controlled by [`footer.credits`](/reference/configuration#footer), default `true`).

### Search

Hidden entirely when [`search.provider`](/reference/configuration#search) is `"disabled"`. Renders a trigger button with a keyboard hint (Ctrl/Cmd+K) and a `<dialog>` modal containing the search input, results list, filters, and a full-search mode toggle. Scopes results by `.Version` and `.Lang` via `data-*` attributes read by the client-side Orama search script.

### PageBanner

Renders when `.PageBanner` is set via [`frontmatter banner`](/reference/frontmatter#banner). Accepts `Content` (text), `Variant` (default `"note"`), and `Icon` (optional). When no icon is specified, one is auto-selected by variant: `tip` uses `lightbulb`, `caution` uses `alert-triangle`, `danger` uses `alert-octagon`, and all others use `info`.

### PageTags

Renders tag chips when [`showPageTags`](/reference/template-functions#content) returns true (page override, then [`taxonomies.tags.show_tags`](/reference/configuration#taxonomies) config, then `true`). Each tag links to its term page via [`termURL`](/reference/template-functions#content) and displays the label, icon, and color from its [`TaxonomyTerm`](/reference/frontmatter#taxonomy-fields) definition.

### EditLink

Resolves the "Edit this page" URL in order: a custom string from `edit_url` in [page frontmatter](/reference/frontmatter#meta-fields), or [`site.edit_url`](/reference/configuration#site) joined with the page's relative path. Set `edit_url: false` in frontmatter to disable the link for a specific page.

Called from the blog, default, and docs layouts. It renders nothing unless `site.edit_url` is set, so sites that do not configure it see no change.

`site.edit_url` must point at the **content directory**, not the repository root, because the page path appended to it is relative to `content/`. For a site whose project lives in `docs/`, that means `https://github.com/user/repo/edit/main/docs/content`.

### TableOfContents and MobileTableOfContents

Both consume `.Page.Headings` and render anchor links for each heading. Both include a synthetic "Overview" link to `#_top`. The mobile variant wraps the list in a collapsible `<details>` element and adds a scroll-progress SVG ring. The desktop variant renders in the sidebar ToC panel. Both are conditionally called only when `.Page.Headings` is non-empty.

### VersionSwitcher and VersionBanner

VersionSwitcher renders a dropdown of all versions from `.Versions` with the current version label, hidden when `.Versions` is empty (non-versioned collections). VersionBanner shows an "unmaintained" or "unreleased" notice when `.VersionBanner` is set, with a link to the latest version. See [`collections versioning`](/reference/configuration#taxonomies) for configuration.

### LanguageSwitcher

Renders a language dropdown from `.AllTranslations`. Hidden on single-language sites (empty `.AllTranslations` list). Each entry shows the language name, direction indicator for RTL languages, and a fallback badge when the translation is a fallback page.

### LastUpdated

Renders `<p class="sarde-last-updated">` containing a `<time>` element with the absolute date from `.Page.Updated`, gated by `show_updated` in [frontmatter](/reference/frontmatter#meta-fields) (default `true`).

Called from the docs, labs, blog single, and default single layouts, below the article. It renders nothing when the page has no resolvable timestamp.

The date is rendered at build time, so it is present without JavaScript and the page does not shift on load.

The display format comes from [`theme.date_format`](/reference/configuration#theme), which accepts `short`, `long`, `iso`, or any Go layout string. The `datetime` attribute is always ISO 8601 regardless, so the markup stays machine-readable. The timestamp itself is resolved by [`build.last_updated`](/reference/configuration#last-updated-strategy).

### ContentPanel and TagSidebar

These two components are registered and overridable but not called by the default docs or default layouts. They serve as insertion points for custom layouts:

- **ContentPanel** wraps `.Page.Content` in a `<div>`. Use it to add a content-area wrapper (ads, feedback widget) without overriding the entire layout.
- **TagSidebar** renders the top 20 tags via `topTerms "tags" 20` as a sidebar widget. Used by blog list templates (`_blog/list.html`, `_blog/list-grid.html`) but not by the docs layout.
