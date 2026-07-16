---
title: Courses with Slides
description: "Embed slide presentations inside docs or courses collections with sidebar integration"
sidebar:
  order: 5
---

Slide presentations can live inside any docs-layout collection. A "Slides" section appears in the sidebar alongside other content, and clicking a deck opens the full-screen viewer.

## The cascade pattern

Create a subdirectory for slides inside a docs or courses collection. Set `cascade: {layout: presentation}` in the section's `_index.md`:

```text
content/
  courses/
    getting-started/
      _index.md
      intro.md
      setup.md
    slides/
      _index.md           ← cascade: {layout: presentation}
      lecture-1.md
      lecture-2.md
```

`content/courses/slides/_index.md`:

```yaml
---
title: Slides
sidebar:
  order: 30
  icon: presentation
cascade:
  layout: presentation
---
```

Every page in the `slides/` section renders as a full-screen presentation. The section's own `_index.md` keeps the docs layout (cascade does not apply to the section's own index page).

## How it appears in the sidebar

The sidebar shows the "Slides" group with its children:

```
Getting Started
  Introduction
  Setup
Slides                ← section group
  Lecture 1           ← opens as full-screen slides
  Lecture 2           ← opens as full-screen slides
```

Clicking a lecture page navigates to the presentation. The close button (or `Escape`) navigates back to the section's index page.

## Sidebar ordering and icons

Control where the Slides section appears with `sidebar.order`. Add an icon with `sidebar.icon`:

```yaml
---
title: Slides
sidebar:
  order: 30
  icon: presentation
  badge:
    text: New
    variant: tip
cascade:
  layout: presentation
---
```

## Writing deck pages in a course

Deck pages inside a course collection use the same frontmatter as any other page:

```yaml
---
title: "Lecture 3: Data Structures"
sidebar:
  order: 3
tags: [data-structures]
---

# Data Structures

Overview of arrays, lists, and trees

---

## Arrays

Fixed-size, contiguous memory...
```

The `sidebar.order` controls the deck's position within the Slides group. Tags and other taxonomy fields work normally.

## Mixing slides and docs in one collection

Not every page in the slides section needs the presentation layout. Override the cascade on individual pages:

```yaml
---
title: "Lab Worksheet"
layout: default
sidebar:
  order: 5
---
```

This page renders as a normal docs page despite being inside the slides section. The cascade sets `layout: presentation` as the default; explicit frontmatter overrides it.

## Multiple slide sections

A single collection can have several slide sections:

```text
content/
  courses/
    module-1/
      _index.md
      content.md
    module-1-slides/
      _index.md           ← cascade: {layout: presentation}
      overview.md
      deep-dive.md
    module-2/
      _index.md
      content.md
    module-2-slides/
      _index.md           ← cascade: {layout: presentation}
      overview.md
```

Each slide section appears as its own group in the sidebar.

## Standalone slides collection vs embedded slides

| | Slides collection | Embedded in docs/courses |
|---|-------------------|--------------------------|
| **Gallery** | Auto-generated card grid at `/slides/` | Section index page (docs list layout) |
| **Sidebar** | No sidebar | Appears in the collection's sidebar |
| **Layout default** | Automatic `presentation` on all deck pages | Via `cascade: {layout: presentation}` in `_index.md` |
| **Best for** | Standalone presentation library | Presentations bundled with course material |
