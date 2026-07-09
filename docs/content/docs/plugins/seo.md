---
title: SEO
sidebar:
  order: 40
  group: Server-Side
---

Injects Open Graph, Twitter Card, and JSON-LD structured data into every page during the build. Populates page-level `seo` params that the Head component reads to render `<meta>` tags. Enabled by default.

## What the plugin generates

For each page, the plugin sets the following in `page.Params["seo"]`:

### Open Graph tags

| Key | Source |
|-----|--------|
| `og_title` | Page title |
| `og_description` | Page description, falling back to summary if `auto_description` is enabled |
| `og_url` | Canonical URL (version-aware) |
| `og_type` | `"article"` for collection pages, `"website"` otherwise |
| `og_site_name` | Site title from `sarde.yaml` |
| `og_locale` | Page language normalized to OGP format (e.g., `en_US`, `pt_BR`) |
| `og_locale_alternate` | Locales of available translations |
| `og_image` | Page image, falling back to `default_image`, resolved to absolute URL |
| `og_image_alt` | Page title |
| `article_published_time` | Page date in RFC 3339 format (articles only) |
| `article_modified_time` | Page updated date in RFC 3339 format (articles only) |

### Twitter Card tags

| Key | Source |
|-----|--------|
| `twitter_card` | Card type (default: `summary_large_image`) |
| `twitter_title` | Page title |
| `twitter_description` | Same as `og_description` |
| `twitter_image` | Same as `og_image` |
| `twitter_image_alt` | Page title |
| `twitter_site` | `twitter_handle` from plugin config, if set |

### JSON-LD structured data

The plugin generates a `@graph` array with up to three schema.org nodes:

| Node type | When emitted |
|-----------|-------------|
| `Article` | Collection pages (blog posts, docs pages) |
| `CollectionPage` | Section index pages and the homepage |
| `WebPage` | All other pages |
| `BreadcrumbList` | When route data has two or more breadcrumb entries |
| `Course` | When the page sets `params.schema_type: "Course"` |

The Article node includes `headline`, `datePublished`, `dateModified`, `author` (from page params), and `image`. The Course node includes `provider` (from page params).

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    seo:
      twitter_handle: "@mysite"
      twitter_card: summary_large_image
      default_image: /images/og-default.png
      json_ld: true
      auto_description: true
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `twitter_handle` | String | `""` | Twitter `@handle` for the `twitter:site` tag. |
| `twitter_card` | String | `"summary_large_image"` | Twitter card type. Common values: `summary`, `summary_large_image`. |
| `default_image` | String | `""` | Fallback image URL when a page has no `image` field. |
| `json_ld` | Boolean | `true` | Generate JSON-LD structured data. |
| `auto_description` | Boolean | `true` | Use page summary as description when no explicit description is set. |

## Version-aware canonical URLs

For versioned collections, non-latest versioned pages set their canonical URL to the latest version's URL. This consolidates search engine signals on the current documentation and avoids duplicate content penalties for older versions.

A page at `/docs/v1/getting-started/` with the latest version being `v2` sets its canonical to `/docs/v2/getting-started/` (if the peer page exists). Pages in the latest version, and unversioned pages, use their own URL as canonical.

## Social Cards plugin interaction

The Social Cards plugin may run before or after the SEO plugin in the same `BeforeRender` hook. The SEO plugin merges into the existing `seo` params map rather than replacing it, so a generated social card image set by the Social Cards plugin is preserved. The SEO plugin only fills `og_image` and `twitter_image` if no value was already set.

## Locale normalization

Language tags are converted to OGP locale format:

| Input | Output |
|-------|--------|
| `en` | `en_US` |
| `fr` | `fr_FR` |
| `pt-BR` | `pt_BR` |
| (empty) | `en_US` |

Bare language codes without a region repeat the language as the region (uppercased).

## Disabling the plugin

Remove `seo` from the enabled list:

```yaml
plugins:
  enabled:
    - search
    # seo removed
    - sitemap
    - robots
```

No meta tags, Open Graph tags, Twitter Card tags, or JSON-LD are generated. The Head component renders whatever `seo` params exist (which would be none).

## Edge cases

- Pages without a collection are typed as `"website"` for Open Graph (not `"article"`).
- If `default_image` is a relative path, it is resolved to an absolute URL using `site.url`.
- The `auto_description` fallback uses the page's auto-generated summary, which may truncate mid-sentence.
- If a versioned page has no peer in the latest version, the canonical URL remains the page's own URL.
- Breadcrumb JSON-LD is only emitted when the route has at least two breadcrumb entries (a single entry is not useful as a list).
