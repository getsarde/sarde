---
title: SEO and Feeds
description: "Configure meta tags, structured data, sitemaps, RSS/Atom feeds, and social cards"
sidebar:
  order: 11
---

Sarde generates meta tags, structured data, sitemaps, and feeds automatically. No configuration is needed for sensible defaults. The SEO, sitemap, feeds, and social cards plugins are all enabled by default.

## Meta tags

The SEO plugin injects Open Graph and Twitter Card meta tags into every page:

**Open Graph:**
- `og:title`, `og:description`, `og:url`, `og:type`, `og:site_name`, `og:locale`
- `og:image` (from page `image` frontmatter, social card, or `default_image` config)
- `article:published_time` and `article:modified_time` for collection pages

**Twitter Card:**
- `twitter:card` (defaults to `summary_large_image`)
- `twitter:title`, `twitter:description`, `twitter:image`
- `twitter:site` (from `twitter_handle` config)

All values are auto-populated from page frontmatter and site config. Override the description per page:

```yaml
---
title: Photosynthesis
description: "How plants convert light into chemical energy."
---
```

When `description` is not set, the SEO plugin falls back to the page summary (first paragraph, truncated).

## JSON-LD structured data

The SEO plugin emits a `<script type="application/ld+json">` block with a `@graph` containing:

| Node | When |
|------|------|
| `Article` | Pages in a collection (blog posts, docs pages) |
| `CollectionPage` | Section index pages and the homepage |
| `WebPage` | Standalone pages without a collection |
| `BreadcrumbList` | All docs-layout pages (built from route breadcrumbs) |
| `Course` | Pages with `schema_type: Course` in frontmatter |

The `BreadcrumbList` follows the same trail as the visible breadcrumbs, including collection and section titles.

## Canonical URLs

Every page gets a `<link rel="canonical">` tag pointing to its absolute URL. For versioned documentation, non-latest versions point their canonical to the latest version's URL, consolidating SEO signals on the current docs.

## Twitter Card configuration

Set the site-wide Twitter handle in plugin config:

```yaml
plugins:
  config:
    seo:
      twitter_handle: "@getsarde"
      twitter_card: "summary_large_image"
```

## Social card images

The social cards plugin auto-generates 1200x630 PNG images for every page, used as `og:image` and `twitter:image`. Cards include the page title, description, site name, collection, and date, styled with the active theme colors.

```yaml
plugins:
  config:
    social_cards:
      skip_if_image: true
      format: "png"
      quality: 90
```

When `skip_if_image` is `true` (the default), pages with an explicit `image` in frontmatter use that image instead of generating a card.

See [Plugins > Social Cards](/plugins/social-cards) for the full options reference.

## Sitemap

The sitemap plugin generates `sitemap.xml` at the site root with all published pages, `lastmod` timestamps, and configurable `changefreq` and `priority` values.

See [Plugins > Sitemap](/plugins/sitemap) for configuration.

## RSS and Atom feeds

Blog-type collections generate feeds automatically:

- RSS 2.0 at `/<collection>/feed.xml`
- Atom 1.0 at `/<collection>/atom.xml`

Feeds include the most recent posts (configurable limit) with title, link, description, published date, and full content.

Disable feeds per collection with `feed: false` in the collection config. See [Plugins > Feeds](/plugins/feeds) for feed options.

## robots.txt

The robots plugin generates `robots.txt` at the site root with a `Sitemap` directive pointing to `sitemap.xml` and a default `Allow: /` rule.

See [Plugins > Robots](/plugins/robots) for configuration.

## hreflang alternates

On multi-language sites, the SEO plugin adds `<link rel="alternate" hreflang="...">` tags for all translations of a page. This helps search engines serve the correct language version to users.

## LLMs.txt

The LLMs.txt plugin generates a machine-readable manifest at `/llms.txt` for AI and LLM consumption, listing all pages with titles and URLs.

See [Plugins > LLMs.txt](/plugins/llms-txt) for configuration.

See [Configuration](/reference/configuration#plugins) for enabling/disabling plugins, and individual plugin pages under [Plugins](/plugins/using-plugins) for detailed options.
