---
title: Redirects
description: "Generate redirect files from page aliases and global redirect mappings"
sidebar:
  order: 37
  group: Server-Side
---

Generates redirect files from page aliases and global redirect mappings. Outputs HTML meta-refresh pages as a universal fallback, plus platform-specific files for Netlify/Cloudflare (`_redirects`) and Vercel (`vercel.json`). Enabled by default.

## Redirect sources

Redirects come from two places, merged at build time:

1. **Global redirects** in `sarde.yaml` (the `redirects` map)
2. **Page aliases** in frontmatter (the `aliases` field)

Page aliases override global redirects when both define the same source path.

## Global redirects

Define source-to-target path mappings in `sarde.yaml`:

`sarde.yaml`
```yaml
redirects:
  /old-page/: /new-page/
  /legacy/guide/: /docs/guides/writing-content/
  /blog/2023/announcement/: /blog/2024/update/
```

Each key is the old URL path. Each value is the target URL path.

## Page aliases

Add an `aliases` list to any page's frontmatter. Each alias creates a redirect from that path to the page's current URL.

```yaml
---
title: Writing Content
aliases:
  - /docs/authoring/
  - /docs/writing-markdown/
---
```

→ Visiting `/docs/authoring/` or `/docs/writing-markdown/` redirects to the current page URL.

## Output formats

The plugin generates up to three output formats per redirect.

### HTML meta-refresh (always generated)

For each redirect, an HTML file is written at the source path. The file uses `<meta http-equiv="refresh">` and a `<link rel="canonical">` pointing to the target. This works on any static hosting provider.

For a redirect from `/old-page/`, the file is written to `old-page/index.html`. Paths ending in `.html` are written as-is.

### Netlify/Cloudflare `_redirects`

A `_redirects` file is generated at the site root with one line per redirect:

```text
/old-page/  /new-page/  301
/legacy/guide/  /docs/guides/writing-content/  301
```

All redirects use status code 301 (permanent).

### Vercel `vercel.json`

A `vercel.json` file is generated at the site root:

```json
{
  "redirects": [
    {
      "source": "/old-page/",
      "destination": "/new-page/",
      "permanent": true
    }
  ]
}
```

## Controlling output formats

Set `deploy.redirect_format` in `sarde.yaml` to control which platform files are generated. HTML meta-refresh files are always written regardless of this setting.

`sarde.yaml`
```yaml
deploy:
  redirect_format: netlify
```

| Value | `_redirects` | `vercel.json` |
|-------|:---:|:---:|
| `""` (default) | Yes | Yes |
| `"all"` | Yes | Yes |
| `"netlify"` | Yes | No |
| `"vercel"` | No | Yes |

## Disabling redirects

Remove `redirects` from the enabled list:

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
    # redirects removed
```

Page `aliases` and the `redirects` map in `sarde.yaml` are still parsed but no redirect files are generated.

## Edge cases

- When a page alias conflicts with a global redirect for the same source path, the page alias wins.
- Source paths without a trailing slash or `.html` extension get `index.html` appended (e.g., `/old-page` becomes `old-page/index.html`).
- Output is deterministic: redirects are sorted alphabetically by source path.
- Target URLs in HTML output are sanitized: `"`, `<`, and `>` characters are percent-encoded.
- If no redirects exist (no aliases and no global mappings), no files are written.
