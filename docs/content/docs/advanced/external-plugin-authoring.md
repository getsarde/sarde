---
title: Authoring External Plugins
description: "Package a plugin.yaml manifest with assets and templates for Sarde sites to install"
sidebar:
  order: 4
---

:::note
This page covers external plugins: declarative `plugin.yaml` packages installed with `sarde plugin install`, no code required. For Go plugins compiled into the `sarde` binary using lifecycle hooks, see [Writing Plugins](/advanced/writing-plugins/). The `manifest.yaml` format described in that page's client-side section is internal to Sarde's bundled plugins and is not something plugin authors write.
:::

:::caution[Manifest format is provisional]
The `plugin.yaml` schema shipped shortly before Sarde 1.0 and has not yet been exercised by a wide range of plugins. It is **not** covered by Sarde's 1.x compatibility promise: field names and defaults may still change in a 1.x release while the format settles. Changes will be announced in the [changelog](/resources/changelog/) with a migration path. Pin the Sarde version you build against if you publish a plugin.
:::

An external plugin is a directory of static files plus one manifest. Sarde reads the manifest at build time and wires the declared assets and templates into the site. Because no plugin code executes, a plugin cannot break or slow down a build; the worst a bad manifest can do is get itself skipped with a warning.

## Plugin anatomy

```
grade-sync/
  plugin.yaml            # manifest (required)
  config.yaml            # default option values (optional)
  blueprint.yaml         # option metadata for settings UIs (optional)
  assets/                # files copied to the build output (optional)
    css/grade-sync.css
    js/grade-sync.js
  templates/             # template contributions (optional)
    partials/
    components/
    shortcodes/
```

Only `plugin.yaml` is required. The directory name must exactly equal the manifest's `slug`; installation enforces this, and a manually placed mismatch is skipped with a build warning.

## The plugin.yaml manifest

A complete manifest for a premium plugin:

`plugin.yaml`
```yaml
name: Grade Sync
slug: grade-sync
version: 2.1.0
description: Sends quiz results to an external gradebook
author: example-author
homepage: https://example.com/grade-sync
premium: true
purchase_url: https://example.com/buy/grade-sync

inject:
  when: collection
  collection: courses
  styles: [css/grade-sync.css]
  module_scripts: [js/grade-sync.js]

output:
  prefix: assets/vendor/grade-sync/
  include: [js/, css/]
```

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name shown in `sarde plugin list` and install output. |
| `slug` | Yes | Unique identifier. Lowercase letters, digits, `-` and `_`; must start and end with a letter or digit. Must equal the directory name. |
| `version` | Yes | Plugin version, dotted numeric (`2.1.0`). Checked against a license's `max_version`. |
| `description` | No | One-line summary. |
| `author` | No | Author name. |
| `homepage` | No | Project or documentation URL. |
| `premium` | No | `true` requires a valid license before the plugin activates. Default `false`. |
| `purchase_url` | No | Where to buy a license. Shown in install output and license warnings. |
| `inject` | No | Conditional per-page asset injection. See below. |
| `output` | No | Which files under `assets/` are copied to the build output, and where. See below. |

Unknown top-level fields are rejected: a manifest with a misspelled key fails to load entirely instead of silently ignoring the typo.

## Conditional asset injection

The `inject` block declares which pages receive the plugin's CSS and JavaScript:

| `when` value | Injects on |
|--------------|-----------|
| `always` | Every page. |
| `layout` | Pages with the layout named in `inject.layout` (for example `presentation`). |
| `collection` | Pages in the collection named in `inject.collection` (for example `courses`). |
| `has_sidebar` | Pages with sidebar navigation. |
| `has_toc` | Pages with a table of contents and headings. |
| `has_headings` | Pages with any headings. |
| `has_code_blocks` | Pages with fenced code blocks. |
| `has_images` | Pages with images. |
| `has_prev_next` | Pages with previous/next navigation. |
| `is_content_page` | Regular content pages, not section indexes. |
| `has_updated` | Pages with a last-updated date. |

Three asset lists control what is injected, all holding paths relative to the plugin's `assets/` directory:

- `styles`: rendered as `<link rel="stylesheet">`
- `scripts`: rendered as `<script defer>`
- `module_scripts`: rendered as `<script type="module">`

