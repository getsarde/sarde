---
title: Project Structure
description: "Understand what each directory and config file in a Sarde project does"
sidebar:
  order: 1
---

A Sarde project is a directory with a `content/` folder. Everything else is optional. The sections below describe what each directory and config file is for, and which ones Sarde generates.

## Directory layout

`sarde new site` scaffolds this layout:

```text
my-site/
  sarde.yaml              # Site configuration
  kazari.config.yaml      # Code block settings
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

Only `content/` is required. A project without a `sarde.yaml` builds with the embedded defaults, and Sarde skips any other missing directory without an error. A build with no `content/` directory fails.

Sarde ignores directories inside `content/` whose names start with `.` or `_`, and files whose names start with `.`.

## Directories

Sarde reads these directories from the project root. Only `content/` is required.

### `content/`

All Markdown content lives here. Each top-level subdirectory becomes a *collection* (for example `blog/`, `docs/`, `courses/`). Files at the root of `content/` become standalone pages.

See [Content and Collections](/guides/content-and-collections/) for how collections are detected and configured.

### `public/`

Files in `public/` are copied to the output directory without processing. Use it for images, fonts, PDFs, or any file that needs no transformation.

A file at `public/images/logo.png` is available at `/images/logo.png` in the built site.

### `layouts/`

Template overrides. Sarde ships a complete embedded theme, so create this directory only to override a default template. Sarde uses the first match in this order:

1. `layouts/<collection>/` in the project, for example `layouts/blog/single.html`
2. `themes/<name>/layouts/<collection>/` in the active theme
3. The layout-type directory (`_docs`, `_blog`, `_slides`, `_presentation`, `_labs`), project first, then theme
4. `layouts/_default/` in the project, then the theme's `_default/`
5. The embedded theme

See [Layouts and Templates](/customization/layouts-and-templates/) for template names and layout types.

### `assets/`

CSS and JavaScript source files. Sarde bundles the files you list under `head.custom_css` and `head.custom_js` in `sarde.yaml` through esbuild and fingerprints the output filenames in production builds. A file in `assets/` that no setting lists is not written to the output. Use `public/` for files that need no bundling.

See [Images and Assets](/guides/images-and-assets/#css-and-js-bundling) for the lookup order and configuration.

### `themes/`

External themes installed with `sarde theme add`. Each theme is a subdirectory with its own `theme.yaml`, layouts, and assets. Most projects use the built-in default theme and do not need this directory.

### `data/`

Data files for templates. A template reads `data/<name>.yaml`, `.yml`, `.json`, or `.toml` with the `data` function. Taxonomy term metadata (custom slugs, descriptions) loads from `data/<taxonomy-name>.yml`.

### `i18n/`

Translation files for UI strings, one file per language code (`i18n/fr.yaml`, `i18n/es.yaml`). Sarde merges them with the embedded defaults and the theme's translations in three layers: embedded, theme, then project. The project wins.

See [Internationalization](/guides/internationalization/) for the full workflow.

### `icons/`

Local SVG files. Reference one by filename with the `:icon[name]` extension. `icons/` is the default value of `icons.local_dir`.

See [Icons](/guides/icons/) for icon sets and configuration.

### `directives/`

Custom `:::` block directives. Each directive is a `<name>.yaml` schema plus a `<name>.html` template, with an optional `<name>.css` file that Sarde bundles into the site stylesheet. Scaffold one with `sarde new directive <name>`.

See [Custom Directives](/extensions/custom-directives/) for the schema and template data.

### `plugins/`

External plugins, one subdirectory per plugin with a `plugin.yaml` manifest. See [External Plugins](/plugins/external-plugins/).

## Config files

Every config file is optional. Each one lives at the project root unless its entry says otherwise.

### `sarde.yaml`

The main site configuration file, at the project root. It controls site metadata, theme settings, build options, plugin toggles, collection overrides, and every other site-wide setting. The file is optional, because every key has an embedded default.

See [Configuration](/reference/configuration/) for every key.

### `kazari.config.yaml`

Code block toolbar and display settings, at the project root. The file is optional.

See [Code Blocks](/guides/code-blocks/) for code block syntax and settings.

### `theme.yaml`

Theme metadata, design tokens, and presets. It lives inside a theme directory (`themes/<name>/theme.yaml`). Its values sit between the embedded defaults and `sarde.yaml` in the config cascade.

See [Themes and Styling](/guides/themes-and-styling/) for presets and token overrides.

### `sidebar.yaml`

Per-entry sidebar overrides, at the project root next to `sarde.yaml`. It relabels, reorders, hides, or un-hides individual sidebar entries without editing their frontmatter, and adjusts the tab bar on tabbed collections. It is the highest-precedence sidebar layer, above `sarde.yaml`.

See [Navigation and Sidebar](/guides/navigation-and-sidebar/#overrides-with-sidebar-yaml) for the override schema.

### `nav.yaml`

A hand-written sidebar tree for one tab of a tabbed collection, for example `content/docs/guides/nav.yaml`. A `nav.yaml` anywhere else is ignored. Sarde builds the sidebar from the directory structure by default, so most projects never need this file.

See [Navigation and Sidebar](/guides/navigation-and-sidebar/#manual-tab-sidebar-with-nav-yaml) for the format.

### `config.yaml` (per collection)

A frontmatter schema for one collection, placed inside the collection directory (for example `content/docs/config.yaml`). Its `frontmatter_schema` key defines custom fields with types, defaults, and validation rules. Schema violations produce warnings and never block the build.

See [Frontmatter](/reference/frontmatter/#per-collection-schema) for the schema options.

## Generated directories

Sarde creates these directories at the project root.

### `dist/`

The build output, written by `sarde build`. It contains the complete static site, ready to deploy. Set a different directory with `build.output` in `sarde.yaml` or `--output` on the command. Each build removes files in the output directory that the new build did not write (`build.clean` is `true` by default), so do not store files there by hand.

### `.cache/`

The build cache: rendered pages, processed images, and generated social cards. Delete it at any time. The next build recreates it.

### `.sarde/`

Local state, including the external link check cache (`.sarde/linkcache.json`) and project license files.

The scaffolded `.gitignore` excludes `dist/`, `.cache/`, and `.sarde/` from version control.
