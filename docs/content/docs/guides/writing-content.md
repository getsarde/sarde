---
title: Writing Content
description: "Cover frontmatter fields, page bundles, drafts, scheduled content, and expiring pages"
sidebar:
  order: 3
---

Content pages are Markdown files inside the `content/` directory. Add optional frontmatter at the top of the file to set the title, date, tags, and other metadata. Sarde infers most fields automatically when they are missing.

## Frontmatter basics

Frontmatter is a metadata block at the top of a Markdown file. The most common fields:

```yaml
---
title: Photosynthesis
description: How plants convert light to chemical energy
date: 2026-03-15
tags: [biology, plants]
draft: false
---

Markdown content starts here.
```

All frontmatter fields are optional. Without a `title`, Sarde uses the first `# Heading` in the content, or the filename converted to title case.

See [Frontmatter](/reference/frontmatter) for the complete field reference.

## Frontmatter formats

Sarde supports three frontmatter formats. YAML is the most common.

:::tabs

== YAML

```yaml
---
title: Photosynthesis
tags: [biology, plants]
---
```

== TOML

```toml
+++
title = "Photosynthesis"
tags = ["biology", "plants"]
+++
```

== JSON

```json
{
  "title": "Photosynthesis",
  "tags": ["biology", "plants"]
}
```

:::

The format is detected from the opening delimiter: `---` for YAML, `+++` for TOML, `{` for JSON. All three produce identical results.

## Auto-inferred fields

Sarde fills in missing frontmatter from the filesystem and content. This is how zero-config works: create a file, write Markdown, and Sarde handles the rest.

| Field | Inference chain |
|-------|-----------------|
| `title` | Frontmatter, then first `# Heading`, then filename title-cased |
| `slug` | Frontmatter, then filename (numeric prefix stripped, slugified) |
| `date` | Frontmatter, then date from filename (`2026-03-15-my-post.md`), then file modification time |
| `updated` | Frontmatter, then git commit date or file modification time (per `build.last_updated` strategy) |
| `order` | Frontmatter `sidebar.order`, then numeric filename prefix (e.g., `01-intro.md` sets order to 1) |

### Numeric filename prefixes

Prefix a filename with a number to control sidebar order without frontmatter:

```text
content/docs/
  01-getting-started.md     # order: 1, slug: "getting-started"
  02-installation.md        # order: 2, slug: "installation"
  03-configuration.md       # order: 3, slug: "configuration"
```

The numeric prefix is stripped from the URL. `01-getting-started.md` produces `/docs/getting-started/`, not `/docs/01-getting-started/`.

### Date-prefixed filenames

Blog-style filenames with a date prefix (`YYYY-MM-DD-slug.md`) set both the date and slug:

```text
content/blog/
  2026-03-15-hello-world.md    # date: 2026-03-15, slug: "hello-world"
  2026-04-01-new-release.md    # date: 2026-04-01, slug: "new-release"
```

A date in frontmatter takes precedence over the filename date.

## Page bundles

A page bundle is a directory containing an `index.md` file alongside related assets (images, PDFs, data files). The assets become part of the page and can be referenced with relative paths.

```text
content/blog/
  cell-biology/
    index.md                  # Page bundle
    mitosis-diagram.png       # Bundle asset
    data.csv                  # Bundle asset
```

Reference assets with relative paths in the Markdown:

```markdown
![Mitosis diagram](mitosis-diagram.png)
```

Page bundles keep a page and its assets together, making content portable. Moving the directory moves everything.

:::note
`index.md` (without underscore) creates a page bundle. `_index.md` (with underscore) creates a section index. These are different things.
:::

## Drafts, scheduled, and expiring content

Three mechanisms control when content appears in the built site:

### Drafts

```yaml
---
draft: true
---
```

Draft pages are excluded from production builds. The dev server includes them by default (use `--no-drafts` to exclude). Set `build.drafts: true` in `sarde.yaml` to include drafts in production builds.

### Scheduled content

```yaml
---
publish_date: 2026-06-01
---
```

Pages with a future `publish_date` are excluded until that date passes. The dev server includes them by default (use `--no-future` to exclude).

### Expiring content

```yaml
---
expiry_date: 2026-12-31
---
```

Pages with a past `expiry_date` are excluded from builds. This is useful for time-limited announcements or seasonal content.

All three filters apply during production builds. The dev server includes draft and future content by default so authors can preview it.

## Heading IDs and anchor links

Sarde automatically assigns an `id` attribute to headings in the configured range (h2 through h4 by default). The ID is a slugified version of the heading text, e.g., `## Getting Started` becomes `<h2 id="getting-started">`. These IDs serve as fragment link targets (`/docs/guide/#getting-started`), table of contents entries, and search index anchors.

To assign a custom ID, use the `{#custom-id}` attribute syntax:

```markdown
## My Section {#custom-id}
```

This produces `<h2 id="custom-id">` instead of the auto-generated slug.

When `site.heading_links` is `true` (the default), each heading also gets a clickable anchor link for sharing direct URLs to that section.

### Configuring the heading range

By default, only h2 through h4 (`##` to `####`) are processed. To include h5 and h6 headings:

```yaml title="sarde.yaml"
markdown:
  toc:
    min_heading_level: 2
    max_heading_level: 6
```

Headings outside this range get no ID, no anchor link, and no TOC entry. See [Configuration](/reference/configuration#markdown-toc) for details.

:::tip
This setting controls heading *extraction* (which headings get IDs and become link targets). A separate [`toc.min_level` / `toc.max_level`](/reference/configuration#toc) setting controls which extracted headings appear in the table of contents sidebar. Set both if you want h5/h6 headings in your TOC.
:::
