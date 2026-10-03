---
title: Homepage
description: "Choose and configure one of eight built-in homepage templates for content/_index.md"
sidebar:
  order: 7
---

The homepage is `content/_index.md`. Sarde renders it with one of eight built-in templates, selected in `sarde.yaml`. The default is `hero`.

## Choosing a template

Set the template in `sarde.yaml`:

```yaml title="sarde.yaml"
homepage:
  template: "hero"
```

The table lists what each template renders and which `homepage.hero` keys it reads. An unrecognized template name renders `hero` without an error or warning.

| Template | Renders | `homepage.hero` keys used |
|----------|---------|---------------------------|
| `hero` (default) | Headline, subtitle, buttons, and an optional image, code, or stats panel. Body content follows. | All |
| `catalog` | Site title and a card for each collection with its description and three most recent entries. Body content follows the grid. | None |
| `minimal` | Site title, then body content. | None |
| `dashboard` | Title and subtitle, counts of collections, pages, and taxonomies, and a card for each collection listing its first five pages. Body content follows. | `title`, `subtitle` |
| `portfolio` | Title, subtitle, one button, body content, then the three most recent entries of each collection as cards. | `title`, `subtitle`, `cta` |
| `landing` | Title, subtitle, one button, body content, then a closing section that repeats the subtitle and button. | `title`, `subtitle`, `cta` |
| `marketing` | Title, subtitle, one button, a page count per collection, a card for each collection, body content, then the three most recent entries of each collection. | `title`, `subtitle`, `cta` |
| `blog` | Title as a heading and subtitle as a short bio, body content, then the five most recent entries of each collection. | `title`, `subtitle` |

Where a template lists "each collection", it covers every collection that has entries, not only blog collections. Keys a template does not use are ignored. No template has settings beyond `homepage.template` and `homepage.hero`, and `title` falls back to the site title.

## Hero template

The hero template shows a large headline, a subtitle, up to two buttons, and an optional panel beside the text.

```yaml title="sarde.yaml"
homepage:
  template: "hero"
  hero:
    eyebrow: "Spring term"
    title: "Introduction to Biology"
    subtitle: "Lessons, labs, and assignments for the spring term."
    background: "gradient"
    cta:
      label: "Start the course"
      url: "/courses/"
    secondary_cta:
      label: "Browse the labs"
      url: "/labs/"
```

→ A full-width hero section appears with the title, subtitle, and two buttons. The second button has an outline style.

<!-- SCREENSHOT: homepage-hero-default - hero template with gradient background and two CTAs -->

### Hero fields

`homepage.hero` accepts these keys:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `eyebrow` | string | - | Small text above the title. |
| `title` | string | site title | Main headline. Supports inline Markdown. |
| `subtitle` | string | - | Text below the title. |
| `background` | string | `"gradient"` | Hero background: `"gradient"`, `"solid"`, or `"none"`. Applies to the `hero` template only. |
| `cta` | object | - | Primary button. |
| `secondary_cta` | object | - | Secondary button with an outline style. |
| `image` | object | - | Image panel. Mutually exclusive with `code`. |
| `code` | object | - | Code panel. Mutually exclusive with `image`. |
| `stats` | list | - | Stat tiles. Combine with `image`, `code`, or use alone. |

Each button takes these keys:

| Key | Type | Description |
|-----|------|-------------|
| `label` | string | Button text. |
| `url` | string | Link target. A site-relative path such as `/courses/` is prefixed with `build.base_path`. A full URL is used as is. |
| `icon` | string | Optional icon name shown after the label. See [Icons](/guides/icons/). |

### Proof panel

The hero template can show a panel beside the text. Setting both `image` and `code` fails the build with `homepage.hero.code and homepage.hero.image are mutually exclusive; remove one`.

**Image panel:**

```yaml title="sarde.yaml"
homepage:
  hero:
    image:
      light: /images/hero-light.svg
      dark: /images/hero-dark.svg
      alt: "A cell diagram"
```

Use `light` and `dark` together for per-theme images, `src` for a single image, or `html` for raw HTML. If more than one is set, `html` wins, then `light` with `dark`, then `src`. Paths resolve against `public/`.

**Code panel:**

```yaml title="sarde.yaml"
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

→ A code card appears beside the hero text. Its header shows the `title` (default `Quick start`) and the `language` label. The card renders only when `body` is set.

**Stats panel:**

```yaml title="sarde.yaml"
homepage:
  hero:
    stats:
      - value: "12"
        label: "Lessons"
      - value: "4"
        label: "Labs"
      - value: "3"
        label: "Assignments"
```

→ A row of stat tiles appears beside the hero text, below the image or code panel when one is set.

## Body content

Write Markdown in `content/_index.md` to add content to the homepage:

```markdown title="content/_index.md"
---
title: Welcome
---

## What you will learn

- Cell structure and function
- Photosynthesis and respiration
- Genetics basics
```

The `hero`, `catalog`, `minimal`, and `dashboard` templates render the body in a content area below their own sections. The `portfolio`, `landing`, `marketing`, and `blog` templates place the body inside their layout, as listed in the table above.

See [Configuration](/reference/configuration/site-and-branding/#homepage) for all homepage settings.
