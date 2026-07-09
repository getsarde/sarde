---
title: Mermaid
sidebar:
  order: 36
  group: Server-Side
---

Ships Mermaid runtime assets and injects them into pages that contain diagram markup. Enabled by default, but assets are only loaded on pages that use Mermaid diagrams.

## How it works

The Mermaid extension converts fenced code blocks with the `sarde-mermaid` language into `<div class="sarde-mermaid">` elements during the build. The Mermaid plugin scans each page's rendered HTML for this class. When found, it appends the Mermaid scripts to the page's asset list. Pages without diagrams receive no Mermaid assets.

The runtime assets are copied to `assets/vendor/mermaid/` in the build output:

- `mermaid.min.js` (core library)
- `init.js` (initialization script)

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    mermaid:
      always: false
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `always` | Boolean | `false` | Load Mermaid assets on every page, regardless of whether a diagram is detected. |

Set `always: true` when diagrams are injected dynamically via client-side JavaScript and the build-time detection cannot find the `sarde-mermaid` class in the static HTML.

## Detection

The plugin checks each page's rendered HTML for the string `class="sarde-mermaid"`. This class is added by Sarde's Mermaid extension when it transforms `sarde-mermaid` fenced code blocks. No detection runs when `always: true` is set.

## Initialization

The init script runs on `DOMContentLoaded` and configures Mermaid with:

- `startOnLoad: false` (the script triggers rendering manually)
- `theme: "dark"` or `"default"` based on the current `data-theme` attribute
- `securityLevel: "strict"`

The script then calls `mermaid.run()` targeting all `.sarde-mermaid` elements.

## Theme awareness

The init script reads `document.documentElement.getAttribute("data-theme")` at render time. When the attribute is `"dark"`, Mermaid uses its dark theme. Otherwise it uses the default (light) theme. Theme changes after initial render require a page reload to re-initialize Mermaid.

## Relationship to the Mermaid extension

The Mermaid extension (documented in the Extensions section) handles the Markdown parsing: converting `sarde-mermaid` fenced code blocks into placeholder `<div>` elements. The Mermaid plugin handles the runtime: shipping the Mermaid library and rendering those placeholders into SVG diagrams in the browser. Both are needed for diagrams to display.

## Disabling Mermaid

Remove `mermaid` from the enabled list:

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
    # mermaid removed
```

Fenced code blocks with the `sarde-mermaid` language are still transformed by the extension, but the rendered output shows the raw diagram source without Mermaid rendering.

## Edge cases

- On incremental rebuilds, vendor assets are not re-copied (they are identical across rebuilds and already present from the initial build).
- The plugin deduplicates script URLs: if another plugin or template already added the Mermaid script, it is not appended a second time.
- Unlike the KaTeX plugin, Mermaid does not inject a stylesheet. All styling comes from the Mermaid library's inline SVG output.
- The `always` flag applies to all pages, including listing pages and the homepage.
