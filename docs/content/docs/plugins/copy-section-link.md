---
title: Copy Section Link
description: "Copy a shareable link to a heading section by clicking its anchor icon"
sidebar:
  order: 11
---

Clicking a heading's anchor icon copies a shareable link to that section. A tooltip confirms the copy. Disabled by default.

## Enable the plugin

`sarde.yaml`
```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    # ... other default plugins
    - copy_section_link
```

## How it works

Headings with IDs (H2, H3, H4) display an anchor icon on hover. Without this plugin, clicking the icon navigates to the `#hash`. With the plugin enabled, clicking copies the full URL (including the `#hash`) to the clipboard instead.

The copied URL is an absolute path: `https://yoursite.com/docs/page/#section-heading`. Query parameters on the current page are not included in the copied link.

The plugin uses the Clipboard API (`navigator.clipboard.writeText`). After a successful copy, a tooltip showing "Link copied!" appears above the anchor for 1.5 seconds, then fades out.

<!-- SCREENSHOT: copy-section-link-tooltip — the "Link copied!" tooltip above a heading anchor -->

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    copy_section_link:
      tooltip_text: "Copied!"
      tooltip_duration: 1000
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `tooltip_text` | String | `"Link copied!"` | Message shown after copying. |
| `tooltip_duration` | Number | `1500` | How long the tooltip stays visible, in milliseconds. Range: 500 to 5000. |

## Injection rule

This plugin activates on pages with at least one heading (`has_headings`). Pages without headings receive no plugin config or behavior.

## Edge cases

- The plugin requires `heading_links: true` in the site config (the default). Without heading links, there are no anchor elements to click.
- The Clipboard API requires a secure context (HTTPS or localhost). On insecure pages, the click is intercepted but nothing is copied.
- Rapid clicks on the same anchor reset the tooltip timer rather than stacking multiple tooltips.
- The tooltip respects `prefers-reduced-motion` by disabling the fade animation.
- H1 headings are not affected. Only H2, H3, and H4 headings receive anchor links.
- The anchor icon is part of the base theme, not this plugin. This plugin only adds the copy-to-clipboard behavior on top of the existing anchor.
