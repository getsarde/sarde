---
title: Writing Content
description: "Write pages with frontmatter, images, links, drafts, scheduled content, and expiring pages"
sidebar:
  order: 3
---

Content pages are Markdown files inside the `content/` directory. Add optional frontmatter at the top of a file to set the title, date, tags, and other metadata. Sarde infers most fields when they are missing.

For the page body, [Markdown Basics](/extensions/markdown-basics/) shows standard formatting, and [Using Extensions](/extensions/using-extensions/) covers Sarde's own syntax for callouts, tabs, cards, and more. For a guided first pass through all of it, see [Write Your First Pages](/start-here/writing-your-first-pages/).

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

Every frontmatter field is optional. Without a `title`, Sarde uses the first `# Heading` in the content, or the filename converted to title case. Sarde fills in other missing fields too, as described in [Auto-inferred fields](#auto-inferred-fields).

See [Frontmatter](/reference/frontmatter/) for the complete field reference.

## Drafts, scheduled, and expiring content

Three frontmatter fields control when a page appears in the built site.

| Field | Excluded from `sarde build` when | Include it with |
|---|---|---|
| `draft: true` | Always | `--drafts` (or `-D`), or `build.drafts: true` |
| `publish_date` | The date is in the future | `--future`, or `build.future: true` |
| `expiry_date` | The date has passed | `build.expired: true` |

When a page appears locally but not in production, check its frontmatter for `draft: true` or a past `expiry_date`.

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

## Images and links

Put images shared across pages in `public/` and reference them from the site root:

```markdown
![Logo](/images/logo.png)
```

To keep a page's images and downloads next to its Markdown, make the page a [page bundle](/guides/content-and-collections/#page-bundles): a directory that contains an `index.md` file and the asset files. Reference each asset by its filename:

```markdown
![Mitosis diagram](mitosis-diagram.png)
```

:::note
`index.md` (without underscore) creates a page bundle. `_index.md` (with underscore) creates a section index.
:::

[Images and Assets](/guides/images-and-assets/) covers both placements and image processing.

Link to another page by the path to its source file. A link that starts with `/` resolves from the collection root, so in `content/docs/`, `[Setup](/guides/setup/)` points to `content/docs/guides/setup.md`. The build checks every internal link and stops on a broken one. See [Internal Links](/guides/internal-links/).

## Heading IDs and anchor links

Sarde assigns an `id` attribute to each heading in the configured range (h2 through h4 by default). The ID is the slugified heading text, so `## Getting Started` becomes `<h2 id="getting-started">`. These IDs are the targets for fragment links (`/docs/guide/#getting-started`), table of contents entries, and search index anchors.

To set a custom ID, add the `{#custom-id}` attribute:

```markdown
## My Section {#custom-id}
```

→ The heading renders as `<h2 id="custom-id">` instead of the generated slug.

When `site.heading_links` is `true` (the default), each heading also gets a clickable anchor link for sharing a direct URL to that section.

Headings outside the range get no ID, no anchor link, and no table of contents entry. To include h5 and h6 headings, widen [`markdown.toc`](/reference/configuration/content/#markdown-toc).

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

A frontmatter `date` takes priority over the filename. [Filename patterns](/reference/frontmatter/#filename-patterns) covers the rarer cases: two posts whose names differ only by the date, and filenames written entirely outside the Latin alphabet.
