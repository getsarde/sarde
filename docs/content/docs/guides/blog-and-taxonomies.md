---
title: Blog and Taxonomies
description: "Set up date-sorted blog collections with feeds, list layouts, tags, and categories"
sidebar:
  order: 8
---

Put posts in a `blog/`, `posts/`, `articles/`, or `news/` directory and Sarde treats it as a blog: newest first, paginated, with RSS and Atom feeds. No configuration is required. Tags, categories, and authors group posts into browsable listing pages.

## Blog collection setup

Create a `blog/` directory inside `content/` with an `_index.md` and one file per post:

```text
content/blog/
  _index.md
  hello-world.md
  new-release.md
```

Give each post a `date`. The date controls sort order:

```yaml title="content/blog/hello-world.md"
---
title: Hello World
date: 2026-03-15
tags: [announcements]
---
```

The collection defaults to posts sorted newest first, 10 posts per page, Newer and Older links between posts, and feeds enabled. The [auto-detection rules](/guides/content-and-collections/#auto-detection-rules) list every directory name Sarde recognizes.

## List layouts

The blog index at `/blog/` renders with one of three templates. Set the template with the `template` field in the collection's `_index.md`:

```yaml title="content/blog/_index.md"
---
title: Blog
template: "blog/list-grid"
---
```

| Template | Description |
|----------|-------------|
| `blog/list` (default) | Post cards with date, reading time, tags, and authors. Posts with `featured: true` also appear in a Featured section above the list. Shows a tag sidebar and a paginator. |
| `blog/list-grid` | Bordered card grid with an optional cover image (the post's `image` field). Shows a tag sidebar and a paginator. |
| `blog/list-minimal` | One line per post with title and date, and a paginator. |

## Single post layouts

Posts render with `blog/single` unless the post's frontmatter sets another template:

```yaml title="content/blog/hello-world.md"
---
title: Hello World
template: "blog/single-cover"
image: cover.jpg
---
```

| Template | Description |
|----------|-------------|
| `blog/single` (default) | Header with date, reading time, authors, and tags, then the content and Newer and Older links. |
| `blog/single-cover` | Shows the post's `image` as a full-width cover above the header. |
| `blog/single-wide` | Widens the content area to 56rem for media-heavy posts. |

## Pagination

The blog index shows 10 posts per page. Later pages live at `/blog/page/2/`, `/blog/page/3/`, and so on. Change the page size in `sarde.yaml`:

```yaml title="sarde.yaml"
collections:
  blog:
    paginate: 20
```

`paginate` takes a positive integer. A value of `0` is ignored and the default of 10 applies, so to show every post on one page, set a number larger than the post count.

## Taxonomies

A taxonomy groups content by shared terms. Sarde enables `tags` by default. Each taxonomy generates a listing page at `/<taxonomy>/` and a page for each term at `/<taxonomy>/<term>/`. Term pages paginate at `/<taxonomy>/<term>/page/2/`.

### Default configuration

The default configuration is:

```yaml title="sarde.yaml"
taxonomies:
  tags: "tag"
```

Add tags to a post in its frontmatter:

```yaml
---
title: Photosynthesis Lab
tags: [biology, lab-work, plants]
---
```

→ The post appears on `/tags/biology/`, `/tags/lab-work/`, and `/tags/plants/`. Each term page lists every post with that tag.

### Custom taxonomies

Add taxonomies under `taxonomies` in `sarde.yaml`:

```yaml title="sarde.yaml"
taxonomies:
  tags: "tag"
  categories: "category"
  authors: "author"
```

Assign terms in frontmatter with the taxonomy name as the key:

```yaml
---
title: Photosynthesis Lab
tags: [biology, plants]
categories: [science]
authors: [dr-chen]
---
```

The default theme links `authors` terms from each post and from blog index cards. It shows `tags` as chips on the page. It does not show `categories` on posts, but the category listing and term pages are still generated.

### Taxonomy options

Replace the short form with an object to set options:

```yaml title="sarde.yaml"
taxonomies:
  tags:
    paginate_by: 20
    undefined_tags: "warn"
    show_tags: true
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `singular` | string | taxonomy name | Singular name of the taxonomy. The default theme does not use it, and URLs always use the taxonomy key (`/tags/`). |
| `paginate_by` | int | `10` | Items per page on term pages. Must be at least 1. |
| `undefined_tags` | string | `"warn"` | How to treat terms used in content but missing from `data/<taxonomy>.yml`: `warn` logs a warning, `error` fails the build, `ignore` skips the check, `create` accepts them silently. The check runs only when the data file exists. |
| `render` | bool | `true` | Generate the listing and term pages. |
| `show_tags` | bool | `true` | Show tag chips on pages. Applies to the `tags` taxonomy only. A post's own `show_tags` frontmatter field overrides it. |

### Term metadata

Define display names, descriptions, icons, and colors for terms in `data/<taxonomy>.yml`. Key each entry by the term's slug (the lowercase, hyphenated form of the term):

```yaml title="data/tags.yml"
biology:
  label: "Biology"
  icon: "leaf"
  color: "green"
lab-work:
  label: "Lab Work"
```

| Field | Description |
|-------|-------------|
| `label` | Display name. Defaults to the term as written in content. |
| `description` | Text shown under the term page title. |
| `color` | Accent color for the term's chips and its pill on the listing page. |
| `icon` | Lucide icon name drawn on tag chips. |
| `hidden` | Leaves the term out of the tag sidebar and the listing page. |
| `priority` | Orders the listing page. Higher values come first. |
| `permalink` | Replaces the term's slug in its URL. |

### Slug collisions

Terms are keyed by slug, so two terms that slugify to the same string become one. Sarde warns and continues the build.

When two term names reduce to the same slug, the warning reads:

```text
taxonomy "tags": terms "Lab Work" and "lab work" collide on slug "lab-work"
```

Their pages merge under whichever name was seen first, so the winner depends on content order. Pick one spelling.

When a `permalink` in `data/<taxonomy>.yml` points a term at a slug another term already uses, the warning starts with:

```text
taxonomy "tags": permalink "biology" of "Life Sciences" collides with an existing slug
```

The pages of the displaced term merge into the existing entry, and the displaced term gets no page of its own. Change the permalink in the data file.

## Feeds

Blog collections generate an RSS feed and an Atom feed with the 20 most recent posts:

- RSS 2.0 at `/<collection>/feed.xml`
- Atom 1.0 at `/<collection>/atom.xml`

Turn feeds off for a collection:

```yaml title="sarde.yaml"
collections:
  blog:
    feed: false
```

See [Feeds](/plugins/feeds/) for the feed limit and the per-collection plugin options, and [SEO and Feeds](/guides/seo-and-feeds/) for the rest of the site metadata. The full list of taxonomy settings is in [Configuration](/reference/configuration/content/#taxonomies), and per-page fields are in [Frontmatter](/reference/frontmatter/#taxonomy-fields).
