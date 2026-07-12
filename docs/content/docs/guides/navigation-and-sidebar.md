---
title: Navigation and Sidebar
description: "Control sidebar ordering, grouping, icons, and collapsed state through frontmatter and directory structure"
sidebar:
  order: 4
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

## Mobile sidebar

On screens narrower than 1024px, the sidebar collapses into a drawer accessible via a hamburger menu button. The drawer slides in from the left and contains the same navigation tree. It closes on link click or by tapping outside the drawer.

See [Configuration](/reference/configuration#sidebar) for all sidebar settings, and [Frontmatter](/reference/frontmatter#sidebar-fields) for per-page sidebar fields.
