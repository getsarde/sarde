---
title: Content and Collections
description: "Learn how folders of Markdown become auto-detected blogs, docs, courses, and other collections"
sidebar:
  order: 2
---

Collections turn folders of related Markdown files into organized areas of a
site, such as a blog, documentation library, course, or reference section.

## What are collections

A *collection* is any top-level subdirectory inside `content/` that contains
Markdown content.

```text
content/
  blog/
    _index.md
    photosynthesis-lab.md
  docs/
    _index.md
    lesson-planning.md
```

`blog/` and `docs/` become separate collections. Sarde uses the folder name as
the collection name and URL mount, so these folders render at `/blog/` and
`/docs/`.

This is a convention, not a special setup step. Any top-level folder with
Markdown content becomes a collection. Known folder names only change the
defaults Sarde applies.

## What collections contain

Collections contain the pages, sections, and navigation data for one area of a
site.

| Item | Description |
|---|---|
| Pages | Markdown files inside the collection folder. |
| Sections | Subdirectories inside the collection. A section can have its own `_index.md`. |
| Index page | The collection landing page, usually `_index.md`. |
| Navigation | Sidebar groups, breadcrumbs, and prev/next links when the layout supports them. |
| Defaults | Inferred sorting, layout, feed, sidebar, and table-of-contents behavior. |

Markdown files placed directly in `content/` are standalone pages, not
collections. The homepage `content/_index.md` is also outside any collection.

## Auto-detection rules

Sarde recognizes four collection families by directory name:

| Directory names | Type | Sort | Layout | Feed | Sidebar |
|-----------------|------|------|--------|------|---------|
| `blog`, `posts`, `articles`, `news` | Blog | date (newest first) | default | yes | no |
| `docs`, `documentation`, `guides`, `reference`, `courses`, `tutorials`, `lessons`, `workshops` | Docs | order (ascending) | docs | no | yes |
| `slides`, `presentations`, `decks` | Slides | date (newest first) | default (gallery) | no | no |
| `labs` | [Labs](/teaching/labs/) | order (ascending) | default at the top, labs inside a lab | no | yes (per lab) |
| Any other name | Default | title (ascending) | default | no | no |

Naming a directory `blog/` is a convention. Sarde auto-detects it as a date-sorted collection with feeds, but any directory name works. A directory named `updates/` would get default-type behavior unless overridden in `sarde.yaml`.

## Per-collection defaults

Each collection type comes with pre-configured behavior:

### Blog collections

- Sorted by `date` descending (newest first)
- Paginated at 10 posts per page
- RSS/Atom feeds enabled
- Prev/next links wired from the sorted post order

### Docs collections

- Sorted by `order` ascending (controlled by frontmatter `sidebar.order` or numeric filename prefix)
- Docs layout with sidebar navigation
- Collapsible sidebar with 4-level depth
- Table of contents enabled (H2 through H4)
- Prev/next links wired from the sidebar order

### Slides collections

- Sorted by `date` descending (newest first)
- Gallery list page with card grid (thumbnails, slide counts, tags, authors)
- Deck pages auto-default to `layout: presentation` (full-viewport slide viewer, no configuration needed)
- Course subdirectories render as nested galleries
- See the [Teaching](/teaching/) section for the full guide

### Default collections

- Sorted by `title` ascending
- Default layout (no sidebar, no TOC)

## Overriding collection config

Override any auto-detected default in `sarde.yaml` under the `collections` key:

`sarde.yaml`

```yaml
collections:
  blog:
    sort: "title asc"
    paginate: 20
    feed: false
  docs:
    sidebar:
      collapsed_by_default: true
      max_depth: 3
    toc:
      depth: 3
  tutorials:
    sort: "order asc"
    layout: "docs"
```

