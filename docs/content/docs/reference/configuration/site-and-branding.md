---
title: Site and Branding
description: "Site identity, social links, header, footer, head tags, and homepage settings in sarde.yaml"
sidebar:
  order: 2
---

Set the site identity, the links in the header and footer, extra tags in `<head>`, and the homepage hero.

## `site`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `title` | string | `"My Site"` | Site title. Appears in the browser tab and header. **Required.** |
| `description` | string | `""` | Site description. Used in meta tags and feeds. Recommended. |
| `url` | string | `""` | Production URL (e.g., `https://example.com`). Used for canonical links, sitemaps, and feeds. Recommended. |
| `language` | string | `"en"` | Default language code (BCP 47). **Required.** |
| `logo` | string or object | - | Site logo, rendered in the header before the site title. Accepts a single path string (used for both themes) or an object with `light`, `dark`, `alt`, and `replaces_title` fields. |
| `favicon` | string | `""` | Path to the favicon file relative to `public/`. |
| `edit_url` | string | `""` | Base URL for "Edit this page" links. Append the content file path to this URL. Example: `https://github.com/user/repo/edit/main/content`. |
| `title_delimiter` | string | `"\|"` | Separator between page title and site title in the browser tab. |
| `heading_links` | bool | `true` | Add anchor links to headings. |
| `custom_404` | string | `""` | Path to a custom 404 page template. |

### `site.logo` (object form)

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `light` | string | `""` | Logo path for light mode. |
| `dark` | string | `""` | Logo path for dark mode. |
| `alt` | string | `""` | Alt text for the logo image. |
| `replaces_title` | bool | `false` | Visually hide the site title text so only the logo shows. The title stays in the DOM for screen readers. |

```yaml
site:
  logo:
    light: /img/logo-light.svg
    dark: /img/logo-dark.svg
    alt: My Site
    replaces_title: true
```

Paths are relative to `public/` and resolve against `build.base_path`. Size the logo with the [`logo-height`](/reference/theme-tokens#layout) token.

See [Branding](/guides/branding) for variant behavior, image formats, and favicon setup.

## `social`

A list of social links displayed in the header and footer.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `label` | string | - | Display label (e.g., `"GitHub"`). |
| `url` | string | - | Full URL to the social profile. |
| `icon` | string | - | Icon name (e.g., `"github"`, `"twitter"`). |

```yaml
social:
  - label: GitHub
    url: https://github.com/user/repo
    icon: github
```

## `header`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `search` | bool | `true` | Show the search button in the header. The index and modal are still built; use `search.enabled` to remove search entirely. |
| `theme_toggle` | bool | `true` | Show the light/dark mode toggle. |
| `social` | bool | `true` | Show social links in the header. |
| `links` | list | `[]` | Navigation links in the header. |

### Header/footer links

Each entry in `header.links` or `footer.links`:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `label` | string | - | Display text. |
| `url` | string | - | Link URL. |
| `external` | bool | `false` | Open in a new tab. |

## `footer`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `text` | string | `""` | Footer text (supports Markdown). |
| `links` | list | `[]` | Footer navigation links. Same format as [header links](#header-footer-links). |
| `credits` | bool | `true` | Show "Powered by Sarde" credit line. |

## `head`

Inject custom tags, CSS, and JavaScript into every page's `<head>`.

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `tags` | list | `[]` | Custom HTML tags to inject. |
| `custom_css` | list | `[]` | Paths to additional CSS files (relative to `assets/`). |
| `custom_js` | list | `[]` | Paths to additional JavaScript files (relative to `assets/`). |

### Head tags

Each entry in `head.tags`:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `tag` | string | - | HTML tag name: `meta`, `link`, `script`, `style`, `noscript`, or `base`. Other names are skipped. |
| `attrs` | map | - | Tag attributes as key-value pairs. |
| `content` | string | - | Tag inner content (for tags like `<script>`). The content of `script`, `style`, and `noscript` is written as given, so quotes in CSS or JavaScript stay intact. |

Site tags are written on every page, before the tags a page sets in its own `head` frontmatter.

```yaml
head:
  tags:
    - tag: meta
      attrs:
        name: google-site-verification
        content: abc123
  custom_css:
    - custom/styles.css
  custom_js:
    - custom/analytics.js
```

## `homepage`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `template` | string | `"hero"` | Homepage template. Available templates: `hero`, `catalog`, `minimal`, `dashboard`, `portfolio`, `landing`, `marketing`, `blog`. |

### `homepage.hero`

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `eyebrow` | string | `""` | Small text above the title. |
| `title` | string | `""` | Hero heading text. |
| `subtitle` | string | `""` | Hero subheading text. |
| `background` | string | `"gradient"` | Background style for the hero section. |
| `cta` | object | - | Primary call-to-action button. Has `label` and `url` fields. |
| `secondary_cta` | object | - | Secondary call-to-action button. Same format as `cta`. |
| `stats` | list | - | Statistics to display. Each entry has `value` and `label` fields. |
| `code` | object | - | Code block to display in the hero. Has `title`, `language`, and `body` fields. **Mutually exclusive with `image`.** |
| `image` | object | - | Image to display in the hero. Has `src`, `light`, `dark`, `alt`, and `html` fields. **Mutually exclusive with `code`.** |

```yaml
homepage:
  template: hero
  hero:
    title: "Build fast sites"
    subtitle: "A static site generator that ships as a single binary."
    background: gradient
    cta:
      label: "Get Started"
      url: /docs/start-here/getting-started
    image:
      light: /images/hero-light.svg
      dark: /images/hero-dark.svg
      alt: "Hero illustration"
```
