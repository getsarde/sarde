---
title: Slides Collection
description: "Create a slides collection with a SlideShare-style gallery and auto-detected deck pages"
sidebar:
  order: 1
---

The slides collection type turns a folder of Markdown files into slide decks with a card gallery. Directories named `slides`, `presentations`, or `decks` are auto-detected.

```text
content/
  slides/
    intro-to-go.md
    testing-basics.md
    design-patterns.md
```

This produces:
- `/slides/` renders a **deck gallery** with cards showing thumbnails, titles, descriptions, slide counts, dates, and tags
- `/slides/intro-to-go/` renders the deck as a **full-screen presentation** (automatic, no frontmatter needed)

## Auto-detection

Sarde infers the slides collection type from directory names:

| Directory names | Sort | Layout | Feed |
|-----------------|------|--------|------|
| `slides`, `presentations`, `decks` | date (newest first) | default (gallery) | no |

No `sarde.yaml` configuration is required. The gallery page keeps normal site chrome (header, footer). Deck pages default to the `presentation` layout automatically.

## Deck frontmatter

Decks use standard frontmatter fields. A few additional params control slide-specific behavior:

```yaml
---
title: "Introduction to Algorithms"
description: "Sorting, searching, and complexity analysis."
date: 2026-03-10
image: /images/decks/algorithms.png
tags: [algorithms, beginner]
params:
  slide_count: 24
  slide_breakers: "hr, h2"
---
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `image` | String | (none) | Gallery card thumbnail. Decks without an image get a styled gradient placeholder. |
| `params.slide_count` | Number | (auto) | Override the auto-computed slide count badge. |
| `params.slide_breakers` | String | `"hr"` | CSS selector list of elements that start a new slide. |

## Multiple courses

Group decks into courses using subdirectories:

```text
content/
  slides/
    _index.md
    standalone-deck.md
    web-dev/
      _index.md
      lecture-1.md
      lecture-2.md
    algorithms/
      _index.md
      lecture-1.md
```

The root gallery at `/slides/` shows one card per course (with a deck count) plus any decks at the root level. Each course page (`/slides/web-dev/`) shows its own deck gallery.

Course `_index.md` files define the course title and description:

```yaml
---
title: Web Development
description: "Building web applications with Go."
---
```

## Gallery customization

Override the gallery template by placing a file at `layouts/_slides/list.html` or `layouts/slides/list.html`. The underscore-prefixed path follows the same convention as `_default` and `_docs` in the template lookup chain: it applies to every collection detected as a slides collection. The non-prefixed path is collection-specific and applies only to the collection named `slides`, taking precedence over the underscore-prefixed override when both exist. See [Template overlay resolution](/customization/layouts-and-templates/#template-overlay-resolution) for the full lookup order. The default template renders:

- **Course cards**: title, description, deck count
- **Deck cards**: thumbnail (or gradient placeholder), title, description, slide count badge, date, tags, author pills

## Overriding collection defaults

Override auto-detected defaults in `sarde.yaml`:

```yaml
collections:
  slides:
    sort: "title asc"
    paginate: 12
```

See [Content and Collections](/guides/content-and-collections/) for the full override reference.
