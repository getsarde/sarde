---
title: Social Cards
description: "Auto-generate Open Graph social card images for pages without one"
sidebar:
  order: 42
---

Auto-generates 1200×630 Open Graph images for pages that do not already have one. Each card displays the page title, description, site branding (with an optional logo mark and watermark), collection name, and date, rendered with the Inter font family. Enabled by default, but images are only generated during production builds.

## How it works

During `BeforeRender`, the plugin checks each page. If the page qualifies (no existing image, not excluded), it sets the `og_image` and `twitter_image` SEO params to the generated card's URL. During `BuildDone`, the plugin renders the cards in parallel using Go's image library and writes them to the `og/` directory.

Cards are not generated during `sarde dev` (dev mode). The SEO params are still set, but the image files are only written during `sarde build`.

## Card layout

Cards use an editorial layout: branding at the top, a large title block anchored to the bottom, and generous negative space in between. Each card contains:

- **Logo mark** top-left, when a logo is configured (see Logo and watermark below)
- **Site title** in the accent color, next to the logo; when no logo is set it renders larger and bold, acting as a text wordmark
- **Background gradient**, a subtle vertical shift of the background color away from the text color (dark backgrounds deepen toward the bottom, light backgrounds brighten), so the bottom-anchored text gains contrast. `bg_gradient` replaces it with an explicit gradient or solid fill
- **Background image**, optional artwork drawn over the gradient and under everything else (see Background image below)
- **Watermark**, an optional large low-opacity rendering of the logo bleeding off the right edge
- **Page title** in bold, bottom-anchored and auto-sized (76pt, 60pt, or 48pt depending on length)
- **Description** below the title (up to 2 lines, 32pt)
- **Footer** with collection name and, for explicitly dated pages, the date (bottom-left)
- **Accent strip**, a thin partial-width line in the bottom-right corner; becomes a two-color gradient when `accent_color_2` is set

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
      accent_color_2: ""
      text_color: "#ffffff"
      logo: ""
      watermark: false
      watermark_opacity: 0.07
      cache: true
      logo_size: 64
      bg_gradient: []
      bg_image: ""
      bg_image_fit: "cover"
      bg_image_opacity: 1.0
      fonts:
        regular: ""
        bold: ""
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
| `accent_color` | String | `""` | Accent color for the strip and branding. Empty uses `theme.accent_color` from `sarde.yaml`. |
| `accent_color_2` | String | `""` | Second accent color. When set, the bottom-right accent strip becomes a horizontal gradient from `accent_color` to this value. |
| `text_color` | String | `"#ffffff"` | Text color for the title and description. |
| `logo` | String | `""` | Logo source: a path under `public/`, the literal `"sarde"` for the built-in Sarde mark, or `"none"` to opt out. Empty falls back to `site.logo`. |
| `watermark` | Boolean | `false` | Draw a large low-opacity rendering of the logo bleeding off the right edge. Ignored when no watermark source resolves. |
| `watermark_image` | String | `""` | Separate artwork for the watermark, as a path under `public/`. Empty reuses the logo. Lets a site pair a compact corner mark with a larger or line-art watermark. |
| `watermark_opacity` | Number | `0.07` | Watermark opacity (0.0-1.0). |
| `cache` | Boolean | `true` | Cache rendered cards under `.cache/social_cards/` across builds. See Caching below. |
| `logo_size` | Number | `64` | Size of the corner logo mark box in pixels, clamped to 16-256. |
| `bg_gradient` | List | `[]` | One or two hex colors replacing the automatic background gradient: two colors blend vertically from the first (top) to the second (bottom), a single color fills solid. |
| `bg_image` | String | `""` | Background image path under `public/`, same convention as `logo`. Raster formats only. |
| `bg_image_fit` | String | `"cover"` | `"cover"` crops the image to fill the whole card, `"contain"` letterboxes it inside the card. |
| `bg_image_opacity` | Number | `1.0` | Background image opacity (above 0.0, up to 1.0). Lower values dim busy artwork so the text stays readable. |
| `fonts.regular` | String | `""` | Path to a TTF or OTF file replacing the embedded Inter Regular, resolved relative to the project directory. |
| `fonts.bold` | String | `""` | Path to a TTF or OTF file replacing the embedded Inter Bold, resolved relative to the project directory. |
| `collections` | List | `[]` | Restrict card generation to these collection names. Empty generates cards for all collections. |

## Logo and watermark

The `logo` option resolves in this order:

1. `"none"`: no logo, even when `site.logo` is set.
2. `"sarde"`: the embedded Sarde mark and ribbon artwork. Used by the Sarde docs site itself.
3. Any other non-empty value: an image path under the project's `public/` directory, the same convention as `site.logo` (e.g. `logo: "/images/mark.png"`).
4. Empty (the default): fall back to the site's own `site.logo`. The dark variant is preferred since cards have dark backgrounds by default, then the light variant.

A missing or unreadable logo file never fails the build; the card renders without a logo.

### Supported image formats

| Format | Supported | Notes |
|--------|-----------|-------|
| PNG | Yes | Recommended. Transparency blends cleanly over the card background. |
| WebP | Yes | Also supports transparency. |
| JPEG | Yes | No alpha channel: the logo renders as an opaque rectangle, and the watermark becomes a ghosted rectangle rather than a ghosted shape. |
| GIF | Yes | First frame only; limited palette. |
| BMP | Yes | No transparency. |
| TIFF | Yes | |
| SVG | No | Cards have no vector rasterizer. SVG logos are skipped with a build log note. |

