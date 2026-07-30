---
title: Navigation and Sidebar
description: "Control sidebar ordering, grouping, icons, and collapsed state through frontmatter and directory structure"
sidebar:
  order: 5
---

Sarde auto-generates sidebar navigation from the directory structure in docs-layout collections. Pages are sorted by `sidebar.order` (from frontmatter or numeric filename prefix), then by title. No configuration is required for the default behavior.

## Auto-generated sidebar

For docs-layout collections (`docs/`, `courses/`, `tutorials/`, etc.), Sarde walks the directory tree and builds a collapsible navigation sidebar. Each subdirectory with an `_index.md` becomes a collapsible group. Pages within a group are sorted by their `sidebar.order` field.

```text
content/docs/
  _index.md
  start-here/                    # Sidebar group: "Start Here"
    _index.md                    # sidebar.order: 1
    getting-started.md
    deploying.md
  guides/                        # Sidebar group: "Guides"
    _index.md                    # sidebar.order: 2
    writing-content.md
```

Result: The sidebar shows two collapsible groups with pages nested inside each.

Each group label is a link to that section's `_index.md`, so clicking "Guides" opens the section index rather than only expanding the group. Set `render: false` in the section's `_index.md` to keep the label as plain text, which suits a section index that exists only to name the group.

<!-- SCREENSHOT: sidebar-auto-generated - auto-generated sidebar with two collapsible groups -->

## Controlling sidebar order

Set `sidebar.order` in frontmatter to control the position of a page or section within its parent group:

```yaml
---
title: Getting Started
sidebar:
  order: 1
---
```

