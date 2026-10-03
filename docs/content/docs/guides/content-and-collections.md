---
title: Content and Collections
description: "Learn how folders of Markdown become auto-detected blogs, docs, courses, and other collections"
sidebar:
  order: 2
---

Collections turn folders of related Markdown files into organized areas of a site, such as a blog, documentation library, course, or reference section.

## What are collections

A *collection* is any top-level subdirectory inside `content/` that contains Markdown content.

```text
content/
  blog/
    _index.md
    photosynthesis-lab.md
  docs/
    _index.md
    lesson-planning.md
```

`blog/` and `docs/` become separate collections. Sarde uses the folder name as the collection name and URL mount, so these folders render at `/blog/` and `/docs/`. Each collection carries its own inferred defaults: sorting, layout, sidebar, table of contents, and feeds.

A known folder name only changes the defaults Sarde applies. Folders that start with `.` or `_` are ignored, and the language folders of a multilingual site are language roots, not collections (see [Internationalization](/guides/internationalization/)).

## Auto-detection rules

Sarde recognizes four collection families by directory name. Any other name gets the default behavior.

| Directory names | Type | Sort | Layout | Feed | Sidebar |
|---|---|---|---|---|---|
| `blog`, `posts`, `articles`, `news` | Blog | date (newest first) | default | yes | no |
| `docs`, `documentation`, `guides`, `reference`, `courses`, `tutorials`, `lessons`, `workshops` | Docs | order (ascending) | docs | no | yes |
| `slides`, `presentations`, `decks` | Slides | date (newest first) | default list rendered as a gallery, presentation per deck | no | no |
| `labs` | [Labs](/teaching/labs/) | order (ascending) | default at the top, labs inside a lab | no | yes (per lab) |
| Any other name | Default | title (ascending) | default | no | no |

A directory named `updates/` gets the default behavior. To give it blog or docs behavior, configure the collection in `sarde.yaml` (see [Overriding collection config](#overriding-collection-config)) instead of renaming the directory.

## Per-collection defaults

Beyond the table, each family sets these defaults.

### Blog collections

- Paginated at 10 posts per page
- RSS and Atom feeds enabled
- Previous and next links follow the sorted post order, labeled Newer and Older

### Docs collections

- Sorted by `sidebar.order` from frontmatter or a numeric filename prefix, then by title
- Collapsible sidebar, 4 levels deep
- Table of contents for H2 through H4, with scroll highlighting
- Previous and next links follow the sidebar order

### Slides collections

- The list page uses the default layout, rendered as a gallery of cards (thumbnails, slide counts, dates, tags, authors). There is no separate `gallery` layout value.
- Deck pages use `layout: presentation` with no configuration.
- Subdirectories appear as course cards on the list page.

See the [Teaching](/teaching/) section for the full guide.

### Default collections

- Sorted by title
- Default layout, with no sidebar or table of contents

## Overriding collection config

Override any inferred default in `sarde.yaml` under the `collections` key:

```yaml title="sarde.yaml"
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

Settings you leave out keep the inferred value. These settings are available per collection:

| Key | Type | Default | Description |
|---|---|---|---|
| `sort` | string | Inferred from folder name | Sort field and direction, such as `"date desc"`, `"order asc"`, or `"title asc"`. The fields are `date`, `order`, `title`, and `slug`. |
| `layout` | string | Inferred from folder name | `default`, `docs`, `splash`, `wide`, `full`, `centered`, `split`, or `presentation`. |
| `paginate` | int | `10` for blog collections, otherwise `0` | Items per page for list views. Set a positive number. `0` keeps the inferred value, so it does not turn pagination off for a blog collection. |
| `feed` | bool | `true` for blog collections, otherwise `false` | Generate RSS and Atom feeds for this collection when the feed plugins are enabled. |
| `tabs` | bool | Auto-detected | Enable or disable tabbed docs navigation. See [Tabbed Navigation](/guides/tabbed-navigation/). |
| `permalink` | string | File path URL | URL pattern for non-index pages. Write the full path, including the collection prefix (for example `/blog/:year/:slug/`). Placeholders: `:slug`, `:year`, `:month`, `:day`, `:section`, `:collection`, `:title`. |
| `sidebar` | object | Inferred for docs collections | `collapsed_by_default`, `collapse_level`, `max_depth`. See [Navigation and Sidebar](/guides/navigation-and-sidebar/#collapsible-sections). |
| `toc` | object | Inferred for docs collections | `enabled`, `depth`, `scroll_highlight`. |
| `versioning` | object | Disabled | `enabled`, `versions`, `last_version`. See [Versioning](/guides/versioning/). |

See [Configuration](/reference/configuration/content/#collections) for the complete reference.

## Sections and `_index.md`

Subdirectories within a collection become *sections*. A section can have an `_index.md` file that sets its title, order, and other metadata.

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

The `_index.md` file is optional. Without it, Sarde infers the section title from the directory name, with any numeric prefix removed and the words title-cased (`my-section/` becomes "My Section").

Set the section's sidebar position and label in the frontmatter of its `_index.md`:

```yaml title="content/docs/guides/_index.md"
---
title: Guides
sidebar:
  order: 2
  label: "How-To Guides"
---
```

Sections can nest deeper than the sidebar renders. Sarde builds a tree from the directory structure, and the sidebar then follows the collection's `sidebar.max_depth`.

## Transparent sections

A transparent section groups files on disk without adding a level to the sidebar. Its pages appear in the parent section.

Set `transparent: true` in the section's `_index.md`:

```yaml title="content/docs/internal/_index.md"
---
title: Internal
transparent: true
---
```

With this layout:

```text
content/docs/
  _index.md
  internal/                 # transparent section
    _index.md               # transparent: true
    page-a.md
    page-b.md
  other-page.md
```

→ The sidebar lists `page-a` and `page-b` beside `other-page`, with no "Internal" group. The URLs still include the directory, such as `/docs/internal/page-a/`.

Transparency changes navigation only. The section page itself still renders at `/docs/internal/`. Add `render: false` to the section's `_index.md` to skip that page when the section is only a navigation group.

## Page bundles

A page bundle is an `index.md` file (not `_index.md`) in a directory that also holds non-Markdown files. Those sibling files become assets of the page.

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

Sarde copies the assets next to the page, so `cover.jpg` is served from `/blog/my-post/cover.jpg`. Moving or renaming the bundle directory moves the page and its assets together.

## Content at the root

Markdown files placed directly in `content/` (not inside a collection directory) become standalone pages, for example `content/about.md` at `/about/`. `content/_index.md` is the homepage. See [Homepage](/guides/homepage/) to configure it.

## Node kinds

Sarde classifies every Markdown file into one of five kinds:

| Kind | File pattern | Example |
|---|---|---|
| Home | `content/_index.md` | Homepage |
| Section | `content/<collection>/<dir>/_index.md` | Section index page |
| Page | Any other `.md` file inside a collection | Regular content page |
| Bundle | `index.md` with sibling non-Markdown files | Page bundle |
| Standalone | `.md` file at the `content/` root, other than `_index.md` | About page, contact page |

An `index.md` with no sibling assets is a regular page and still takes its directory's URL. See [Frontmatter](/reference/frontmatter/) for all frontmatter fields and inference rules.
