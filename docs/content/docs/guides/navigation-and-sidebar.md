---
title: Navigation and Sidebar
description: "Control sidebar ordering, grouping, icons, and collapsed state through frontmatter, sidebar.yaml, and directory structure"
sidebar:
  order: 5
---

Sarde builds the sidebar of a docs-layout collection from its directory tree. Pages sort by `sidebar.order`, then alphabetically by label. The default behavior needs no configuration; this page covers the controls for changing it.

## Auto-generated sidebar

For docs-layout collections (`docs/`, `courses/`, `tutorials/`, and the other names in [Content and Collections](/guides/content-and-collections/)), Sarde walks the directory tree and builds a collapsible sidebar. Every subdirectory becomes a group, and the pages inside it are its children.

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

→ The sidebar shows two collapsible groups with their pages nested inside.

A group takes its label and position from the section's `_index.md`. A subdirectory without an `_index.md` still becomes a group, labeled from the directory name and listed at order `0`.

A group label is a link to the section's `_index.md`, so selecting "Guides" opens the section page as well as expanding the group. Set `render: false` in the `_index.md` to keep the label as plain text. Sarde then does not generate a page for that section, which suits an index that exists only to name the group.

<!-- SCREENSHOT: sidebar-auto-generated - auto-generated sidebar with two collapsible groups -->

## Controlling sidebar order

Set `sidebar.order` in frontmatter to position a page or section within its parent group. Lower values come first, and entries with the same value sort alphabetically.

```yaml title="getting-started.md"
---
title: Getting Started
sidebar:
  order: 1
---
```

