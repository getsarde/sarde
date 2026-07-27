---
title: Core Concepts
description: "How Sarde turns content, configuration, and a theme into a finished site."
sidebar:
  order: 2
---

Every Sarde build combines three inputs and writes one output directory.

```text
content + configuration + theme → sarde build → dist/
```

Content supplies the pages. Configuration adjusts the defaults. The theme decides how pages look. Understanding how each one is resolved explains most of Sarde's behavior.

## Content and URLs

Markdown files in `content/` become pages, and the file path becomes the URL. No routing table connects them.

| File | URL |
|---|---|
| `content/_index.md` | `/` |
| `content/about.md` | `/about/` |
| `content/docs/_index.md` | `/docs/` |
| `content/docs/photosynthesis.md` | `/docs/photosynthesis/` |
| `content/docs/lab-1/index.md` | `/docs/lab-1/` |

A file named `_index.md` is the landing page for the directory that contains it, so `content/docs/_index.md` becomes `/docs/` rather than `/docs/_index/`. Every permalink ends in a trailing slash.

Two filename patterns carry extra meaning:

- A numeric prefix sets the sidebar position and is stripped from the URL. `content/docs/01-installation.md` becomes `/docs/installation/` with `sidebar.order` set to `1`.
- A date prefix sets the publication date and is stripped from the URL. `content/blog/2024-03-15-first-post.md` becomes `/blog/first-post/`, dated 15 March 2024.

## Frontmatter

Frontmatter is an optional metadata block at the top of a Markdown file. Sarde reads YAML between `---` fences, TOML between `+++` fences, or JSON between braces.

```markdown
---
title: Photosynthesis Overview
description: How plants convert light into chemical energy
draft: false
sidebar:
  order: 2
---
```

Common fields:

| Field | Type | Default | Description |
|---|---|---|---|
| `title` | string | Inferred | Page heading and browser title |
| `description` | string | Empty | Used in metadata and link previews |
| `draft` | bool | `false` | Excluded from `sarde build`, shown by `sarde dev` |
| `date` | date | Inferred | Sort key for date-sorted collections |
| `sidebar.order` | int | Inferred | Position within the sidebar |

Omitted fields are inferred rather than left empty. `title` falls back to the first H1 in the body, then to the filename. `date` falls back to a date prefix in the filename, then to the file modification time. `sidebar.order` falls back to a numeric prefix, then to `0`. See [Frontmatter](/reference/frontmatter/) for every supported field.

## Collections

Each top-level directory inside `content/` is a collection, and Sarde infers how it should behave from its name.

| Directory names | Inferred behavior |
|---|---|
| `blog`, `posts`, `articles`, `news` | Date-sorted newest first, feed enabled, paginated at 10, Newer/Older links |
| `docs`, `documentation`, `guides`, `reference`, `courses`, `tutorials`, `lessons`, `workshops` | Docs layout, sorted by `sidebar.order`, collapsible sidebar, table of contents, Previous/Next links |
| `labs` | Labs layout for lab pages, sorted by `sidebar.order` |
| `slides`, `presentations`, `decks` | Presentation layout per deck, date-sorted |
| Any other name | Default layout, sorted by title |

These names are a convention, not a requirement. Any directory name works, and an unrecognized name produces a general collection sorted by title. To use a different name with docs behavior, set the collection explicitly in `sarde.yaml` rather than renaming the directory.

`sarde.yaml`

```yaml
collections:
  handbook:
    sort: order
    layout: docs
```

## Configuration

Configuration resolves in five layers. Later layers override earlier ones.

1. **Embedded defaults** compiled into the binary
2. **`theme.yaml`** from the active theme
3. **`sarde.yaml`** at the project root
4. **CLI flags** passed to the command
5. **`SARDE_` environment variables**

A value set nowhere falls through to the embedded default, which is why an empty `sarde.yaml` still produces a complete site. Overriding the output directory for one build takes a flag, and leaves the config file untouched:

```sh
sarde build --output public
```

To see the fully resolved configuration after all five layers merge:

```sh
sarde effective-config
```

See [Configuration](/reference/configuration/) for the full key reference.

## Themes

A theme supplies layouts, components, styles, and design tokens. Content carries no styling information, so switching themes changes the appearance of the site without touching a single Markdown file.

Sarde resolves templates by specificity, and the first match wins: a collection override in the project, then a collection template in the theme, then a project default, then a theme default, then an embedded fallback. Overriding one template therefore means copying one file into the project, not forking the theme.

See [Themes and Styling](/guides/themes-and-styling/) to customize the look.

## Developing and building

`sarde dev` runs a local server on port 4727, watches the content directory, and reloads the browser on change. CSS edits swap in without a full page reload. Drafts and expired pages are included so work in progress stays visible.

`sarde build` writes the publishable site to `dist/`. Drafts, future-dated pages, and expired pages are excluded, and broken internal links fail the build.

That difference matters when a page appears locally but not in production: the usual cause is `draft: true` in its frontmatter.
