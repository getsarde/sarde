---
title: External Links
sidebar:
  order: 12
  group: Client-Side
---

External links in the content area receive a small arrow icon and open in a new browser tab. Disabled by default.

## Enable the plugin

`sarde.yaml`
```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    # ... other default plugins
    - external-links
```

## How it works

On page load, the plugin scans all links inside the main content area. Links pointing to a different hostname than the current site are treated as external and receive:

- An arrow icon appended after the link text (inline SVG, sized to match the surrounding text).
- `target="_blank"` to open in a new tab.
- `rel="noopener noreferrer"` for security (prevents reverse tabnabbing and referrer leakage).

Links in the header, sidebar, footer, and navigation are not processed. Only links inside the main content article are affected.

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    external-links:
      show_icon: true
      open_in_new_tab: true
      exclude_domains: "github.com, docs.example.com"
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `show_icon` | Boolean | `true` | Display the arrow icon after external links. |
| `open_in_new_tab` | Boolean | `true` | Open external links in a new browser tab. |
| `exclude_domains` | String | `""` | Comma-separated hostnames to treat as internal. |

## Excluding domains

Use `exclude_domains` to prevent specific external hostnames from receiving the icon and new-tab behavior:

`sarde.yaml`
```yaml
plugins:
  config:
    external-links:
      exclude_domains: "github.com, docs.example.com"
```

Domain matching is **exact by hostname**. Excluding `example.com` does not exclude `www.example.com` or `docs.example.com`. List each subdomain separately if needed.

## Injection rule

This plugin activates on every page (`always`). It runs once at page load and does not re-process dynamically inserted links.

## Edge cases

- Anchor links (`#section`), `mailto:`, `tel:`, and `javascript:` URLs are never treated as external.
- Links without an `href` attribute are ignored entirely.
- When `open_in_new_tab` is `false`, neither `target` nor `rel` attributes are modified.
- The icon is hidden on image links, link buttons, link cards, and navigation links via CSS, even if the plugin's JavaScript were to process them.
- The icon inherits the link's text color via `currentColor` and dims to 50% opacity, brightening to 80% on hover.
- Setting `show_icon: false` and `open_in_new_tab: false` effectively disables the plugin without removing it from `plugins.enabled`.
