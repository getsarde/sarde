---
title: Presentation Layout
description: "Render any page as a full-screen slide deck using the presentation layout"
sidebar:
  order: 2
---

The `presentation` layout renders a page with no site chrome (no header, footer, or sidebar) and activates the SlideViewer runtime. It works on any page in any collection.

```yaml
---
title: "Lecture 3: Data Structures"
layout: presentation
---
```

## How it works

Pages with `layout: presentation` render using the `_presentation/baseof.html` template, which contains only the page content, the SlideViewer shell markup, and the Scripts component. The [SlideViewer plugin](/plugins/slideviewer/) injects the viewer's JS and CSS assets on these pages.

Without JavaScript, presentation pages degrade to a plain readable page with a link back to the parent section.

## Setting the layout

Three ways to activate the presentation layout:

### Per page

Set `layout: presentation` in frontmatter:

```yaml
---
title: "My Presentation"
layout: presentation
---
```

### Per section via cascade

Set `cascade` in a section's `_index.md` to apply the layout to all children:

```yaml
---
title: Lectures
cascade:
  layout: presentation
---
```

Every page in this section renders as slides. The `_index.md` itself keeps its parent collection's layout (cascade does not apply to the section's own index page).

### Slides collection (automatic)

Pages in a [slides collection](/teaching/slides-collection/) default to `layout: presentation` with no configuration. This is applied after cascade resolution, so explicit frontmatter and cascade both take precedence.

## Dual-output workflow

The same Markdown content can serve as both a documentation page and a slide deck. Configure the collection with its normal layout, then set individual pages or sections to use the presentation layout:

```yaml
# sarde.yaml
collections:
  lectures:
    layout: docs
```

```yaml
# content/lectures/week-3/_index.md
---
title: "Week 3: Data Structures"
cascade:
  layout: presentation
---
```

Pages in `week-3/` render as presentations. The rest of the `lectures` collection renders as documentation with sidebar navigation.

## Relationship to the SlideViewer plugin

The layout and the plugin are independent:

- The **layout** controls the page shell (no chrome, full viewport). It is an engine feature.
- The **plugin** injects the SlideViewer runtime (JS + CSS). It gates on `layout == presentation`.

Disabling the `slideviewer` plugin removes the viewer from presentation pages. The pages still render with the presentation layout (no site chrome) but display as plain readable content.

## Template overrides

Override the presentation templates by placing files in the project's `layouts/` directory:

| File | Purpose |
|------|---------|
| `layouts/_presentation/baseof.html` | The page shell (replaces the default no-chrome shell) |
| `layouts/_presentation/single.html` | The content block (content container + viewer shell include) |
| `layouts/partials/slide-viewer-shell.html` | The viewer UI markup (nav buttons, TOC, search, modals) |
