---
title: Project Structure
description: "Understand what each directory and config file in a Sarde project does"
sidebar:
  order: 1
---

A Sarde project is a directory with a `sarde.yaml` file and a `content/` folder. Everything else is optional. This page explains what each directory and config file does.

## Directory layout

A typical project looks like this after scaffolding with `sarde new site`:

```text
my-site/
  sarde.yaml              # Site configuration (required)
  kazari.config.yaml      # Code block highlighting settings
  content/                # All Markdown content (required)
    _index.md             # Homepage
    blog/
      _index.md
      hello-world.md
    docs/
      _index.md
      getting-started.md
  public/                 # Files copied as-is to output
    images/
  .gitignore
```

Only `sarde.yaml` and `content/` are required. Sarde ignores missing directories without errors.

## Directories

### `content/`

All Markdown content lives here. Each top-level subdirectory becomes a *collection* (e.g., `blog/`, `docs/`, `courses/`). Files at the root of `content/` become standalone pages.

See [Content & Collections](/guides/content-and-collections) for how collections are detected and configured.

### `public/`

Files in `public/` are copied to the output directory without processing. Use it for images, fonts, PDFs, or any file that does not need transformation.

A file at `public/images/logo.png` is available at `/images/logo.png` in the built site.

### `layouts/`

Custom template overrides. Sarde ships a complete embedded theme, so this directory is only needed when overriding default templates. The override resolution order (most specific wins):

1. `layouts/<collection>/` (e.g., `layouts/_docs/single.html`)
2. `layouts/_default/`
3. Theme layouts (if using an external theme)
4. Embedded default layouts

See [Layouts & Templates](/guides/layouts-and-templates) for details on template resolution and available layout types.

### `assets/`

CSS and JavaScript files for bundling. Sarde processes these through esbuild, producing fingerprinted output files in production builds. Files here are not copied as-is (use `public/` for that).

### `themes/`

External themes installed via `sarde theme add`. Each theme is a subdirectory containing its own `theme.yaml`, layouts, and assets. Most projects use the built-in default theme and do not need this directory.

### `data/`

YAML data files accessible in templates. Taxonomy term metadata (custom slugs, descriptions) is loaded from files matching `data/<taxonomy-name>.yml`.

### `i18n/`

Translation files for UI strings. Each file is named by language code (e.g., `i18n/fr.yaml`, `i18n/es.yaml`). Sarde merges these with embedded defaults and theme translations in a three-layer cascade: embedded, theme, then project (project wins).

See [Internationalization](/guides/internationalization) for the full i18n workflow.

### `icons/`

Local SVG icon files. Place `.svg` files here and reference them by filename with the `:icon[name]` extension. This is the default `icons.local_dir` location.

See [Icons](/guides/icons) for icon sets and configuration.

### `directives/`

Custom `:::` block directives. Each directive is a `<name>.yaml` schema plus a `<name>.html` template, with an optional `<name>.css` sidecar that is bundled into the site stylesheet automatically. Scaffold one with `sarde new directive <name>`.

See [Custom Directives](/extensions/custom-directives) for the schema and template data.

## Config files

### `sarde.yaml`

The main site configuration file. Controls site metadata, theme settings, build options, plugin toggles, collection overrides, and all other site-wide settings. This is the only required config file (beyond having a `content/` directory).

See [Configuration](/reference/configuration) for every available key.

### `kazari.config.yaml`

Code block highlighting configuration. Controls syntax themes, toolbar buttons (copy, fullscreen, wrap), line numbers, frame detection, and language-specific defaults. This file is optional. Without it, Sarde uses sensible defaults.

See [Code Blocks](/guides/code-blocks) for code block features and syntax.

### `theme.yaml`

Theme-specific configuration. Only present inside a theme directory (`themes/<name>/theme.yaml`). Defines preset token overrides and theme-level layout defaults. Values from `theme.yaml` sit between embedded defaults and `sarde.yaml` in the config cascade.

See [Themes & Styling](/guides/themes-and-styling) for theme presets and token overrides.

### `sidebar.yaml`

Per-entry sidebar overrides, placed at the project root. Relabels, reorders, hides, or un-hides individual sidebar entries without editing their frontmatter, and adjusts the tab bar on tabbed collections. It is the highest-precedence sidebar layer, above `sarde.yaml`.

See [Navigation & Sidebar](/guides/navigation-and-sidebar#overrides-with-sidebar-yaml) for the override schema.

### `nav.yaml`

Manual sidebar navigation overrides. Only applicable to tabbed docs collections. Sarde auto-generates sidebar navigation from the directory structure by default, so this file is rarely needed.

See [Navigation & Sidebar](/guides/navigation-and-sidebar) for sidebar configuration.

### `config.yaml` (per-collection)

Per-collection schema definitions placed inside a collection directory (e.g., `content/docs/config.yaml`). Defines required frontmatter fields, types, and validation rules for pages in that collection.

See [Frontmatter](/reference/frontmatter) for per-collection schema options.

## Output directories

### `dist/`

The build output directory. Created by `sarde build`. Contains the complete static site ready for deployment. Override the name with `build.output` in `sarde.yaml`.

### `.cache/`

Build cache for processed images and dev-mode render results. Safe to delete at any time. Sarde regenerates cached files on the next build.

Both `dist/` and `.cache/` are excluded from version control by the scaffolded `.gitignore`.