Prefer a transparent PNG (or WebP) around 512 to 1024 pixels: the logo mark draws at 64px (configurable via `logo_size`) and the watermark scales its file up to roughly 820px, so that range keeps both crisp. If your logo only exists as SVG, export it to PNG once and point `logo` at the exported file.

The watermark reuses whatever logo resolved: when nothing resolves, `watermark: true` is a no-op. With `logo: "sarde"`, the watermark uses a dedicated ribbon-only artwork (no background disc) so the bleed stays airy.

Set `watermark_image` to give the watermark its own artwork instead of reusing the logo. This mirrors the built-in Sarde split: a simplified, small-size-legible mark in the corner and a more detailed or line-art rendering as the watermark. A detail-dense logo often benefits from this, since marks draw at 64px where fine features disappear, while the watermark runs large where they read well.

## Per-page overrides

A page can restyle its own card with an `og_card` frontmatter block:

```yaml
---
title: Getting Started
og_card:
  bg_color: "#0d1117"
  accent_color: "#58a6ff"
  accent_color_2: ""
  text_color: ""
  hide_watermark: true
  hide_logo: false
---
```

The scope is deliberately colors and toggles only: the card's text always comes from the page's own title and description. Empty or omitted color fields fall back to the plugin config, and the `hide_*` toggles default to false (show), matching the site-wide setting. See [Frontmatter](/reference/frontmatter#og-card) for the field reference.

## Background image

Set `bg_image` to draw artwork behind the card content. The path resolves under `public/`, like `logo`, and accepts the same raster formats (no SVG). The image is drawn over the background gradient and under the watermark and all text, so content always stays on top.

- `bg_image_fit: "cover"` (the default) scales and center-crops the image to fill the full 1200×630 card.
- `bg_image_fit: "contain"` scales the image to fit inside the card and letterboxes it against the gradient.
- `bg_image_opacity` composites the image at reduced opacity. Busy or high-contrast artwork usually needs dimming (values around 0.1 to 0.3) to keep the title readable.

An opaque cover image completely hides the gradient (including a `bg_gradient` override); this is expected, not an error. Pair a dimmed image with `bg_gradient` when you want both visible.

## Theme-aware colors

When `bg_color` is not set, the plugin derives the background from the site's theme accent color (or primary color), darkened by 30% in HSL space. When `accent_color` is not set, the plugin uses the theme's accent color directly.

If neither a config override nor a theme accent color is available, the defaults are:

| Color | Default |
|-------|---------|
| Background | `#1a1a2e` (dark navy) |
| Accent | `#e94560` (coral red) |
| Text | `#ffffff` (white) |

## Fonts

The plugin embeds the Inter font family (Regular and Bold weights) as `.ttf` files. These are loaded once per build and shared across all card renders. Each parallel worker gets its own set of font faces to avoid concurrency issues.

Custom fonts replace either embedded face independently:

```yaml
plugins:
  config:
    social_cards:
      fonts:
        regular: "assets/fonts/MyFont-Regular.otf"
        bold: "assets/fonts/MyFont-Bold.otf"
```

Both TTF and OTF files work, and paths resolve relative to the project directory (they do not need to live under `public/`, since the font itself is never served). A slot whose file is missing or unparsable logs a warning and keeps the embedded Inter face; the build never fails over a font. Different fonts have different metrics, so after switching check a few cards with long titles: the auto-sizing ladder still prevents overflow, but line counts may shift.

## Caching

Rendered cards are cached across builds under `.cache/social_cards/` in the project directory, keyed by a content hash of everything that affects the card's pixels: its text, resolved colors, format and quality, logo, watermark, background image, and fonts. A rebuild where nothing changed serves every card from cache; changing any input re-renders only the affected cards. The build log reports the split, e.g. `Generated 119 social card(s) (117 from cache)`.

The cache directory is safe to delete at any time to force full regeneration, and there is no eviction. Set `cache: false` to disable caching entirely. The default project `.gitignore` already excludes `.cache/`.

## Title auto-sizing

Long titles are automatically resized to fit the card:

| Font size | Used when |
|-----------|-----------|
| 76pt | Title fits in 3 lines or fewer |
| 60pt | Title is too long at 76pt |
| 48pt | Title is too long at 60pt |

If the title still exceeds 3 lines at 48pt, the third line is truncated with "...".

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

- Cards are rendered in parallel using a worker pool sized to the available CPU cores (capped at the number of pages). Logo, watermark, and background images are decoded once per build and shared read-only across workers.
- Description text falls back to the page's auto-generated summary when no explicit description is set. HTML tags in the summary are stripped. When the summary is also empty (a body that is entirely directive blocks has no prose paragraph to extract), the card falls back to plain text taken from the rendered page content, with a leading title heading trimmed so the card does not repeat its own title.
- HTML entities in the title, description, and site title are decoded before drawing, so a title stored as `Tips &amp;amp; Tricks` renders a literal ampersand instead of the raw entity. Markdown syntax is not interpreted: `**bold**` in a title draws literally.
- The footer shows the collection title and date separated by " · ". The date only appears when it was explicitly authored, via a frontmatter `date` field or a `YYYY-MM-DD-slug` filename prefix. Dates inferred from file modification time are never shown, so docs pages without frontmatter dates do not display a misleading build date. The collection name is dropped when it matches the page title, so a collection's own index card does not repeat its title in the footer. If both parts are missing, the footer is omitted.
- PNG format produces larger files but preserves sharp text. JPEG at quality 90 is a good compromise for smaller file sizes.
- Hex colors accept both `#rgb` shorthand and `#rrggbb` full format.
