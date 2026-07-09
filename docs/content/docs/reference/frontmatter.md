---
title: Frontmatter
sidebar:
  order: 2
---

# Frontmatter

Frontmatter is optional metadata at the top of a Markdown file. Sarde uses it to control page titles, dates, sidebar ordering, layout, and more. Fields not provided are auto-inferred from the filename, content, or file metadata.

## Formats

Sarde supports three frontmatter formats. All three use the same field names.

:::tabs

```yaml title="YAML (---)"
---
title: My Page
date: 2024-03-15
tags:
  - tutorial
---
```

```toml title="TOML (+++)"
+++
title = "My Page"
date = 2024-03-15
tags = ["tutorial"]
+++
```

```json title="JSON ({...})"
{
  "title": "My Page",
  "date": "2024-03-15",
  "tags": ["tutorial"]
}
```

:::

If a file has no frontmatter delimiters, the entire file is treated as Markdown content.

## Core fields

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `title` | string | inferred | Page title. Inferred from the first `# H1` heading or the filename if not set. **Required** (after inference). |
| `slug` | string | inferred | URL slug for the page. Inferred from the filename with numeric prefixes stripped. |
| `date` | date | inferred | Publication date. Inferred from a `YYYY-MM-DD` filename prefix or the file modification time. |
| `updated` | date | inferred | Last modification date. Inferred from git commit date or file modification time (controlled by [`build.last_updated`](/reference/configuration#build)). |
| `publish_date` | date | - | Future publication date. Pages with a future `publish_date` are excluded unless `build.future` is enabled. |
| `expiry_date` | date | - | Expiration date. Pages past this date are excluded unless `build.expired` is enabled. |
| `aliases` | list of string | `[]` | Alternative URL paths that redirect to this page. |
| `layout` | string | - | Page layout. `default`, `docs`, `splash`, `wide`, `full`, `centered`, `split`, or `presentation`. |
| `type` | string | - | Content type identifier. |
| `template` | string | - | Template override for rendering this page. |

## Meta fields

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `draft` | bool | `false` | Mark the page as a draft. Drafts are excluded from builds unless `build.drafts` is enabled. |
| `description` | string | inferred | Page description for meta tags and feeds. Auto-derived from the first paragraph (truncated to 160 characters) if not set. |
| `image` | string | - | Featured image path for social cards and Open Graph tags. |
| `summary` | string | inferred | Page summary. Falls back to `description`, then to the first paragraph truncated to [`content.summary_length`](/reference/configuration#content) words. |
| `render` | bool | - | Whether to render this page. Treated as `true` when unset. Set to `false` to process the page in the content pipeline without generating an output file. |
| `pagefind` | bool | - | Include this page in the search index. Treated as `true` when unset. |
| `show_updated` | bool | - | Show the "last updated" date on this page. Set to `false` to suppress the updated date even when one is available. |
| `edit_url` | bool or string | - | Controls the "Edit this page" link. `false` hides it. `true` uses the site-wide [`site.edit_url`](/reference/configuration#site). A string provides a custom URL for this page. |

## Sidebar fields

Nested under the `sidebar` key.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `order` | int | inferred | Sort order in the sidebar. Inferred from a numeric filename prefix (e.g., `03-setup.md` sets order to 3). Lower values appear first. |
| `label` | string | - | Override the display label in the sidebar. Defaults to the page title. |
| `hidden` | bool | `false` | Hide this page from the sidebar. The page is still accessible via its URL. |
| `group` | string | - | Sidebar group name. Pages sharing the same group value are displayed together. |
| `badge` | string or object | - | Badge displayed next to the sidebar entry. See [badge formats](#sidebar-badge) below. |
| `icon` | string | - | Icon displayed next to the sidebar entry. |
| `attrs` | map | - | Custom HTML attributes added to the sidebar link element. |

### Sidebar badge

The `sidebar.badge` field accepts two forms.

**String form** (defaults to the `default` variant):

```yaml
sidebar:
  badge: "New"
```

**Object form** (specify variant explicitly):

```yaml
sidebar:
  badge:
    text: "Deprecated"
    variant: danger
```

| Variant | Description |
|---------|-------------|
| `default` | Neutral styling. |
| `note` | Informational. |
| `tip` | Positive/helpful. |
| `success` | Success state. |
| `caution` | Warning state. |
| `danger` | Critical/breaking. |

Legacy color aliases are supported: `green` maps to `tip`, `amber` maps to `caution`, `red` maps to `danger`.

## Table of contents fields

Nested under the `toc` key. Accepts either a boolean or an object.

**Disable the TOC** for a single page:

```yaml
toc: false
```

**Configure per-page TOC levels:**

```yaml
toc:
  enabled: true
  min_level: 2
  max_level: 3
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `enabled` | bool | - | Show or hide the table of contents. Inherits from the site-level [`toc.enabled`](/reference/configuration#toc) setting. |
| `min_level` | int | - | Minimum heading level to include. Range: 1-6. Must be &le; `max_level`. |
| `max_level` | int | - | Maximum heading level to include. Range: 1-6. |

## Navigation fields

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `prev` | bool, string, or object | - | Override the previous page link. `false` hides it. A string specifies a page slug. An object provides a custom `link` and `label`. |
| `next` | bool, string, or object | - | Override the next page link. Same forms as `prev`. |

```yaml
prev: false
next:
  link: /docs/guides/code-blocks
  label: "Code Blocks"
```

Disable the previous link:

```yaml
prev: false
```

Link by slug:

```yaml
prev: "getting-started"
```

Custom link and label:

```yaml
next:
  link: /docs/guides/code-blocks
  label: "Code Blocks"
```

## Taxonomy fields

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `tags` | list of string | `[]` | Tags assigned to the page. |
| `categories` | list of string | `[]` | Categories assigned to the page. |
| `show_tags` | bool | - | Show or hide taxonomy terms on this page. Inherits from the taxonomy configuration. |

```yaml
tags:
  - tutorial
  - deployment
categories:
  - guides
```

## Page-level fields

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `transparent` | bool | `false` | Mark this section as transparent. Transparent sections do not appear as a level in the sidebar tree; their children are promoted one level up. |
| `icon` | string | - | Page-level icon. Distinct from `sidebar.icon`, which controls the sidebar entry. |
| `head` | list | `[]` | Custom HTML tags injected into the page's `<head>`. |
| `banner` | object | - | Per-page announcement banner. See [`banner`](#banner) below. |
| `hero` | object | - | Hero section for splash layout pages. See [`hero`](#hero) below. |
| `cascade` | map | - | Arbitrary key-value pairs propagated to all descendant pages in the section tree. |
| `params` | map | - | Arbitrary user-defined data accessible in templates via `.Params`. |

### `banner`

Per-page announcement banner displayed at the top of the content area.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `content` | string | - | Banner message text. |
| `variant` | string | `"note"` | Banner style. `note`, `tip`, `caution`, or `danger`. |
| `icon` | string | - | Lucide icon name. Overrides the variant's default icon. |

```yaml
banner:
  content: "This page is under construction"
  variant: caution
  icon: construction
```

### `hero`

Hero section configuration for pages using the `splash` layout.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `title` | string | - | Hero heading text. |
| `tagline` | string | - | Hero subheading text. |
| `image` | object | - | Hero image. Has `src`, `light`, `dark`, and `alt` fields. |
| `actions` | list | - | Call-to-action buttons. |

Each entry in `hero.actions`:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `text` | string | - | Button label text. |
| `link` | string | - | Button URL. |
| `variant` | string | - | Button style variant. |
| `icon` | string | - | Icon displayed in the button. |
| `attrs` | map | - | Custom HTML attributes. Event handler attributes (`on*`) are stripped for security. |

```yaml
hero:
  title: Welcome
  tagline: A fast documentation site
  image:
    light: /images/hero-light.svg
    dark: /images/hero-dark.svg
    alt: "Illustration"
  actions:
    - text: Get Started
      link: /docs/start-here/getting-started
      variant: primary
```

### `head`

Each entry in the `head` list injects a tag into the page's `<head>` element.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `tag` | string | - | HTML tag name. Allowed values: `meta`, `link`, `script`, `style`, `noscript`, `base`. |
| `attrs` | map | - | Tag attributes as key-value pairs. |
| `content` | string | - | Tag inner content. |

```yaml
head:
  - tag: meta
    attrs:
      name: robots
      content: noindex
```

### `cascade`

A map of arbitrary key-value pairs propagated to all descendant pages in the section tree. Add `cascade` to a section's `_index.md` to apply values to every child page.

```yaml
cascade:
  draft: true
  sidebar:
    hidden: true
```

### `params`

A map of arbitrary user-defined data. Access values in templates via `.Params.key_name`. Unlike standard frontmatter fields, keys in `params` are not validated and do not produce unknown-key warnings.

```yaml
params:
  difficulty: intermediate
  estimated_time: 15
```

## Auto-inference rules

Sarde fills in missing frontmatter fields automatically. Frontmatter values always take priority. Inference only applies when a field is empty or unset.

| Field | Inference chain |
|-------|----------------|
| `title` | Frontmatter, then first `# H1` heading in content, then filename title-cased |
| `slug` | Frontmatter, then date-prefix remainder, then numeric-prefix remainder, then parent directory name (for `_index.md`), then filename slugified |
| `date` | Frontmatter, then `YYYY-MM-DD` filename prefix, then file modification time |
| `updated` | Frontmatter, then git commit date or file modification time (per [`build.last_updated`](/reference/configuration#build) strategy) |
| `sidebar.order` | Frontmatter, then numeric filename prefix |
| `description` | Frontmatter, then first paragraph of content (truncated to 160 characters) |
| `summary` | Frontmatter, then `description`, then first paragraph (truncated to [`content.summary_length`](/reference/configuration#content) words) |

### Filename patterns

Sarde parses filenames to extract dates, ordering, and slugs.

**Date prefix** (`YYYY-MM-DD-slug.md`):

```
2024-03-15-hello-world.md
```

Infers `date: 2024-03-15` and `slug: hello-world`.

**Numeric prefix** (`NN-slug.md` or `NN_slug.md`):

```
03-advanced-topics.md
```

Infers `sidebar.order: 3` and `slug: advanced-topics`.

**Combined date and numeric prefix**:

```
2024-01-15-01-intro.md
```

Infers `date: 2024-01-15`, `sidebar.order: 1`, and `slug: intro`.

**Index files** (`_index.md` or `index.md`):

The slug is derived from the parent directory name. For example, `docs/_index.md` produces the slug `docs`.

### Derived page metadata

These values are computed from the page content and are available in templates. They are not frontmatter fields and cannot be set manually.

| Field | Description |
|-------|-------------|
| `word_count` | Number of words in the content. |
| `reading_time` | Estimated reading time in minutes (words / 200, minimum 1). |

## Per-collection schema

Define a frontmatter schema for a collection by creating a `config.yaml` file in the collection directory.

`content/docs/config.yaml`

```yaml
frontmatter_schema:
  fields:
    difficulty:
      type: enum
      label: Difficulty
      required: true
      options:
        - beginner
        - intermediate
        - advanced
    estimated_time:
      type: int
      label: "Estimated time (minutes)"
      min: 1
      max: 120
```

### `FieldDef` options

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `type` | string | - | Field type. `string`, `int`, `float`, `bool`, `date`, `list`, or `enum`. Aliases: `text`/`textarea`/`color`/`image`/`url` for string, `number` for int, `toggle` for bool, `tags` for list, `select`/`radio` for enum. |
| `label` | string | - | Display label for editor UI. |
| `required` | bool | `false` | Produce a validation error if this field is missing. |
| `default` | any | - | Default value applied when the field is absent from frontmatter. |
| `min` | float | - | Minimum numeric value. |
| `max` | float | - | Maximum numeric value. |
| `max_length` | int | - | Maximum string length (in bytes). |
| `options` | list of string | - | Valid values for `enum` type fields. |

Schema validation produces warnings. It never blocks the build.

## Validation

Sarde validates built-in frontmatter fields automatically, independent of per-collection schemas. Validation issues are reported as warnings.

| Rule | Severity |
|------|----------|
| `title` is required (after inference) | error |
| `toc.min_level` and `toc.max_level` must be 1-6 | warning |
| `toc.min_level` must be &le; `toc.max_level` | warning |
| `sidebar.order` must be &ge; 0 | warning |
| `sidebar.badge.variant` must be one of: `default`, `note`, `tip`, `success`, `caution`, `danger` | warning |
| `layout` must be one of: `default`, `docs`, `splash`, `wide`, `full`, `centered`, `split`, `presentation` | warning |
| `slug` cannot be whitespace-only | warning |
| `head[].tag` must be one of: `meta`, `link`, `script`, `style`, `noscript`, `base` | warning |
| Unknown frontmatter keys produce a warning | warning |

Keys inside `cascade`, `params`, custom schema fields, and configured taxonomy names are excluded from unknown-key detection.
