---
title: Icons
description: "Insert inline SVG icons from Iconify sets or local files using the :icon[name] extension"
sidebar:
  order: 17
---

Sarde renders inline SVG icons from Iconify icon sets and from your own SVG files. Insert an icon in Markdown with `:icon[name]`. The Lucide set is bundled, so those icons work with no setup.

## Basic usage

Insert an icon by name:

```markdown
Click the :icon[settings] icon to open preferences.
```

→ The Lucide "settings" icon appears inline in the text.

Add attributes after the name to set color and size:

```markdown
:icon[alert-triangle color="orange" width=20 height=20]
```

→ An orange warning triangle, 20 pixels square, appears in the text. Without `width` and `height`, icons render at 16 pixels.

## Icon resolution

For a name without a prefix, Sarde looks for the icon in this order:

1. **Local SVGs:** `name.svg` in the `icons/` directory.
2. **Default icon set:** `name` in the set named by `icons.default_prefix` (Lucide by default).
3. **Alias:** a few shorthand names map to Lucide icons when the default prefix is `lucide`, for example `warning`, `note`, `tip`, `arrow`, and `close`.
4. **Fallback:** a `circle-help` placeholder icon.

A missing icon never fails the build and produces no warning, so a typo shows up as the placeholder on the page.

### Prefixed names

Write `prefix:name` to take an icon from a specific set, skipping local files:

```markdown
:icon[tabler:brand-github]
:icon[simple-icons:typescript]
```

`brands:` is a shortcut for `simple-icons:`:

```markdown
:icon[brands:github]
```

A prefixed icon renders only when its set is loaded. See [Add icon sets](#add-icon-sets).

## Add icon sets

Sarde reads icon sets in Iconify JSON format. Adding a set takes two steps: download it, then tell Sarde where the downloaded files are.

1. From the project directory, download one or more sets by prefix:

   ```sh
   sarde icons add tabler simple-icons
   ```

   → The command saves `tabler.json` and `simple-icons.json` into `icon-sets/` (or into `--dest` or `icons.sets_dir` when given), and prints one line per set. It fetches the sets from the npm registry.

2. Point `icons.sets_dir` at that directory:

   ```yaml title="sarde.yaml"
   icons:
     sets_dir: "icon-sets"
   ```

   Sarde loads every `*.json` file in `sets_dir` and registers each set under the `prefix` stored inside the file.

Reference the icons with the set prefix:

```markdown
:icon[tabler:home]
:icon[simple-icons:react]
```

To see which sets are available, run `sarde icons list`. It prints the prefix, name, icon count, category, license, and a `*` in the `DL` column for sets already downloaded, 30 rows at a time. Filter by prefix, name, or category:

```sh
sarde icons list --search "brand"
```

See [CLI Commands](/reference/cli-commands/#icons) for all flags. Some sets use licenses that require attribution, such as CC BY. When a page uses such a set and `icons.attribution` is empty, the build logs a warning. Set `attribution` to the credit line you show on the site.

## Custom local icons

Put SVG files in the `icons/` directory (change it with `icons.local_dir`):

```text
icons/
  logo.svg
  custom-arrow.svg
```

Reference them by filename without `.svg`:

```markdown
:icon[logo]
:icon[custom-arrow]
```

Subdirectories of `icons/` are not scanned. Local icons win over set icons for names without a prefix, so `icons/home.svg` replaces the Lucide `home` icon. `:icon[lucide:home]` still reaches the Lucide icon.

## Configuration

Icons need no configuration for Lucide and local files. Add keys to `sarde.yaml` to load other sets or change the output:

```yaml title="sarde.yaml"
icons:
  default_prefix: "lucide"
  sets_dir: "icon-sets"
  local_dir: "icons"
  render: "inline"
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `default_prefix` | string | `"lucide"` | Icon set used for names without a prefix. |
| `sets` | array | `[]` | Individual Iconify JSON files to load, as `file` entries such as `[{prefix: "mdi", file: "icon-sets/mdi.json"}]`. The set's prefix comes from the file's own `prefix` field. |
| `sets_dir` | string | `""` | Directory whose `*.json` files are all loaded as icon sets. |
| `local_dir` | string | `"icons"` | Directory of local `*.svg` files. |
| `attribution` | string | `""` | Credit line for sets whose license requires attribution. |
| `render` | string | `"inline"` | `"inline"` writes a full `<svg>` for each use. `"sprite"` writes one `<symbol>` per unique icon and references it with `<use>`. |

### Render modes

- **`inline`** (default) writes a complete `<svg>` element for each `:icon[name]`. The HTML is larger, and each icon is self-contained.
- **`sprite`** writes one hidden `<symbol>` per unique icon on each page and references it with `<svg><use href="#..."></svg>`. The HTML is smaller when the same icon appears many times on a page.

## Icon attributes

The `:icon[]` extension accepts these attributes after the name:

| Attribute | Description |
|-----------|-------------|
| `color` | Icon color, any CSS color value. Lucide icons draw with `currentColor`, so this recolors them. |
| `width` | Width in pixels. Defaults to 16. |
| `height` | Height in pixels. Defaults to 16. |
| `rotate` | Rotation in degrees. |
| `flip` | Flip direction: `horizontal`, `vertical`, or `both`. |
| `class` | CSS class added to the `<svg>` element. |
| `style` | Inline CSS added to the `<svg>` element. |
| `title` | Accessible title text. Adds a `<title>` element and `role="img"`. |
| `aria-label` | ARIA label for screen readers. Adds `role="img"`. |

There is no `size` attribute. Set `width` and `height` instead. Any other attribute passes through to the `<svg>` element unchanged.

An icon without `title` or `aria-label` is hidden from assistive technology with `aria-hidden="true"`.

## Icons on pages and sections

`sidebar.icon` in a page's frontmatter sets the icon on its sidebar entry. The same icon is also drawn before the page's `<h1>`:

```yaml title="content/docs/guides/_index.md"
---
title: Guides
sidebar:
  icon: compass
---
```

On a section's `_index.md`, this gives the section index page a heading icon that matches its sidebar group. See [Frontmatter](/reference/frontmatter/#sidebar-fields) for the sidebar fields.

A page-level `icon` field is separate from `sidebar.icon`. The default theme does not render it, and it is available to custom templates as `.Page.Params.icon`.

Templates can draw icons with the `icon` function. See [Template Functions](/reference/template-functions/) and [Configuration](/reference/configuration/theme-and-appearance/#icons) for all icon settings.
