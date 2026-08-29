---
title: Using Themes
description: "Install, activate, and manage theme packages for a Sarde site"
sidebar:
  order: 2
---

A theme is a directory containing templates, stylesheets, and a `theme.yaml` manifest. Install one with `sarde theme add` or place it manually under `themes/`.

## Installing a theme

Install a theme from any of four source types:

```sh
# From a GitHub repository
sarde theme add github.com/user/theme-name

# From a zip URL
sarde theme add https://example.com/theme.zip

# From a tar.gz URL
sarde theme add https://example.com/theme.tar.gz

# From a local directory
sarde theme add ./path/to/theme
```

Override the theme's directory name with `--name`:

```sh
sarde theme add github.com/user/theme-name --name my-theme
```

→ The theme installs to `themes/my-theme/` regardless of the source name.

The command validates that the installed directory contains a valid `theme.yaml`. If validation fails, the installation rolls back.

## Activating a theme

Set `theme.name` in `sarde.yaml` to the directory name under `themes/`:

```yaml
theme:
  name: "my-theme"
```

→ Sarde loads templates, stylesheets, and tokens from `themes/my-theme/`.

Leave `theme.name` empty or set it to `"default"` to use the embedded theme compiled into the binary.

If `themes/my-theme/theme.yaml` is missing, Sarde falls back to the embedded tokens without a warning, while any templates under `themes/my-theme/layouts/` still apply. Run `sarde theme list` to confirm the theme is detected.

Presets and token overrides still apply on top of the active theme. Only the active theme's presets are selectable: the embedded set is listed under [Choosing a preset](/guides/themes-and-styling/#choosing-a-preset), and [Creating a Preset](/customization/creating-a-preset/) covers adding your own. See [Themes and Styling](/guides/themes-and-styling/) for token customization.

## Theme directory structure

A valid theme directory requires only `theme.yaml`. All other files are optional and override their embedded counterparts when present.

```
themes/<name>/
  theme.yaml            # Required: metadata, tokens, presets
  css/                  # Stylesheets; missing ones fall back to the embedded theme
  assets/               # Fonts, vendor scripts, JS; replaces the embedded set as a whole
  layouts/
    _default/           # Default layout templates
    _blog/              # Blog-type templates
    _docs/              # Docs-type templates
    _presentation/      # Presentation layout templates
    _labs/              # Labs layout templates
    _taxonomy/          # Taxonomy templates
    components/         # Component overrides
    partials/           # Partial overrides
    shortcodes/         # Shortcode templates
```

See [Creating a Theme](/customization/creating-a-theme/) for what goes in each directory.

## The `theme.yaml` manifest

Every theme must include a `theme.yaml` at the root of the theme directory. It declares metadata, design tokens, and optional presets.

```yaml
name: "Magazine"
slug: "magazine"
version: "1.0.0"
author: "Sarde Themes"
description: "A bold magazine-style theme for blog collections"
license: "MIT"

tokens:
  accent: "oklch(0.58 0.22 15)"
  font-sans: "'Playfair Display', Georgia, serif"
  radius-md: "0"

dark_tokens:
  bg: "oklch(0.13 0.01 15)"
  text: "oklch(0.93 0.005 15)"

presets:
  editorial:
    name: "Editorial"
    tokens:
      accent: "oklch(0.45 0.03 250)"
    dark_tokens:
      bg: "oklch(0.12 0.02 250)"
```

| Field | Description |
|-------|-------------|
| `name` | Display name |
| `slug` | Identifier, conventionally the directory name |
| `version` | Semantic version |
| `author` | Author name |
| `description` | Short description |
| `license` | License identifier |
| `tokens` | Light-mode token overrides; any key from [Theme Tokens](/reference/theme-tokens/) |
| `dark_tokens` | Dark-mode token overrides; same keys |
| `presets` | Named presets, each with its own `tokens` and `dark_tokens` maps |

No field is required. Set `name` and `slug` at minimum, since `sarde theme list` and `sarde theme info` display them.

Tokens defined in `theme.yaml` sit at layer 2 in the [token cascade](/guides/themes-and-styling/#token-cascade), between embedded defaults and preset tokens. User overrides in `sarde.yaml` always win.

## Managing themes

| Command | Description |
|---------|-------------|
| `sarde theme list` | List all themes (embedded default + installed) |
| `sarde theme info <name>` | Show metadata, token counts, and preset names |
| `sarde theme remove <name>` | Remove an installed theme. Add `--force` to remove the active theme |
| `sarde theme eject [path...]` | Copy embedded theme files (all, or only the given paths) into `themes/default/`, or into `themes/<name>/` with `--name` |

Remove an inactive theme:

```sh
sarde theme remove old-theme
```

Remove the currently active theme (resets to embedded default):

```sh
sarde theme remove my-theme --force
```

## Overriding theme files

Files in the project's `layouts/` directory override same-named files from the active theme. This allows selective customization without modifying the theme itself.

For example, to replace only the blog single-page template from a theme:

```
layouts/_blog/single.html
```

→ This file takes priority over `themes/<name>/layouts/_blog/single.html` in the template overlay chain.

See [Layouts and Templates](/customization/layouts-and-templates/#template-overlay-resolution) for the full resolution order, [Themes and Styling](/guides/themes-and-styling/) for presets and design tokens, and [Theme Tokens](/reference/theme-tokens/) for the complete token reference.
