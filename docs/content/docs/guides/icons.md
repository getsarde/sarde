---
title: Icons
description: "Insert inline SVG icons from Iconify sets or local files using the :icon[name] extension"
sidebar:
  order: 14
---

Sarde renders inline SVG icons from Iconify icon sets and local SVG files. Insert icons in Markdown with the `:icon[name]` extension. Lucide is the default bundled set.

## Basic usage

Insert an icon by name:

```markdown
Click the :icon[settings] icon to open preferences.
```

Result: An inline SVG of the Lucide "settings" icon appears in the text.

Add attributes for color and size:

```markdown
:icon[alert-triangle color="orange" size=20]
```

Result: An orange warning triangle at 20px.

## Icon resolution

When Sarde encounters `:icon[name]`, it resolves the icon in this order:

1. **Local SVGs** (`icons/` directory): if `name.svg` exists, use it
2. **Default icon set** (Lucide): look up `name` in the bundled set
3. **Fallback**: render a `circle-help` placeholder icon

### Prefixed names

Reference icons from a specific set with the `prefix:name` syntax:

```markdown
:icon[tabler:brand-github]
:icon[simple-icons:typescript]
```

The `brands:` prefix is a shortcut for `simple-icons:`:

```markdown
:icon[brands:github]
```

## Adding icon sets

Sarde supports any icon set published in Iconify JSON format (4000+ sets available). Download sets with the CLI:

```sh
sarde icons add tabler simple-icons
```

Result: Downloads `tabler.json` and `simple-icons.json` into the `icon-sets/` directory or the path configured in `icons.sets_dir`.

Browse available sets:

```sh
sarde icons list
```

Result: Displays a paginated table of all Iconify sets with prefix, name, icon count, and local download status.

Filter the list:

```sh
sarde icons list --search "brand"
```

After downloading, reference icons with the set prefix:

```markdown
:icon[tabler:home]
:icon[simple-icons:react]
```

## Custom local icons

Place SVG files in the `icons/` directory (configurable via `icons.local_dir`):

```text
icons/
  logo.svg
  custom-arrow.svg
```

Reference them by filename (without `.svg`):

```markdown
:icon[logo]
:icon[custom-arrow]
```

Local icons take priority over set icons. A file at `icons/home.svg` overrides the Lucide `home` icon.

## Configuration

Configure the icon system in `sarde.yaml`:

```yaml
icons:
  default_prefix: "lucide"
  sets_dir: "icon-sets"
  local_dir: "icons"
  render: "inline"
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `default_prefix` | string | `"lucide"` | Icon set used for bare (prefixless) names |
| `sets` | array | `[]` | Extra Iconify JSON sets to load: `[{prefix: "mdi", file: "icon-sets/mdi.json"}]` |
| `sets_dir` | string | `""` | Directory of Iconify `*.json` files, auto-discovered by filename |
| `local_dir` | string | `"icons"` | Directory of local `*.svg` files |
| `attribution` | string | `""` | Attribution line for sets whose license requires it |
| `render` | string | `"inline"` | Output mode: `"inline"` (full `<svg>` per use) or `"sprite"` (one `<symbol>` per unique icon, referenced via `<use>`) |

### Render modes

- **`inline`** (default): each `:icon[name]` emits a complete `<svg>` element. Produces larger HTML but each icon is self-contained.
- **`sprite`**: emits one hidden `<symbol>` per unique icon per page and references it with `<svg><use href="#..."></svg>`. Smaller HTML when the same icon appears many times on a page.

## Icon attributes

The `:icon[]` extension accepts these attributes inside the brackets:

| Attribute | Description |
|-----------|-------------|
| `color` | Icon color (any CSS color value) |
| `size` | Icon size in pixels |
| `width` | Icon width (overrides size for non-square) |
| `height` | Icon height (overrides size for non-square) |
| `rotate` | Rotation in degrees |
| `flip` | Flip direction: `horizontal`, `vertical`, `both` |
| `title` | Accessible title text |
| `aria-label` | ARIA label for screen readers |

Icons without `title` or `aria-label` are rendered with `aria-hidden="true"` for accessibility.

See [CLI Commands](/reference/cli-commands#icons) for `sarde icons` usage, and [Configuration](/reference/configuration#icons) for all icon settings.