Available per-collection settings:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `sort` | string | Inferred from folder name | Sort field and direction, such as `"date desc"`, `"order asc"`, or `"title asc"`. |
| `layout` | string | Inferred from folder name | Layout type: `default`, `docs`, `splash`, `wide`, `full`, `centered`, or `split`. |
| `paginate` | int | `10` for blog collections, otherwise `0` | Items per page for list views. `0` disables pagination. |
| `feed` | bool | `true` for blog collections, otherwise `false` | Generate RSS/Atom feeds for this collection when feed plugins are enabled. |
| `tabs` | bool | Auto-detected | Enable or disable tabbed docs navigation. See [Tabbed Navigation](/guides/tabbed-navigation). |
| `permalink` | string | File path URL | URL pattern for non-index pages. |
| `sidebar` | object | Inferred for docs collections | Sidebar sub-config: `collapsible`, `collapsed_by_default`, `max_depth`, `search`. |
| `toc` | object | Inferred for docs collections | Table of contents sub-config: `enabled`, `depth`, `scroll_highlight`. |
| `versioning` | object | Disabled | Version config: `enabled`, `versions`, `last_version`. |

See [Configuration](/reference/configuration) for the complete reference.

## Sections and `_index.md`

Subdirectories within a collection become *sections*. Each section can have an `_index.md` file that defines the section's title, order, and metadata.

```text
content/docs/
  _index.md                 # Collection root
  getting-started.md
  guides/
    _index.md               # Section: "Guides"
    writing-content.md
    code-blocks.md
  reference/
    _index.md               # Section: "Reference"
    configuration.md
    frontmatter.md
```

The `_index.md` file is optional. Without it, Sarde infers the section title from the directory name and assigns a default order of 0.

Use frontmatter in `_index.md` to control the section's sidebar position and label:

```yaml
---
title: Guides
sidebar:
  order: 2
  label: "How-To Guides"
---
```

Sections can nest deeper than the sidebar renders. Sarde builds a tree from the
directory structure, then sidebar rendering follows the collection's
`sidebar.max_depth` setting.

## Transparent sections

A transparent section groups files on disk without appearing as a separate level in the sidebar. Its child pages are hoisted into the parent section.

Set `transparent: true` in the section's `_index.md`:

```yaml
---
title: Internal
transparent: true
---
```

```text
content/docs/
  _index.md
  internal/                 # transparent section
    _index.md               # transparent: true
    page-a.md
    page-b.md
  other-page.md
```

Result: The sidebar shows `page-a` and `page-b` alongside `other-page`, not nested under an "Internal" group. The `internal/` directory still organizes the files on disk.

Transparent sections affect navigation structure only. A transparent section can
still render its own section page. Add `render: false` to the section's
`_index.md` when the section acts as a navigation group without its own
page.

## Page bundles

A page bundle is an `index.md` file (not `_index.md`) placed in a directory alongside non-Markdown files. The sibling files (images, PDFs, data files) become assets of that page.

```text
content/blog/
  my-post/
    index.md                # Page bundle
    cover.jpg               # Bundle asset
    diagram.svg             # Bundle asset
```

Reference bundle assets with relative paths in the Markdown:

```markdown
![Cover image](cover.jpg)
```

Page bundles keep content and its assets together, making pages self-contained and portable.

## Content at the root

Markdown files placed directly in `content/` (not inside a collection directory) become standalone pages. The homepage `_index.md` at `content/_index.md` is a special case that renders using the configured homepage template.

## Node kinds

Sarde classifies every Markdown file into one of five kinds:

| Kind | File pattern | Example |
|------|-------------|---------|
| Home | `content/_index.md` | Homepage |
| Section | `content/<collection>/<dir>/_index.md` | Section index page |
| Page | Any `.md` file inside a collection | Regular content page |
| Bundle | `index.md` with sibling non-Markdown files | Page bundle |
| Standalone | `.md` file at `content/` root (not `_index.md`) | About page, contact page |

See [Frontmatter](/reference/frontmatter) for all available frontmatter fields and auto-inference rules.
