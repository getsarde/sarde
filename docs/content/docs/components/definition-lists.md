---
title: "Definition Lists"
description: "Term-and-definition pairs using `dl` markup."
sidebar:
  order: 28
---

Definition lists render term-definition pairs using semantic `<dl>`, `<dt>`, and `<dd>` HTML elements. They suit glossaries, config references, and any structured vocabulary.

## Basic Usage

Write the term on its own line, then indent the definition with `:`:

```md
Frontmatter
: Metadata block at the top of a Markdown file, delimited by `---`.

Collection
: A directory whose name matches a known pattern (`blog`, `docs`, `posts`, etc.). Sarde applies layout and sorting rules automatically.

Weight
: An integer that controls the order of pages in a docs-layout sidebar. Lower values appear first.
```

Frontmatter
: Metadata block at the top of a Markdown file, delimited by `---`.

Collection
: A directory whose name matches a known pattern (`blog`, `docs`, `posts`, etc.). Sarde applies layout and sorting rules automatically.

Weight
: An integer that controls the order of pages in a docs-layout sidebar. Lower values appear first.

## Multiple Definitions per Term

A term can have more than one definition:

```md
Layout
: The template used to render a page.
: Determined by the collection type — `blog`, `docs`, or `default`.
```

Layout
: The template used to render a page.
: Determined by the collection type — `blog`, `docs`, or `default`.

## Multiple Terms per Definition

Place multiple terms above a single definition block:

```md
`sarde dev`
`sarde serve`
: Starts the local development server on port 4727.
```

`sarde dev`
`sarde serve`
: Starts the local development server on port 4727.
