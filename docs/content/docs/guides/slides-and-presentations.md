---
title: Slides and Presentations
description: "Write once, publish as both a website and a slide presentation."
sidebar:
  order: 14
---

Sarde can render Markdown content as a slide presentation. Write content once and publish it simultaneously as a documentation page and a full-screen slide deck.

## The presentation layout

Set a collection or individual page to use the presentation layout:

```yaml
# Per collection in sarde.yaml
collections:
  lectures:
    layout: presentation   # full-width slide viewer, no sidebar or ToC
```

```yaml
# Per page in frontmatter
---
title: "Lecture 3: Data Structures"
layout: presentation        # overrides the collection default for this page
---
```

The `presentation` layout renders content as full-width slides with no sidebar or table of contents.

## Slide separation

Slides are separated by horizontal rules (`---`) in Markdown:

````markdown
---
title: Introduction to Algorithms
layout: presentation
---

# Introduction to Algorithms

Course overview and objectives

---

## What is an Algorithm?

A step-by-step procedure for solving a problem.

- Finite number of steps
- Well-defined instructions
- Produces an output

---

## Why Study Algorithms?

Understanding algorithms helps you:

1. Write efficient code
2. Solve complex problems
3. Ace technical interviews
````

Each section between `---` markers becomes one slide.

## Dual output workflow

The same Markdown file renders in two modes:

1. **Website mode**: renders as a standard documentation page with the collection's normal layout (sidebar, ToC, prev/next navigation)
2. **Presentation mode**: renders as a full-screen slide deck

To use dual output, configure a collection with its default layout for the website and set individual pages to use the presentation layout via frontmatter:

```yaml
collections:
  lectures:
    layout: docs           # Website view uses docs layout
```

Pages with `layout: presentation` in their frontmatter render as slides instead.

## Presenter features

In presentation mode:

- **Keyboard navigation**: arrow keys or space bar advance slides
- **Full-screen**: press `F` to enter full-screen mode
- **Code highlighting**: syntax-highlighted code blocks work identically to the website view
- **Images**: page-bundled images are resolved and displayed

## Writing for both outputs

When writing content intended for dual output:

- Use clear heading hierarchy. `##` for slide titles works well in both modes.
- Keep bullet points concise. They need to be readable on a projected screen.
- Use code blocks freely. They render in both formats.
- Place `---` separators where natural slide breaks occur.
- Complex content (long tables, detailed diagrams) may need separate treatment for each format.

## Course collections

For structured courses, combine the presentation layout with other Sarde features:

```yaml
collections:
  course:
    layout: docs              # website view with sidebar navigation
    sort: sidebar_order       # lessons ordered by sidebar.order frontmatter
    prev_next:
      enabled: true
      labels: ["Previous Lesson", "Next Lesson"]
    sidebar:
      collapsible: true       # collapse sections in the sidebar
```

Use frontmatter to annotate lessons:

```yaml
---
title: "Lesson 3: Data Structures"
sidebar:
  order: 3                    # position in the sidebar
  badge:
    text: Lab                 # badge label shown next to the title
    variant: tip              # badge color variant
tags:
  - data-structures
  - beginner
params:
  difficulty: beginner        # custom metadata, accessible in templates
  estimated_time: 45min
---
```
