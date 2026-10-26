---
title: Writing Content
description: "Write pages with frontmatter, page bundles, drafts, scheduled content, and expiring pages"
sidebar:
  order: 3
---

Content pages are Markdown files inside the `content/` directory. Add optional frontmatter at the top of a file to set the title, date, tags, and other metadata. Sarde infers most fields when they are missing.

## Create a page

Create the file by hand, or let `sarde new` write it with starter frontmatter:

```sh
sarde new docs "Lesson Planning"
```

→ Sarde creates `content/docs/lesson-planning.md` containing a `title`, the current `date`, and `draft: true`.

The new page is a draft, so `sarde build` leaves it out. Delete the `draft: true` line when the page is ready to publish. See [Drafts](#drafts).

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

Every frontmatter field is optional. Without a `title`, Sarde uses the first `# Heading` in the content, or the filename converted to title case.

See [Frontmatter](/reference/frontmatter/) for the complete field reference.

## Frontmatter formats

Sarde reads three frontmatter formats. YAML is the most common.

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

The opening delimiter selects the format: `---` for YAML, `+++` for TOML, `{` for JSON. All three produce the same page.

## Auto-inferred fields

Sarde fills in missing frontmatter from the filesystem and the content, so a file with only Markdown still builds into a complete page.

| Field | Inference chain |
|---|---|
| `title` | Frontmatter, then first `# Heading`, then filename title-cased |
| `slug` | Frontmatter, then filename (date and numeric prefixes stripped, slugified) |
| `date` | Frontmatter, then a `YYYY-MM-DD-` filename prefix, then file modification time |
| `updated` | Frontmatter, then git commit date or file modification time (per `build.last_updated`) |
| `order` | Frontmatter `sidebar.order`, then numeric filename prefix (for example `01-intro.md` sets order to 1), then `0` |

The page URL comes from the filename, not from `slug`. A frontmatter `slug` changes the URL only when the collection sets a `permalink` pattern that uses `:slug` (see [Content and Collections](/guides/content-and-collections/#overriding-collection-config)).

### Numeric filename prefixes

Prefix a filename with a number and a hyphen or underscore to control sidebar order without frontmatter:

```text
content/docs/
  01-getting-started.md     # order: 1, slug: "getting-started"
  02-installation.md        # order: 2, slug: "installation"
  03-configuration.md       # order: 3, slug: "configuration"
```

The prefix is stripped from the URL: `01-getting-started.md` produces `/docs/getting-started/`, not `/docs/01-getting-started/`. Directory names keep their prefix in the URL, so `content/docs/01-setup/install.md` is served at `/docs/01-setup/install/`.

### Date filename prefixes

A blog-style name that starts with a date sets the page's date, and the date is stripped from the URL:

```text
content/blog/
  2026-03-15-hello-world.md   # date: 2026-03-15, served at /blog/hello-world/
  2026-04-02-01-intro.md      # date: 2026-04-02, order: 1, served at /blog/intro/
```

Sarde 1.5 and earlier kept part of the date in the URL: `2026-03-15-hello-world.md` was served at `/blog/03-15-hello-world/`. The build redirects that old URL to the current one, the same way as a page's [`aliases`](/plugins/redirects/#page-aliases), so published links keep working. A page, an alias or a [global redirect](/plugins/redirects/#global-redirects) at the old path takes priority. Collections with a `permalink` pattern get no redirect, because their URLs never contained the date prefix.

A frontmatter `date` takes priority over the filename. Two posts in one folder whose names differ only by the date, such as `2025-01-10-recap.md` and `2026-01-10-recap.md`, resolve to the same URL. The build warns about the collision and keeps the first; rename one, or give the collection a `permalink` pattern with `:year`.

A filename made only of characters outside the Latin alphabet, such as `日本語.md`, has no slug to strip down to. Its slug and URL use the first 8 hex characters of a hash of the name instead: `/blog/a1b2c3d4/`. For a readable URL, name the file in Latin characters, or set `slug` in frontmatter and give the collection a `permalink` pattern with `:slug`.

## Page bundles

To keep a page's images and downloads next to its Markdown, make the page a [page bundle](/guides/content-and-collections/#page-bundles): a directory that contains an `index.md` file and the asset files. Reference each asset by its filename:

```markdown
![Mitosis diagram](mitosis-diagram.png)
```

:::note
`index.md` (without underscore) creates a page bundle. `_index.md` (with underscore) creates a section index.
:::

## Drafts, scheduled, and expiring content

Three frontmatter fields control when a page appears in the built site.

| Field | Excluded from `sarde build` when | Include it with |
|---|---|---|
| `draft: true` | Always | `--drafts` (or `-D`), or `build.drafts: true` |
| `publish_date` | The date is in the future | `--future`, or `build.future: true` |
| `expiry_date` | The date has passed | `build.expired: true` |

### Drafts

Mark a page as a draft:

```yaml
---
draft: true
---
```

`sarde build` excludes draft pages. `sarde dev` includes them by default; pass `--no-drafts` to exclude them. To publish drafts in a build, pass `--drafts` or set `build.drafts: true` in `sarde.yaml`.

### Scheduled content

Set `publish_date` to hold a page back until a date:

```yaml
---
publish_date: 2026-06-01
---
```

Both `sarde build` and `sarde dev` exclude a page with a future `publish_date` until that date passes. Pass `--future` to either command to include it.

### Expiring content

Set `expiry_date` to retire a page on a date:

```yaml
---
expiry_date: 2026-12-31
---
```

`sarde build` excludes a page once its `expiry_date` has passed, which suits time-limited announcements and seasonal content. `sarde dev` still shows expired pages so you can edit them.

When a page appears locally but not in production, check its frontmatter for `draft: true` or a past `expiry_date`.

## Heading IDs and anchor links

Sarde assigns an `id` attribute to each heading in the configured range (h2 through h4 by default). The ID is the slugified heading text, so `## Getting Started` becomes `<h2 id="getting-started">`. These IDs are the targets for fragment links (`/docs/guide/#getting-started`), table of contents entries, and search index anchors.

To set a custom ID, add the `{#custom-id}` attribute:

```markdown
## My Section {#custom-id}
```

→ The heading renders as `<h2 id="custom-id">` instead of the generated slug.

When `site.heading_links` is `true` (the default), each heading also gets a clickable anchor link for sharing a direct URL to that section.

### Configuring the heading range

To include h5 and h6 headings, widen the range:

```yaml title="sarde.yaml"
markdown:
  toc:
    min_heading_level: 2
    max_heading_level: 6
```

Headings outside the range get no ID, no anchor link, and no table of contents entry. See [Configuration](/reference/configuration/content/#markdown-toc) for details.

:::tip
`markdown.toc` controls which headings get IDs and become link targets. A separate [`toc.min_level` and `toc.max_level`](/reference/configuration/theme-and-appearance/#toc) setting controls which of those headings appear in the table of contents sidebar. Set both to show h5 and h6 headings in the table of contents.
:::
