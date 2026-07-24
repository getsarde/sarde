---
title: Sitemap
description: "Generate a sitemap.xml listing every non-draft page with metadata"
sidebar:
  order: 41
---

Generates a `sitemap.xml` file at the site root listing every non-draft page with its URL, last modification date, change frequency, and priority. Enabled by default.

## Output

The generated file follows the Sitemaps 0.9 protocol:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/docs/getting-started/</loc>
    <lastmod>2025-03-15T10:00:00Z</lastmod>
    <changefreq>weekly</changefreq>
    <priority>0.5</priority>
  </url>
</urlset>
```

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    sitemap:
      changefreq: weekly
      priority: "0.5"
      exclude:
        - /admin/*
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `changefreq` | String | `"weekly"` | Change frequency hint for search engines. Common values: `always`, `hourly`, `daily`, `weekly`, `monthly`, `yearly`, `never`. |
| `priority` | String | `"0.5"` | Relative priority of pages within the site. Range: `"0.0"` to `"1.0"`. |
| `exclude` | List | `[]` | URL glob patterns for pages to exclude from the sitemap. |

## Last modification date

Each entry's `<lastmod>` uses the page's `updated` date if set, falling back to the `date` field. Both are formatted in RFC 3339. If neither date is available, the `<lastmod>` element is omitted for that entry.

## Excluded pages

The following pages are automatically excluded:

- Draft pages
- Paginated list pages (URLs containing `/page/`)
- Pages matching `exclude` glob patterns

## Disabling the plugin

Remove `sitemap` from the enabled list:

```yaml
plugins:
  enabled:
    - search
    - seo
    # sitemap removed
    - robots
```

No `sitemap.xml` is generated. The Robots plugin's `Sitemap:` directive still references `sitemap.xml` unless the robots plugin is also reconfigured with `sitemap: false`.

## Edge cases

- The `changefreq` and `priority` values apply uniformly to all pages. Per-page overrides are not supported.
- URLs are constructed as absolute URLs using `site.url` from `sarde.yaml`.
- The output is deterministic: pages appear in build order (the order the engine processes them).
- The XML file includes the standard `<?xml version="1.0" encoding="UTF-8"?>` header.
