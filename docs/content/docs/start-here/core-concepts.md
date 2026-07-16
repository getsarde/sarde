---
title: Core Concepts
description: "Sarde combines content, configuration, and a theme to generate a static site."
sidebar:
  order: 3
---

```text
content + configuration + theme → sarde build → dist/
```

## Content

Markdown files contain page content. Frontmatter adds metadata such as the title, date, draft status, and sidebar order.

## Collections

Each top-level directory inside `content/` is a collection. Common directory names receive specialized defaults.

| Directory | Default behavior |
|---|---|
| `blog`, `posts`, `articles`, `news` | Date-sorted blog |
| `docs`, `documentation`, `courses`, `tutorials` | Documentation navigation |
| Any other name | General content collection |

## Configuration

Sarde starts with embedded defaults. Themes, `sarde.yaml`, command-line flags, and `SARDE_` environment variables can override them.

```bash
sarde build --output public
```

See [Configuration](/reference/configuration/) for the full configuration reference.

## Themes

Themes provide layouts, components, styles, and design tokens. Changing a theme does not change the Markdown source.

See [Themes and Styling](/guides/themes-and-styling/) to customize the site.

## Development and builds

Use `sarde dev` for local authoring with live reload. Use `sarde build` to generate the publishable site.
