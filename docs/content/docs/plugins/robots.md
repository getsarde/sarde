---
title: Robots
description: "Generate a robots.txt file that allows all crawlers and links the sitemap"
sidebar:
  order: 38
  group: Server-Side
---

Generates a `robots.txt` file at the site root that allows all crawlers and includes a sitemap directive. Enabled by default.

## Output

The generated file contains:

```text
User-agent: *
Allow: /
Sitemap: https://example.com/sitemap.xml
```

The `Sitemap` URL is constructed from `site.url` in `sarde.yaml`. If no site URL is configured, the sitemap line is omitted.

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    robots:
      sitemap: true
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `sitemap` | Boolean | `true` | Include the `Sitemap:` directive pointing to `sitemap.xml`. |

## Disabling the sitemap directive

Set `sitemap: false` to generate a `robots.txt` without the `Sitemap:` line. This is useful when the sitemap plugin is disabled or when using an external sitemap.

```yaml
plugins:
  config:
    robots:
      sitemap: false
```

→ The output contains only `User-agent: *` and `Allow: /`.

## Disabling the plugin

Remove `robots` from the enabled list:

```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    # robots removed
    - rss
    - atom
```

No `robots.txt` file is generated. Crawlers typically treat a missing `robots.txt` as "allow all."

## Edge cases

- The `Sitemap:` directive uses the absolute URL from `site.url`. If `site.url` is empty, no sitemap line is written.
- The file is always overwritten on each build. Custom `robots.txt` files in the `public/` directory take precedence over the generated one (public files are copied after plugin output).