A numeric filename prefix does the same without frontmatter: `01-getting-started.md` sets `sidebar.order` to 1. See [Numeric filename prefixes](/guides/writing-content/#numeric-filename-prefixes).

## Sidebar labels

Override the sidebar text without changing the page title:

```yaml
---
title: Internationalization and Localization
sidebar:
  label: "i18n"
---
```

→ The sidebar shows "i18n" while the page heading stays "Internationalization and Localization".

## Hiding pages

Hide a page from the sidebar while keeping it reachable by URL:

```yaml
---
title: Internal Notes
sidebar:
  hidden: true
---
```

## Sidebar badges

Add a badge chip next to a sidebar entry:

```yaml
---
title: New Feature
sidebar:
  badge: "New"
---
```

Use the object form to pick a variant (`default`, `note`, `tip`, `success`, `caution`, or `danger`):

```yaml
---
sidebar:
  badge:
    text: "Beta"
    variant: "caution"
---
```

A badge set in a section's `_index.md` shows on that section's group. See [Frontmatter](/reference/frontmatter/#sidebar-badge) for the legacy color aliases.

## Collapsible sections

Sidebar groups start open. Configure the sidebar per collection in `sarde.yaml`:

```yaml title="sarde.yaml"
collections:
  docs:
    sidebar:
      collapsed_by_default: true
      collapse_level: 1
      max_depth: 4
```

The `sidebar` keys:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `collapsed_by_default` | bool | `false` | Start all groups collapsed. The group holding the current page is always open. |
| `collapse_level` | int | unset | Open groups down to this depth and collapse deeper ones. `1` opens the top-level groups only. Overrides `collapsed_by_default` for the groups it covers. |
| `max_depth` | int | `4` | Maximum nesting depth in the sidebar tree. Accepts 1 to 10. Deeper sections do not appear. |
| `collapsible` | bool | `true` | Accepted, but the default theme always renders collapsible groups. |
| `search` | bool | `true` | Accepted, but the default theme renders no sidebar filter. |

Each group remembers whether the reader opened or closed it. The state lasts for the browser session (`sessionStorage`) and resets when the sidebar structure changes.

## Overrides with `sidebar.yaml`

A `sidebar.yaml` next to `sarde.yaml` adjusts individual sidebar entries without touching their frontmatter. Use it to relabel or reorder pages you do not own, or to keep presentation choices out of the content files.

```yaml title="sidebar.yaml"
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

The top-level key is the collection name. `sidebar.yaml` wins over both `sarde.yaml` and frontmatter, and a `collapse_level` here wins over the one in `sarde.yaml`.

### Override keys

Each key under `overrides` is a page or section path relative to the collection root. A page at `/docs/guide/intro/` in the `docs` collection is keyed `guide/intro`.

Sarde canonicalizes keys before matching: backslashes become forward slashes, and leading and trailing slashes are trimmed. Two keys that canonicalize to the same path stop the build with an error naming both spellings, so `/guide/intro/` and `guide/intro` cannot share a file.

A key that matches no page or section produces a warning naming the key and its collection, and the build continues.

Each override accepts these fields:

| Key | Type | Description |
|-----|------|-------------|
| `label` | string | Replaces the sidebar text. |
| `description` | string | Shown as a tooltip on the entry in the default theme. |
| `order` | int | Sort position, on the same scale as `sidebar.order`. |
| `collapsed` | bool | `false` opens the group by default. `true` closes it, but only when the collection already starts groups collapsed (`collapsed_by_default` or `collapse_level`). |
| `icon` | string | Icon on the entry. |
| `badge` | string or object | Badge chip, as a string or `{text, variant}`. |
| `hidden` | bool | Show or hide the entry. See [Un-hiding a page](#un-hiding-a-page). |
| `attrs` | map | Extra attributes on the entry for themes that render them. The default theme ignores it. |

A closed group still opens while the reader is on a page inside it, so the current page is never hidden behind a closed section.

### Un-hiding a page

`hidden` has three states. Leaving it out changes nothing, `hidden: true` removes the entry, and `hidden: false` restores a page that set `sidebar.hidden: true` in its own frontmatter. This lets a site reveal a page without editing it. Sections have no `hidden` frontmatter field, so `hidden: false` on a section does nothing.

### Tab overrides

For [tabbed collections](/guides/tabbed-navigation/), `tabs` adjusts the tab switcher. Each key is the tab's directory name.

```yaml title="sidebar.yaml"
docs:
  tabs:
    api:
      label: "API"
      icon: plug
      order: 1
```

Each tab accepts these fields:

| Key | Type | Description |
|-----|------|-------------|
| `label` | string | Replaces the tab title taken from the tab's `_index.md`. |
| `description` | string | Description under the tab title in the switcher menu. |
| `icon` | string | Icon on the tab. |
| `order` | int | Tab position. |

### Validation

Sarde parses the file strictly. An unknown field stops the build with an error naming the line and the field, so a typo such as `lable:` stops the build instead of being ignored. An empty or comments-only file is valid and contributes nothing.

:::caution
`sidebar.yaml` does not support a structural `items:` list. Supplying one prints `structural sidebar items are not implemented yet; ignoring` and drops the entry. Use [`nav.yaml`](#manual-tab-sidebar-with-nav-yaml) to hand-author a tab's tree.
:::

## Manual tab sidebar with `nav.yaml`

A [tabbed collection](/guides/tabbed-navigation/) can replace the auto-generated sidebar of one tab with a `nav.yaml` inside that tab's directory. Use it when the tab needs a different structure than its file tree.

The file applies per tab only. Sarde ignores a `nav.yaml` at the collection root, and collections without tabs always use the auto-generated sidebar.

```yaml title="content/docs/guides/nav.yaml"
- label: "Getting Started"
  page: getting-started
- label: "Guides"
  items:
    - label: "Writing Content"
      page: guides/writing-content
    - label: "Code Blocks"
      page: guides/code-blocks
```

Each item supports:

| Key | Type | Description |
|-----|------|-------------|
| `label` | string | Display text. Falls back to the page's sidebar label, then its title. |
| `page` | string | Page slug, or path relative to the collection root. |
| `badge` | string or object | Badge chip on a page item. Overrides the page's own badge. |
| `collapsed` | bool | `false` opens the group by default. |
| `attrs` | map | Extra attributes on the entry for themes that render them. The default theme ignores it. |
| `items` | array | Nested child items. |

An item without `page` acts as a group label, a heading with no link. A `page` that matches no page in the tab is skipped without a warning. The tab's `_index.md` gets no Overview entry, because the file defines the whole tree.

## Breadcrumbs

Docs-layout pages show breadcrumbs from the collection title through each section to the page. Tabbed collections add the tab after the collection title. Transparent sections are skipped, and sections without an `_index.md` appear as plain text.

## Previous and next links

Docs-layout collections show Previous and Next links at the bottom of each page. The order follows the sidebar tree depth-first, so readers move through sections in sequence. In a tabbed collection the links stay inside the active tab.

Override a link for one page in frontmatter. A string names the target page by slug, and the object form sets an explicit URL and label:

```yaml
---
prev: "installation"
next:
  link: "/docs/guides/advanced-config/"
  label: "Advanced Configuration"
---
```

A string slug must match a page in the same sidebar tree. Setting only `label` in the object form renames the automatic link without changing its target.

Remove a link from a page:

```yaml
---
prev: false
next: false
---
```

To turn the links off for a whole collection, set `collections.<name>.prev_next.enabled: false` in `sarde.yaml`.

## Header navigation

The site header lists each collection in alphabetical order, linking to its root. Links from `header.links` in `sarde.yaml` follow them:

```yaml title="sarde.yaml"
header:
  links:
    - label: "GitHub"
      url: "https://github.com/getsarde/sarde"
      external: true
```

External links open in a new tab.

## Table of contents

Docs-layout pages show a table of contents on the right of the content area. It lists the page's headings and highlights the current section as the reader scrolls.

Turn it off for the whole site:

```yaml title="sarde.yaml"
toc:
  enabled: false
```

Turn it off for one collection:

```yaml title="sarde.yaml"
collections:
  docs:
    toc:
      enabled: false
```

Or for one page, in frontmatter:

```yaml
---
toc: false
---
```

### Heading level range

Two settings control which headings appear in the table of contents:

1. **`markdown.toc.min_heading_level` and `max_heading_level`** control which headings Sarde extracts during the build. Headings outside the range get no `id` attribute or anchor link, and fragment URLs cannot target them. The default is 2 through 4.
2. **`toc.min_level` and `toc.max_level`** control which extracted headings the table of contents displays. This can narrow the range but not widen it past what was extracted. The default is 2 through 4.

To include every heading level:

```yaml title="sarde.yaml"
markdown:
  toc:
    max_heading_level: 6

toc:
  max_level: 6
```

To extract h2 through h6 for IDs and link validation but display only h2 and h3:

```yaml title="sarde.yaml"
markdown:
  toc:
    max_heading_level: 6

toc:
  max_level: 3
```

Frontmatter can override the display range for a single page. See [Frontmatter](/reference/frontmatter/#table-of-contents-fields).

## Mobile sidebar

On screens narrower than 1024px, the sidebar becomes a drawer opened from a menu button in the header. The drawer slides in from the left and holds the same navigation tree.

See [Configuration](/reference/configuration/) for every sidebar setting, and [Frontmatter](/reference/frontmatter/#sidebar-fields) for the per-page sidebar fields.
