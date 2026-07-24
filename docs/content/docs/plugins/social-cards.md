---
title: Social Cards
description: "Auto-generate Open Graph social card images for pages without one"
sidebar:
  order: 42
---

Auto-generates 1200×630 Open Graph images for pages that do not already have one. Each card displays the page title, description, site branding, collection name, and date, rendered with the Inter font family. Enabled by default, but images are only generated during production builds.

## How it works

During `BeforeRender`, the plugin checks each page. If the page qualifies (no existing image, not excluded), it sets the `og_image` and `twitter_image` SEO params to the generated card's URL. During `BuildDone`, the plugin renders the cards in parallel using Go's image library and writes them to the `og/` directory.

Cards are not generated during `sarde dev` (dev mode). The SEO params are still set, but the image files are only written during `sarde build`.

## Card layout

Each card contains:

- **Site title** in the accent color (top, small text)
- **Accent bar** (4px horizontal line)
- **Page title** in bold, auto-sized (56pt, 44pt, or 36pt depending on length)
- **Description** at 75% opacity (up to 3 lines, 22pt)
- **Footer** with collection name and date, right-aligned (bottom)
- **Bottom accent strip** (10px)

<!-- SCREENSHOT: social-card-example - a generated social card showing title, description, and footer -->

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    social_cards:
      skip_if_image: true
      format: png
      quality: 90
      bg_color: ""
      accent_color: ""
      text_color: "#ffffff"
      collections:
        - docs
        - blog
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `skip_if_image` | Boolean | `true` | Skip card generation for pages that already have an `image` field in frontmatter. |
| `format` | String | `"png"` | Output format: `"png"` or `"jpeg"` (also accepts `"jpg"`). |
| `quality` | Number | `90` | JPEG quality (1-100). Ignored for PNG output. |
| `bg_color` | String | `""` | Card background color as a hex value (e.g., `"#1a1a2e"`). Empty derives from the theme accent color (darkened by 30%). |
| `accent_color` | String | `""` | Accent color for the bar and branding. Empty uses `theme.accent_color` from `sarde.yaml`. |
| `text_color` | String | `"#ffffff"` | Text color for the title and description. |
| `collections` | List | `[]` | Restrict card generation to these collection names. Empty generates cards for all collections. |

## Theme-aware colors

When `bg_color` is not set, the plugin derives the background from the site's theme accent color (or primary color), darkened by 30% in HSL space. When `accent_color` is not set, the plugin uses the theme's accent color directly.

If neither a config override nor a theme accent color is available, the defaults are:

| Color | Default |
|-------|---------|
| Background | `#1a1a2e` (dark navy) |
| Accent | `#e94560` (coral red) |
| Text | `#ffffff` (white) |

## Font embedding

The plugin embeds the Inter font family (Regular and Bold weights) as `.ttf` files. These are loaded once per build and shared across all card renders. Each parallel worker gets its own set of font faces to avoid concurrency issues.

## Title auto-sizing

Long titles are automatically resized to fit the card:

| Font size | Used when |
|-----------|-----------|
| 56pt | Title fits in 3 lines or fewer |
| 44pt | Title is too long at 56pt |
| 36pt | Title is too long at 44pt |

If the title still exceeds 3 lines at 36pt, the third line is truncated with "...".

## Output paths

Cards are written to the `og/` directory in the build output:

| Page URL | Card path |
|----------|-----------|
| `/docs/getting-started/` | `og/docs/getting-started.png` |
| `/blog/my-post/` | `og/blog/my-post.png` |
| `/` (homepage) | `og/_index.png` |

On multi-language sites, cards include the language prefix: `og/fr/docs/getting-started.png`.

## SEO plugin interaction

The Social Cards plugin sets `og_image` and `twitter_image` in the page's `seo` params map. The SEO plugin merges into the same map without overwriting existing values, so the generated card URL is preserved regardless of plugin execution order.

## Skipping pages

A page is skipped (no card generated) when:

- `skip_if_image` is `true` and the page has a non-empty `image` frontmatter field
- The page's collection is not in the `collections` list (when `collections` is set)
- A previous plugin already set `og_image` in the page's SEO params

## Disabling the plugin

Remove `social_cards` from the enabled list:

```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    - robots
    - rss
    - atom
    - content_lint
    - link_validator
    - redirects
    - llms_txt
    - katex
    - mermaid
    # social_cards removed
```

## Edge cases

- Cards are rendered in parallel using a worker pool sized to the available CPU cores (capped at the number of pages).
- Description text falls back to the page's auto-generated summary when no explicit description is set. HTML tags in the summary are stripped.
- The footer shows the collection title and date separated by " · ". If both are missing, the footer is omitted.
- PNG format produces larger files but preserves sharp text. JPEG at quality 90 is a good compromise for smaller file sizes.
- Hex colors accept both `#rgb` shorthand and `#rrggbb` full format.
