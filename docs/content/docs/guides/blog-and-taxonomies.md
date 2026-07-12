---
title: Blog and Taxonomies
description: "Set up date-sorted blog collections with feeds, list layouts, tags, and categories"
sidebar:
  order: 7
---

Blog collections are auto-detected by directory name. Place content in `blog/`, `posts/`, `articles/`, or `news/` and Sarde applies date-sorted, feed-enabled defaults with no configuration required.

## Blog collection setup

Create a `blog/` directory inside `content/`:

```text
content/blog/
  _index.md
  2026-03-15-hello-world.md
  2026-04-01-new-release.md
```

Sarde auto-detects this as a blog collection with date-sorted posts (newest first), pagination at 10 posts per page, and RSS/Atom feeds enabled.

Any of these directory names trigger blog-type detection: `blog`, `posts`, `articles`, `news`. See [Content & Collections](/guides/content-and-collections) for the full auto-detection table.

## List layouts

Blog list pages (the index at `/blog/`) render with one of three layouts. Set the layout with the `template` field in `_index.md` frontmatter:

```yaml
---
title: Blog
template: "blog/list-grid"
---
```

| Layout | Description |
|--------|-------------|
| `blog/list` (default) | Post list with optional featured posts, reading time, tags, and paginator |
| `blog/list-grid` | Bordered card grid with optional cover images |
| `blog/list-minimal` | Minimal layout showing title and date per post |

Without a `template` override, blog list pages use `blog/list`.

## Single post layouts

Individual blog posts render with one of three layouts. Set the layout with the `template` field in the post's frontmatter:

```yaml
---
title: Hello World
template: "blog/single-cover"
image: cover.jpg
---
```

| Layout | Description |
|--------|-------------|
| `blog/single` (default) | Centered header with date, reading time, tags, content, and newer/older navigation |
| `blog/single-cover` | Full-width cover image bleed at the top of the page |
| `blog/single-wide` | Wider content area (56rem) for media-heavy posts |

Without a `template` override, blog posts use `blog/single`.

## Pagination

Blog list pages paginate automatically. Override the default of 10 per page in `sarde.yaml`:

```yaml
collections:
  blog:
    paginate: 20
```

Set `paginate: 0` to disable pagination and show all posts on a single page.

## Taxonomies

Taxonomies group content by shared terms. Sarde ships with `tags` enabled by default. Each taxonomy generates a listing page at `/<taxonomy>/` and a term page for each value at `/<taxonomy>/<term>/`.

### Default configuration

The default `sarde.yaml` includes:

```yaml
taxonomies:
  tags: "tag"
```

The value (`"tag"`) is the singular form used in URL slugs. Adding tags to a post:

```yaml
---
title: Photosynthesis Lab
tags: [biology, lab-work, plants]
---
```

Result: The post appears on `/tags/biology/`, `/tags/lab-work/`, and `/tags/plants/`. Each term page lists all posts with that tag.

### Custom taxonomies

Add additional taxonomies in `sarde.yaml`:

```yaml
taxonomies:
  tags: "tag"
  categories: "category"
  authors: "author"
```

Assign terms in frontmatter using the taxonomy name as the key:

```yaml
---
title: Photosynthesis Lab
tags: [biology, plants]
categories: [science]
authors: [dr-chen]
---
```

### Taxonomy options

Each taxonomy supports per-taxonomy configuration:

```yaml
taxonomies:
  tags:
    singular: "tag"
    paginate_by: 20
    undefined_tags: "warn"
    render: true
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `singular` | string | taxonomy name | Singular form for URL slugs |
| `paginate_by` | int | `0` | Items per page on term listing pages. `0` uses the collection default |
| `undefined_tags` | string | `""` | How to handle terms not defined in `data/<taxonomy>.yml`: `"warn"` or `""` (ignore) |
| `render` | bool | `true` | Generate HTML pages for this taxonomy |

### Term metadata

Define term metadata (display name, description, icon, color) in a YAML file at `data/<taxonomy-name>.yml`:

```yaml title="data/tags.yml"
biology:
  label: "Biology"
  icon: "leaf"
  color: "green"
lab-work:
  label: "Lab Work"
  icon: "flask-conical"
```

Term metadata controls how tags appear in the sidebar tag cloud and on tag chips throughout the site.

## Feeds

Blog collections generate RSS and Atom feeds automatically:

- RSS 2.0 at `/<collection>/feed.xml`
- Atom 1.0 at `/<collection>/atom.xml`

Disable feeds per collection:

```yaml
collections:
  blog:
    feed: false
```

See [Configuration](/reference/configuration#taxonomies) for all taxonomy settings, and [Frontmatter](/reference/frontmatter#taxonomy-fields) for per-page taxonomy fields.
