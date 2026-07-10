---
title: Homepage
sidebar:
  order: 5
---

The homepage is the `content/_index.md` file. Sarde renders it using one of eight built-in templates, each designed for a different type of site. The default is `hero`.

## Choosing a template

Set the template in `sarde.yaml`:

```yaml
homepage:
  template: "hero"
```

| Template | Best for |
|----------|----------|
| `hero` (default) | Documentation sites, course portals. Large title, subtitle, CTA buttons, optional proof panel |
| `catalog` | Content directories with multiple collections |
| `minimal` | Clean single-column landing with body content |
| `dashboard` | Data-heavy sites with stat tiles |
| `portfolio` | Personal sites, project showcases |
| `landing` | Product pages with marketing-style sections |
| `marketing` | Feature grids, testimonials, pricing blocks |
| `blog` | Blog-first sites with recent posts |

## Hero template

The hero template is the default. It renders a large headline, subtitle, call-to-action buttons, and an optional proof panel (image, code snippet, or stats).

```yaml
homepage:
  template: "hero"
  hero:
    eyebrow: "Open Source"
    title: "Build Documentation Sites"
    subtitle: "A zero-config static site generator for educators and developers."
    background: "gradient"
    cta:
      label: "Get Started"
      url: "/docs/start-here/getting-started"
    secondary_cta:
      label: "View on GitHub"
      url: "https://github.com/getsarde/sarde"
```

Result: A full-width hero section with the title, subtitle, and two CTA buttons.

<!-- SCREENSHOT: homepage-hero-default - hero template with gradient background and two CTAs -->

### Hero fields

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `eyebrow` | string | - | Small text above the title |
| `title` | string | site title | Main headline. Supports inline Markdown |
| `subtitle` | string | - | Text below the title |
| `background` | string | `"gradient"` | Background style: `"gradient"`, `"solid"`, `"none"` |
| `cta` | object | - | Primary call-to-action button (`label`, `url`) |
| `secondary_cta` | object | - | Secondary button with outline style |

### Proof panel

The hero template supports a proof panel on the right side. Three panel types are available, and the first one configured is rendered:

**Image panel:**

```yaml
homepage:
  hero:
    image:
      light: /images/hero-light.svg
      dark: /images/hero-dark.svg
      alt: "Hero illustration"
```

Supports separate light/dark images, a single `src`, or raw `html` for custom content.

**Code panel:**

```yaml
homepage:
  hero:
    code:
      title: "Quick start"
      language: "sh"
      body: |
        sarde new site my-site
        cd my-site
        sarde dev
```

Result: A styled code card appears next to the hero text.

**Stats panel:**

```yaml
homepage:
  hero:
    stats:
      - value: "200+"
        label: "Languages"
      - value: "25"
        label: "Extensions"
      - value: "<1s"
        label: "Build time"
```

Result: A row of stat tiles appears next to the hero text.

## Other templates

### Catalog

Displays collection cards (docs, blog, courses) as a grid. Each card links to the collection root. Body content from `_index.md` renders above the grid.

### Minimal

A single-column page that renders body content from `_index.md` below an optional hero. Good for sites that need a custom landing page built in Markdown.

### Dashboard

Renders stat tiles and collection summaries in a dashboard layout. Suitable for project portals or internal documentation hubs.

### Portfolio

Full-width sections with alternating layouts. Body content from `_index.md` renders inline. The hero, CTA buttons, and content sections flow as a single scrollable page.

### Landing

A marketing-style landing page with full-width sections. Body content renders in a centered container between the hero and footer. No sidebar.

### Marketing

Feature grids, social proof sections, and CTA blocks. Similar to landing but with more structured sections for product pages.

### Blog

Displays recent blog posts below the hero. Pulls from the first blog-type collection found. Suitable for blog-first sites.

## Body content

All templates render the Markdown body of `content/_index.md` below the hero section (except portfolio, landing, marketing, and blog, which integrate body content into their own layouts). Write Markdown content in `_index.md` to add additional sections:

```markdown
---
title: Welcome
---

## Features

- Zero configuration
- 25 Markdown extensions
- Offline search
```

See [Configuration](/reference/configuration#homepage) for all homepage settings.
