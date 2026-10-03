---
title: Tabbed Navigation
description: "Split a large docs collection into top-level tabs with an auto-detected or configured tab switcher"
sidebar:
  order: 6
---

Tabbed navigation splits a large docs collection into top-level areas. A tab switcher appears at the top of the sidebar, and the sidebar shows only the pages of the active tab. Sarde turns tabs on automatically when a collection's directory structure fits, and config can force tabs on or off.

## How tabs work

Each top-level section of the collection becomes a tab. The switcher is a card at the top of the sidebar: a tile with the tab's icon (or its initials when it has no icon), the collection title as a small label, and the active tab's title. Opening the card lists every tab with its tile, title, and description.

→ Selecting a tab opens that section's index page, and the sidebar shows only that section's navigation tree.

The collection root (`/docs/`) redirects to the first tab, so the root `_index.md` of a tabbed collection is never shown as a page.

<!-- SCREENSHOT: docs-tab-switcher - the tab switcher open above the sidebar, listing tabs with icons and descriptions -->

## The sidebar inside a tab

The sidebar does not repeat the tab as a group, since the switcher already names it. The tab's pages and subsections start at the top level, led by an **Overview** entry that links to the tab's `_index.md`:

```text
Overview              <- guide/_index.md
Installation
Configuration
Advanced          ⌄
   Caching
```

- **Rename the entry** with `sidebar.label` in the tab's `_index.md`, for example `label: Introduction`. The default label is the `nav.overview` [translation string](/guides/internationalization/#translation-strings), so it follows the page language.
- **Remove the entry** with `sidebar.hidden: true` in the tab's `_index.md`. A tab without an `_index.md` has no Overview entry.
- **Icon:** the entry uses the tab's `sidebar.icon`. The tab's `sidebar.badge` is not shown on it.
- **Order:** Overview always comes first. `sidebar.order` on the tab's `_index.md` sets the tab's position in the switcher, not the entry's position in the sidebar.
- A tab with its own [`nav.yaml`](#per-tab-nav-yaml) keeps exactly the tree that file describes.

Previous and Next links follow the sidebar order within the active tab. Readers never cross from the last page of one tab to the first page of the next.

## Auto-detection

Sarde enables tabs automatically when all of these hold:

1. The collection uses a sidebar layout (`docs`, `wide`, or `labs`).
2. The collection root contains two or more sections (subdirectories with content).
3. Every top-level section has an `_index.md` file.
4. No loose pages sit at the collection root. Only `_index.md` is allowed there.

This structure triggers tabs:

```text
content/docs/
  _index.md
  guide/
    _index.md
    installation.md
  api/
    _index.md
    endpoints.md
  plugins/
    _index.md
    search.md
```

→ The sidebar shows a switcher with three tabs: Guide, API, and Plugins.

Adding a loose page (for example `content/docs/changelog.md`) or removing one of the `_index.md` files turns auto-detection off, and the sidebar falls back to a single tree of collapsible groups.

Detection ignores version directories. With [versioning](/guides/versioning/) enabled, a `v2/` directory does not become a tab.

## Enabling and disabling explicitly

Override auto-detection per collection in `sarde.yaml`:

```yaml title="sarde.yaml"
collections:
  docs:
    tabs: true    # force tabs even when auto-detection declines
    # tabs: false # never use tabs for this collection
```

A collection can also opt out in the frontmatter of its root `_index.md`:

```yaml title="content/docs/_index.md"
---
title: Documentation
tabs: false
---
```

The frontmatter form only opts out. `tabs: true` in frontmatter does not force tabs, and `tabs: true` in `sarde.yaml` takes precedence over `tabs: false` in frontmatter.

Forcing tabs with `tabs: true` skips the auto-detection checks:

- A collection with a single top-level section gets one tab.
- Loose pages at the collection root stay outside every tab and do not appear in any tab's sidebar.
- A top-level section without an `_index.md` still becomes a tab. Its label is the directory name, and it has no icon or description.

## Tab labels, icons, and order

Each tab takes its label, icon, description, and position from the section's `_index.md`:

```yaml title="content/docs/api/_index.md"
---
title: API
description: Endpoint and schema reference
icon: braces
sidebar:
  order: 2
---
```

| Field | Used for |
|-------|----------|
| `title` | Tab label in the switcher. Falls back to the directory name when no `_index.md` exists. |
| `description` | Secondary line under the label in the switcher menu. When empty, the menu shows a short excerpt of the page body. |
| `icon` | Icon in the tab's tile, in place of the initials of its title. This is the page-level `icon` field, not `sidebar.icon`. |
| `sidebar.order` | Tab position. Lower values come first, and ties sort alphabetically by title. |

A tab with no `sidebar.order` counts as `0`, so it sorts before a tab with `order: 1`. Set `sidebar.order` on every tab to fix the sequence, because the first tab is also where the collection root redirects.

To change a tab's label, icon, description, or position without editing its `_index.md`, use the `tabs` block of [`sidebar.yaml`](/guides/navigation-and-sidebar/#tab-overrides).

## Per-tab `nav.yaml`

A tabbed collection can replace the auto-generated tree of a single tab with a manual `nav.yaml` placed inside that tab's directory:

```text
content/docs/
  guide/
    _index.md
    nav.yaml        # manual navigation for the Guide tab only
  api/
    _index.md       # API tab keeps its auto-generated tree
```

See [Navigation and Sidebar](/guides/navigation-and-sidebar/#manual-tab-sidebar-with-nav-yaml) for the item format. Sarde ignores a `nav.yaml` at the collection root. If a tab's `nav.yaml` fails to parse, Sarde falls back to the auto-generated tree for that tab without reporting an error.

## Tabs with versioning and i18n

Tabs compose with both [versioning](/guides/versioning/) and [internationalization](/guides/internationalization/). Each language and version pair gets its own tab set and navigation trees, so a reader browsing French v2 docs sees French v2 tabs. The switcher links stay within the current language and version.

See [Navigation and Sidebar](/guides/navigation-and-sidebar/) for sidebar behavior inside a tab, and [Configuration](/reference/configuration/content/#collections) for the `tabs` collection setting.
