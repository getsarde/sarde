---
title: KaTeX
description: "Ship KaTeX runtime assets to pages that contain math markup"
sidebar:
  order: 33
  group: Server-Side
---

Ships KaTeX runtime assets (JS, CSS, fonts) and injects them into pages that contain math markup. Enabled by default, but assets are only loaded on pages that use math.

## How it works

The Math extension converts `$...$` (inline) and `$$...$$` (display) syntax into HTML elements with `class="sarde-math"`. During the build, the KaTeX plugin scans each page's rendered HTML for this class. When found, it appends the KaTeX stylesheet and scripts to the page's asset list. Pages without math receive no KaTeX assets.

The runtime assets are copied to `assets/vendor/katex/` in the build output:

- `katex.min.css` (stylesheet)
- `katex.min.js` (core library)
- `auto-render.min.js` (auto-render extension)
- `init.js` (initialization script)
- `fonts/` (20 `.woff2` font files)

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    katex:
      always: false
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `always` | Boolean | `false` | Load KaTeX assets on every page, regardless of whether math is detected. |

Set `always: true` when math is rendered dynamically (e.g., via client-side JavaScript) and the build-time detection cannot find the `sarde-math` class in the static HTML.

## Detection

The plugin checks each page's rendered HTML for the string `class="sarde-math"`. This class is added by Sarde's Math extension when it processes `$` or `$$` delimiters. No detection happens when `always: true` is set.

## Initialization

The init script runs on `DOMContentLoaded` and calls `renderMathInElement` on `document.body` with these delimiters:

| Delimiter | Type |
|-----------|------|
| `$...$` | Inline |
| `$$...$$` | Display |
| `\(...\)` | Inline |
| `\[...\]` | Display |

Errors in individual expressions are silently swallowed (`throwOnError: false`).

## Relationship to the Math extension

The Math extension (documented in the Extensions section) handles the Markdown parsing: converting `$` and `$$` syntax into HTML nodes. The KaTeX plugin handles the runtime: shipping the KaTeX library and rendering those nodes into formatted equations in the browser. Both are needed for math to display.

## Disabling KaTeX

Remove `katex` from the enabled list:

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
    # katex removed
```

Math syntax (`$...$`) is still parsed by the Math extension, but the rendered output is not styled or processed without the KaTeX runtime.

## Edge cases

- On incremental rebuilds, vendor assets are not re-copied (they are identical across rebuilds and already present from the initial build).
- The plugin deduplicates asset URLs: if another plugin or template already added the KaTeX stylesheet, it is not appended a second time.
- The `always` flag applies to all pages, including listing pages and the homepage.

See the [Math extension](/extensions/math) for the `$...$` and `$$...$$` syntax.
