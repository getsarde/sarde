---
title: Tabbed Navigation
description: "Split a large docs collection into top-level tabs with an auto-detected or configured tab switcher"
sidebar:
  order: 5
---

Tabbed navigation splits a large docs collection into top-level areas. A tab switcher appears at the top of the sidebar, and the sidebar shows only the pages of the active tab. Sarde turns tabs on automatically when a collection's directory structure fits, and the behavior can be forced on or off in config.

## How tabs work

Each top-level section of the collection becomes a tab. The switcher is a dropdown at the top of the sidebar showing the active tab's icon and title. Opening it lists every tab with its icon, title, and description.

Result: Selecting a tab navigates to that section's index page, and the sidebar shows only that section's navigation tree.

<!-- SCREENSHOT: docs-tab-switcher - the tab switcher open above the sidebar, listing tabs with icons and descriptions -->

Prev/next links follow the sidebar order within the active tab. Readers never cross from the last page of one tab to the first page of the next.

## Auto-detection

Sarde enables tabs automatically when all of these hold:

1. The collection uses a sidebar layout (`docs` or `wide`).
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

Result: The sidebar shows a switcher with three tabs: Guide, API, and Plugins.

Adding a loose page (for example `content/docs/changelog.md`) or removing one of the `_index.md` files disables auto-detection, and the sidebar falls back to a single tree with collapsible groups.

Version directories are ignored by detection. With [versioning](/guides/versioning) enabled, a `v2/` directory does not become a tab.

## Enabling and disabling explicitly

Override auto-detection per collection in `sarde.yaml`:

```yaml
collections:
  docs:
    tabs: true    # force tabs even when auto-detection declines
    # tabs: false # never use tabs for this collection
```

A collection can also opt out in the frontmatter of its root `_index.md`:

```yaml
---
title: Documentation
tabs: false
---
```

The frontmatter form is opt-out only. Setting `tabs: true` in frontmatter does not force tabs; use the `sarde.yaml` setting for that.

## Tab labels, icons, and order

Each tab takes its label, icon, description, and position from the section's `_index.md`:

```yaml
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
| `description` | Secondary line under the label in the switcher dropdown. |
| `icon` | Icon shown next to the label. This is the page-level `icon` field, not `sidebar.icon`. |
| `sidebar.order` | Tab position. Lower values appear first; ties sort alphabetically by title. |

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

See [Navigation and Sidebar](/guides/navigation-and-sidebar) for the `nav.yaml` item format. A `nav.yaml` at the collection root is ignored; the file only applies per tab. If a tab's `nav.yaml` fails to parse, Sarde falls back to the auto-generated tree for that tab.

## Tabs with versioning and i18n

Tabs compose with both [versioning](/guides/versioning) and [internationalization](/guides/internationalization). Each language and version pair gets its own tab set and navigation trees, so a reader browsing French v2 docs sees French v2 tabs. The switcher links stay within the current language and version.

## Edge cases

- Auto-detection requires at least two top-level sections. Forcing `tabs: true` skips that check and builds tabs from whatever top-level sections exist, even a single one.
- Forcing `tabs: true` on a collection with loose root pages leaves those pages outside every tab. They are reachable by direct link but do not appear in any tab's sidebar.
- With `tabs: true`, a top-level section without an `_index.md` still becomes a tab. Its label is the directory name and it has no icon or description.
- The switcher only appears on tabbed collections. Other collections on the same site keep the regular sidebar.

See [Navigation and Sidebar](/guides/navigation-and-sidebar) for sidebar behavior inside a tab, and [Configuration](/reference/configuration#collections) for the `tabs` collection setting.
