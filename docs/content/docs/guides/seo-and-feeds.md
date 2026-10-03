---
title: SEO and Feeds
description: "Configure meta tags, structured data, sitemaps, RSS/Atom feeds, and social cards"
sidebar:
  order: 12
---

A default build already emits meta tags, structured data, a sitemap, `robots.txt`, `llms.txt`, feeds, and social card images. The `seo`, `sitemap`, `robots`, `rss`, `atom`, `llms_txt`, and `social_cards` plugins are all enabled by default, so the work left is setting the site URL and per-page descriptions and images. Turn any of them off in [Plugins and checks](/reference/configuration/plugins-and-checks/#plugins).

Set `site.url` in `sarde.yaml` to the address the site is published at. Canonical links, sitemap entries, feed links, and social card URLs are built from it:

```yaml title="sarde.yaml"
site:
  url: "https://example.com"
```

## Meta tags

The `seo` plugin adds Open Graph and Twitter Card meta tags to every page:

- **Open Graph:** `og:title`, `og:description`, `og:url`, `og:type`, `og:site_name`, `og:locale`, and `og:image` with its alt text. Pages in a collection get `og:type` set to `article` plus `article:published_time` and `article:modified_time`. On multi-language sites, `og:locale:alternate` lists the translations.
- **Twitter Card:** `twitter:card` (default `summary_large_image`), `twitter:title`, `twitter:description`, `twitter:image`, and `twitter:site` when a handle is configured.

Values come from the page's frontmatter and the site configuration. Set the description and image per page:

```yaml
---
title: Photosynthesis
description: "How plants convert light into chemical energy."
image: /images/photosynthesis.png
---
```

The image resolves in this order: the page's `image`, a generated [social card](#social-card-images), then the `default_image` option.

When `description` is not set, the plugin uses the page summary (the first prose paragraph, truncated; code fences and directive blocks are skipped). If the body has no prose at all, such as a homepage built only from directive blocks, it uses text taken from the rendered page. Set `auto_description: false` to turn the fallback off.

To control indexing for one page, set `robots` in its frontmatter:

```yaml
---
title: Internal Notes
robots: "noindex,nofollow"
---
```

Pagination pages after the first (`/blog/page/2/` and later) get `noindex,follow` automatically and are left out of the sitemap.

## Site-wide SEO options

Set the Twitter handle, card type, and a fallback image in the `seo` plugin config:

```yaml title="sarde.yaml"
plugins:
  config:
    seo:
      twitter_handle: "@getsarde"
      twitter_card: "summary_large_image"
      default_image: "/images/og-default.png"
```

See [SEO](/plugins/seo/) for every option and for the full list of generated tags.

## JSON-LD structured data

The `seo` plugin writes a `<script type="application/ld+json">` block containing a `@graph` of these nodes:

| Node | When |
|------|------|
| `Article` | Pages in a collection (blog posts, docs pages) |
| `CollectionPage` | Section index pages and the homepage |
| `WebPage` | Standalone pages without a collection |
| `BreadcrumbList` | Pages whose breadcrumb trail has two or more entries |
| `Course` | Pages with `schema_type: Course` under `params` |

The `BreadcrumbList` follows the visible breadcrumbs, including collection and section titles. `Article` reads its author from `params.author`, and `Course` reads its provider from `params.provider`. Set all three under `params` in frontmatter:

```yaml
---
title: Photosynthesis
params:
  schema_type: Course
  provider: "Riverside High School"
  author: "Dr. Chen"
---
```

## Canonical URLs

Every page gets a `<link rel="canonical">` tag with its absolute URL. In versioned documentation, pages of older versions point their canonical to the matching page in the latest version, which consolidates search ranking on the current docs.

## hreflang alternates

On multi-language sites, the default theme adds a `<link rel="alternate" hreflang="...">` tag for every translation of a page, plus an `x-default` entry for the default language. Search engines use them to serve the right language version.

## Social card images

The `social_cards` plugin generates a 1200x630 PNG for each page that has no `image` of its own, and uses it as `og:image` and `twitter:image`. A card shows the page title, description, site name, the collection name, and, for pages with an explicit date, the date, in the active theme colors. A logo mark and a watermark are optional.

Cards are written to `og/` during `sarde build`. `sarde dev` does not write them. The plugin depends on `seo`, which supplies the title and description tags that accompany the image.

```yaml title="sarde.yaml"
plugins:
  config:
    social_cards:
      skip_if_image: true
      format: "png"
```

When `skip_if_image` is `true` (the default), a page with an `image` in its frontmatter uses that image and gets no card. `format` also accepts `jpeg`, which adds the `quality` option. See [Social Cards](/plugins/social-cards/) for colors, logo, background, fonts, and per-page overrides.

## Sitemap

The `sitemap` plugin writes `sitemap.xml` at the site root. It lists every published page with a `lastmod` date and leaves out drafts and pagination pages. `changefreq` and `priority` are configurable, and `exclude` takes URL patterns to skip.

See [Sitemap](/plugins/sitemap/) for the options.

## RSS and Atom feeds

Blog collections generate feeds without configuration:

- RSS 2.0 at `/<collection>/feed.xml`
- Atom 1.0 at `/<collection>/atom.xml`

Each feed holds the 20 most recent posts by default. An entry carries the title, link, date, and a summary taken from the page `description`, falling back to the page summary. Feeds do not include the full post body.

Set `feed: false` on a collection to turn its feeds off, or `feed: true` on another collection to add feeds for it. See [Feeds](/plugins/feeds/) for the entry limit and the `collections` option.

## robots.txt

The `robots` plugin writes `robots.txt` with an `Allow: /` rule. It adds a `Sitemap:` line pointing to `sitemap.xml` only when the `sitemap` plugin is enabled and `site.url` is set.

See [Robots](/plugins/robots/) for the options.

## llms.txt

The `llms_txt` plugin writes `/llms.txt`, a Markdown index of the site for AI tools. It lists each content page with its title and URL and leaves out section index pages and the homepage. Configure it under the top-level `llms_txt` key, not under `plugins.config`. See [LLMs.txt](/plugins/llms-txt/) for the options.
