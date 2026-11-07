---
title: Core Concepts
description: "How Sarde turns content, configuration, and a theme into a finished site."
sidebar:
  order: 5
---

Every Sarde build combines three inputs and writes one output directory. [Getting Started](/start-here/getting-started/) and [Write Your First Pages](/start-here/writing-your-first-pages/) showed the commands; this page explains the model behind them.

```text
content + configuration + theme → sarde build → dist/
```

Content supplies the pages. Configuration adjusts the defaults. The theme decides how pages look.

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

A numeric filename prefix sets the sidebar position and is stripped from the URL. `content/docs/01-installation.md` becomes `/docs/installation/` with `sidebar.order` set to `1`.

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

Omitted fields are inferred rather than left empty. `title` falls back to the first H1 in the body, then to the filename. `date` falls back to the file modification time. `sidebar.order` falls back to a numeric prefix, then to `0`. See [Frontmatter](/reference/frontmatter/) for every supported field.

## Collections

Each top-level directory inside `content/` is a collection, and Sarde infers how it behaves from its name.

| Directory names | Inferred behavior |
|---|---|
| `blog`, `posts`, `articles`, `news` | Date-sorted newest first, feed enabled, paginated at 10, Newer/Older links |
| `docs`, `documentation`, `guides`, `reference`, `courses`, `tutorials`, `lessons`, `workshops` | Docs layout, sorted by `sidebar.order`, collapsible sidebar, table of contents, Previous/Next links |
| `labs` | Labs layout for lab pages, sorted by `sidebar.order` |
| `slides`, `presentations`, `decks` | Date-sorted; deck pages use the presentation layout, the list page is a card gallery |
| Any other name | Default layout, sorted by title |

The names are a convention, and the behavior they select is a default. To give a directory with another name the docs behavior, configure the collection in `sarde.yaml` instead of renaming the directory:

```yaml title="sarde.yaml"
collections:
  handbook:
    sort: order
    layout: docs
```

See [Content and Collections](/guides/content-and-collections/) for the full per-family defaults and every collection setting.

## Configuration

`sarde.yaml` at the project root changes the defaults. Every key has a default, so the file is optional and lists only the values that differ. A project with an empty `sarde.yaml`, or none at all, still builds a complete site.

A command-line flag overrides a setting for one run and leaves the file unchanged. To write one build to `public/` instead of `dist/`, run:

```sh
sarde build --output public
```

To see every setting with the value Sarde uses, after defaults, `sarde.yaml`, flags, and [environment variables](/reference/configuration/environment-variables/) are combined, run:

```sh
sarde effective-config
```

[Configuration](/reference/configuration/) lists every key and explains the order in which these sources override each other.

## Themes

A theme supplies layouts, components, styles, and design tokens. Content carries no styling information, so switching themes changes the appearance of the site without editing any Markdown file.

To change one part of a layout, copy that one template into the project instead of forking the whole theme. [Layouts and Templates](/customization/layouts-and-templates/) explains which file wins when several exist.

See [Themes and Styling](/guides/themes-and-styling/) to customize the look.

## Developing and building

`sarde dev` runs a local server on port 4727, watches the project files, and reloads the browser on change. CSS edits swap in without a full page reload.

`sarde build` writes the publishable site to `dist/`. It also checks internal links and anchors, and most broken ones stop it. [Internal Links](/guides/internal-links/#links-that-do-not-resolve) lists the cases that only warn.

Drafts appear in `sarde dev` and are left out of `sarde build`. [Drafts, scheduled, and expiring content](/guides/writing-content/#drafts-scheduled-and-expiring-content) explains which pages each command includes.