Alternatively, prefix filenames with numbers: `01-getting-started.md` sets `sidebar.order` to 1 without frontmatter. See [Writing Content](/guides/writing-content#numeric-filename-prefixes) for details.

## Sidebar labels

Override the sidebar display label without changing the page title:

```yaml
---
title: Internationalization and Localization
sidebar:
  label: "i18n"
---
```

Result: The sidebar shows "i18n" while the page heading remains "Internationalization and Localization".

## Hiding pages

Hide a page from the sidebar while keeping it accessible via direct URL:

```yaml
---
title: Internal Notes
sidebar:
  hidden: true
---
```

## Sidebar badges

Add a badge chip next to a sidebar item:

```yaml
---
title: New Feature
sidebar:
  badge: "New"
---
```

Badges also support variant styling:

```yaml
---
sidebar:
  badge:
    text: "Beta"
    variant: "caution"
---
```

## Collapsible sections

Sidebar groups are collapsible by default. Configure this per collection in `sarde.yaml`:

```yaml
collections:
  docs:
    sidebar:
      collapsible: true
      collapsed_by_default: false
      max_depth: 4
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `collapsible` | bool | `true` | Allow groups to collapse/expand |
| `collapsed_by_default` | bool | `false` | Start all groups collapsed |
| `max_depth` | int | `4` | Maximum nesting depth in the sidebar tree |
| `search` | bool | `true` | Show the sidebar filter input |

Open/closed state for each group persists across page navigations via `sessionStorage`.

## Overrides with `sidebar.yaml`

A `sidebar.yaml` at the project root adjusts individual sidebar entries without touching their frontmatter. Use it to relabel or reorder pages you do not own, or to keep presentation choices out of the content files.

```yaml
docs:
  collapse_level: 2
  overrides:
    guide:
      label: "Getting Started"
      order: 1
      icon: rocket
      collapsed: true
    guide/intro:
      label: "Introduction"
      badge: New
    guide/legacy:
      hidden: true
```

The top-level key is the collection name. `sidebar.yaml` sits above `sarde.yaml` in the cascade, so anything set here wins.

### Override keys

Each key under `overrides` is a page or section path relative to the collection root, with no leading or trailing slash. A page at `/docs/guide/intro/` in the `docs` collection is keyed `guide/intro`.

Keys are canonicalized before matching: backslashes become forward slashes, and leading and trailing slashes are trimmed. Two keys that canonicalize to the same path are a build error rather than a silent last-wins, so `/guide/intro/` and `guide/intro` in the same file stop the build and name both spellings.

A key matching nothing produces a warning naming the key and its collection, and the build continues.

| Key | Type | Description |
|-----|------|-------------|
| `label` | string | Replaces the sidebar text |
| `description` | string | Description text where the theme shows one |
| `order` | int | Sort position, same scale as `sidebar.order` |
| `collapsed` | bool | Start this group open or closed |
| `icon` | string | Icon on the entry |
| `badge` | string or object | Badge chip, scalar or `{text, variant}` |
| `hidden` | bool | Show or hide the entry. See below |
| `attrs` | map | Extra HTML attributes on the link |

`collapsed: true` is ignored while the reader is on a page inside that group, so the current page is never hidden behind a closed section.

### Un-hiding a page

`hidden` is three-state. Leaving it out changes nothing, `hidden: true` removes the entry, and `hidden: false` restores a page that set `sidebar.hidden: true` in its own frontmatter.

That last case is the reason the field is not a plain boolean: it lets a site reveal a page it does not want to edit. Sections have no frontmatter `hidden` field, so `hidden: false` on a section does nothing.

### Tab overrides

For [tabbed collections](/guides/tabbed-navigation), `tabs` adjusts the tab bar. Each key is the tab's directory name.

```yaml
docs:
  tabs:
    api:
      label: "API"
      icon: plug
      order: 1
```

| Key | Type | Description |
|-----|------|-------------|
| `label` | string | Replaces the tab title from the tab's `_index.md` |
| `description` | string | Description shown in the mobile tab menu |
| `icon` | string | Icon on the tab |
| `order` | int | Tab position |

### Validation

The file is strictly parsed. An unknown field is a build error naming the line and the field, so a typo like `lable:` fails loudly instead of being silently dropped. An empty or comments-only file is valid and contributes nothing.

:::caution
A structural `items:` list is accepted by the parser but not implemented. Supplying one emits `structural sidebar items are not implemented yet; ignoring` and the entry is dropped. Use `nav.yaml` below to hand-author a tree.
:::

## Manual tab sidebar with `nav.yaml`

[Tabbed collections](/guides/tabbed-navigation) can use `nav.yaml` inside a tab
directory to replace the auto-generated navigation for that tab. Use this when
the tab needs links that do not match the file tree.

The file only applies per tab. A `nav.yaml` at the collection root is ignored,
and non-tabbed collections always use the auto-generated sidebar.

`content/docs/guides/nav.yaml`

```yaml
- label: "Getting Started"
  page: getting-started
- label: "Guides"
  items:
    - label: "Writing Content"
      page: guides/writing-content
    - label: "Code Blocks"
      page: guides/code-blocks
- label: "External Docs"
  url: "https://example.com"
  external: true
```

Each item supports:

| Key | Type | Description |
|-----|------|-------------|
| `label` | string | Display text (falls back to page title) |
| `page` | string | Page slug or relative path within the collection |
| `url` | string | External URL (use with `external: true`) |
| `external` | bool | Open in new tab with `noopener noreferrer` |
| `badge` | object | Badge chip on this item |
| `collapsed` | bool | Override group open/closed default |
| `items` | array | Nested child items |

Items without `page` or `url` act as group labels (headings with no link).

## Breadcrumbs

Docs-layout pages render breadcrumbs automatically: Collection Root > Section > Page. Transparent sections are skipped in the breadcrumb trail. Sections without an `_index.md` appear as plain text (no link).

## Prev/next links

Docs-layout collections render prev/next navigation links at the bottom of each page. The order follows a depth-first traversal of the sidebar tree, so readers navigate through sections sequentially.

Override prev/next for a specific page in frontmatter:

```yaml
---
prev: "installation"
next:
  slug: "advanced-config"
  label: "Advanced Configuration"
---
```

Disable prev/next for a page:

```yaml
---
prev: false
next: false
---
```

## Global navigation

The site header displays navigation links configured in `sarde.yaml`:

```yaml
header:
  links:
    - label: "Docs"
      url: "/docs/"
    - label: "Blog"
      url: "/blog/"
    - label: "GitHub"
      url: "https://github.com/getsarde/sarde"
      external: true
```

Header links appear as a horizontal navigation bar. External links open in a new tab.

## Table of contents

Docs-layout pages display a table of contents panel on the right side of the content area. The TOC lists headings extracted from the page content, with scroll-synchronized highlighting of the current section.

Disable the TOC site-wide or per collection:

```yaml title="sarde.yaml"
toc:
  enabled: false
```

Or per page in frontmatter:

```yaml
---
toc: false
---
```

### Heading level range

Two settings control which headings appear in the TOC:

1. **`markdown.toc.min_heading_level` / `max_heading_level`** controls which headings are *extracted* during the build. Headings outside this range get no `id` attribute, no anchor link, and cannot be linked to with fragment URLs. Default: 2 through 4.

2. **`toc.min_level` / `toc.max_level`** controls which extracted headings are *displayed* in the TOC sidebar. This can only narrow the range, not widen it beyond what was extracted. Default: 2 through 4.

To include all heading levels in the TOC:

```yaml title="sarde.yaml"
markdown:
  toc:
    max_heading_level: 6

toc:
  max_level: 6
```

To extract h2 through h6 for IDs and link validation, but only display h2 and h3 in the sidebar:

```yaml title="sarde.yaml"
markdown:
  toc:
    max_heading_level: 6

toc:
  max_level: 3
```

Per-page frontmatter can override the display range for individual pages. See [Frontmatter](/reference/frontmatter#table-of-contents-fields) for per-page `toc:` options.

## Mobile sidebar

On screens narrower than 1024px, the sidebar collapses into a drawer accessible via a hamburger menu button. The drawer slides in from the left and contains the same navigation tree. It closes on link click or by tapping outside the drawer.

See [Configuration](/reference/configuration/) for all sidebar settings, and [Frontmatter](/reference/frontmatter#sidebar-fields) for per-page sidebar fields.