Leaving `when` empty while listing any assets implies `always`. Site owners can override the condition per site with `plugins.config.<slug>.always: true`; see [External Plugins](/plugins/external-plugins/#forcing-injection).

## Copying assets to the build output

The `output` block controls the copy from the plugin's `assets/` tree into the built site:

| Key | Default | Description |
|-----|---------|-------------|
| `prefix` | `assets/vendor/<slug>/` | Destination directory inside the build output. |
| `include` | entire `assets/` tree | Allowlist of paths to copy. Entries ending in `/` match a directory and everything below it; other entries match one file. |

An asset at `assets/css/grade-sync.css` ends up at `dist/assets/vendor/grade-sync/css/grade-sync.css` and is referenced as `/assets/vendor/grade-sync/css/grade-sync.css`. Sites hosted under a subdirectory get the base path prepended automatically; injected URLs never need manual prefixing.

## Shipping default configuration

Two optional files provide defaults for the plugin's options:

`config.yaml`
```yaml
start_date: "2026-09-01"
message: "Next cohort starts {date}. Enroll now."
```

`blueprint.yaml`
```yaml
fields:
  start_date:
    type: string
    label: "Start date"
    hint: "First day of the next cohort, shown in the banner"
    default: "2026-09-01"
  message:
    type: string
    label: "Banner message"
    default: "Next cohort starts {date}. Enroll now."
```

`config.yaml` is a flat map of default values. `blueprint.yaml` adds field metadata (`type`, `label`, `hint`, `default`, and `min`/`max` for numbers) consumed by the Sarde desktop app for a future plugin settings UI; its `default` values also act as a base layer of configuration.

The effective value of an option resolves as: `blueprint.yaml` defaults, overridden by `config.yaml`, overridden by the site's `plugins.config.<slug>` in `sarde.yaml`. See [`plugins` configuration](/reference/configuration#plugins).

## Contributing templates

Files in `templates/partials/`, `templates/components/`, and `templates/shortcodes/` join the site's template resolution with this priority:

```
user layouts/  >  theme  >  plugin  >  embedded defaults
```

A plugin can therefore ship new partials, components, or shortcodes, or replace embedded defaults, while the site owner and theme always keep the last word. A file at `templates/shortcodes/gradebook.html` becomes a `gradebook` shortcode usable in any Markdown page of the host site. See [Shortcodes](/reference/shortcodes) for the invocation syntax.

When two installed plugins ship a template with the same filename, the alphabetically last slug wins and the build warns:

```
template shortcodes/gradebook.html is provided by multiple plugins (grade-sync, quiz-widgets); quiz-widgets wins
```

## Developing and testing locally

Develop a plugin directly inside a test site: create `plugins/<slug>/` by hand and run `sarde dev`. The dev server watches `plugins/` and runs a full rebuild on every change, so manifest edits, asset changes, and template changes all show up on save.

To distribute, zip the plugin directory (a wrapping top-level folder inside the zip is fine) or push it to a GitHub repository. Consumers install it with the commands described in [External Plugins](/plugins/external-plugins/#installing-a-plugin).

## What external plugins can't do

The manifest model is deliberately limited to presentation. An external plugin cannot:

- Add Markdown syntax or Goldmark extensions
- Transform content, inject pages, or read the content model
- Register dev-server routes
- Run build-time computation such as image generation or index building

If a plugin needs any of these, it needs the Go lifecycle hooks described in [Writing Plugins](/advanced/writing-plugins/), which means shipping as a built-in rather than an external package.

## Selling a premium plugin

Two manifest fields make a plugin premium:

```yaml
premium: true
purchase_url: https://example.com/buy/grade-sync
```

With `premium: true`, the plugin stays inactive on any site that lacks a valid license file, and Sarde's warnings point buyers at `purchase_url`. The whole license mechanism (file format, verification, install commands, CI usage) is described in [Premium Plugins](/plugins/premium-plugins/).

Sarde has no built-in marketplace: distribution, payment, and license issuing are the author's own responsibility.

## Edge cases

- A slug matching any built-in plugin name is rejected at install time and skipped with a warning when placed on disk directly.
- Asset paths in `inject` and `output` reject backslashes, absolute paths, and any `..` segment. A manifest failing these checks fails to load as a whole, not just for the offending field.
- One plugin's malformed manifest, reserved slug, or failed license check never blocks the rest of the build; only that plugin is skipped.
- `include` entries are matched against forward-slash paths relative to `assets/`, regardless of the operating system the site builds on.
