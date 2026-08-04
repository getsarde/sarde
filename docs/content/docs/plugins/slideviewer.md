---
title: SlideViewer
description: "Ship the SlideViewer presentation runtime to pages using the presentation layout"
sidebar:
  order: 44
---

Ships the SlideViewer runtime (JS and CSS) and injects it on pages that use the `presentation` layout. SlideViewer is distributed as an [external plugin](/plugins/external-plugins): install it into your project's `plugins/` directory and it activates automatically. Assets are only loaded on presentation pages.

## Installation

```bash
sarde plugin install slideviewer.zip
```

This places the plugin at `{project}/plugins/slideviewer/`:

```
plugins/slideviewer/
  plugin.yaml
  assets/
    js/SlideViewer.min.js          (bundled ES module)
    css/slideviewer-index.min.css  (bundled stylesheet)
```

External plugins are enabled by presence on disk; no `plugins.enabled` entry is needed.

## How it works

During `BeforeRender`, the plugin manifest's inject rule (`when: layout`, `layout: presentation`) checks each page's resolved layout. When it matches, the SlideViewer module and stylesheet are appended to the page's asset list. Pages with any other layout receive no SlideViewer assets.

During `BuildDone`, the runtime is copied to `assets/vendor/slideviewer/` in the build output.

The viewer's UI markup is provided by the embedded theme partial `slide-viewer-shell.html`. The JS binds to pre-existing element IDs in that shell; it does not build its own DOM.

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    slideviewer:
      always: false
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `always` | Boolean | `false` | Load SlideViewer assets on every page, regardless of layout. |

Set `always: true` when the presentation layout is applied dynamically or when another mechanism triggers the viewer outside the standard layout gate.

## Detection

The plugin gates on the page's resolved `layout`, not on content scanning. A page reaches `layout: presentation` through any of:

- The `slides` collection type (deck pages default to `presentation` automatically)
- Explicit `layout: presentation` in frontmatter
- Section cascade (`cascade: { layout: presentation }` in an `_index.md`)

See the [Teaching](/teaching/) section for the full guide.

## Relationship to the slides collection

The plugin handles asset delivery only. The `slides` collection type and `layout: presentation` are engine features that exist independently of this plugin. Removing the plugin removes the viewer runtime from the output; presentation pages then render as readable pages with no slide navigation.

## Disabling the plugin

Either remove the `plugins/slideviewer/` directory, or disable it in place:

```yaml
plugins:
  disabled:
    - slideviewer
```

Presentation pages still render with the presentation layout (no site chrome), but the viewer JS and CSS are not loaded.

## Edge cases

- On incremental rebuilds, vendor assets are not re-copied (they are identical across rebuilds and already present from the initial build).
- The plugin deduplicates asset URLs: if another plugin or template already added the SlideViewer stylesheet, it is not appended a second time.
- The `always` flag applies to all pages, including listing pages and the homepage.
- Assets are loaded as ES modules (`<script type="module">`), not classic scripts. Browsers that do not support ES modules will not run the viewer.
