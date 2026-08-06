---
title: Dev Server
description: "Dev server internals, WebSocket live reload, CSS hot-swap, and draft content handling."
sidebar:
  order: 5
---

`sarde dev` starts a local development server with file watching, live reload, and incremental rebuilds. It serves the built output directory with clean URL support and injects a WebSocket client for instant browser updates.

## Live reload

The server maintains WebSocket connections to all open browser tabs via a connection hub. When a build completes, the hub broadcasts a reload message and every connected tab refreshes.

Each HTML response is intercepted before serving: the server injects a `<script>` tag before `</body>` containing the live-reload client and a build ID (a monotonic timestamp), exposed to the page as `window.__SARDE_LR_BUILD__`. When the client receives a reload message with a newer build ID, it triggers a page reload. CSS-only changes use a lighter mechanism that swaps stylesheets without a full page reload.

Messages arrive over the WebSocket with a `type` field:

| Type | Meaning |
|------|---------|
| `reload` | Full page reload |
| `css` | Swap stylesheets in place |
| `error` | Build failed; show the error overlay |
| `warning` | Build succeeded with warnings |
| `sync` | Sent to a client on connect, carrying the latest successful build ID |

`sync` is what keeps a tab from going stale. On connect the client compares the announced build ID against the one embedded in its own page, and reloads only if the page predates the build. A tab that was open across a server restart, a dropped connection, or a missed broadcast catches up exactly once, and a freshly loaded tab never reloads spuriously.

A ping/pong keepalive runs every 30 seconds to detect stale connections.

## File watching

The server watches these project directories for changes: `content/`, `layouts/`, `assets/`, `data/`, `public/`, `themes/`, `plugins/`. It also watches the project root non-recursively for config file changes (`sarde.yaml`, `theme.yaml`, `nav.yaml`, `sidebar.yaml`).

Changes are debounced (150ms) and batched before processing.

## Change classification

Each changed file is classified into one of six kinds:

| Kind | Trigger | Behavior |
|------|---------|----------|
| Config | `sarde.yaml`, `theme.yaml`, `nav.yaml`, `sidebar.yaml` | Fresh builder, full build |
| Template | Files under `layouts/` or `themes/` (except theme CSS) | Fresh builder, full build |
| Plugin | Files under `plugins/` | Fresh builder, full build |
| Content | Files under `content/` | Incremental rebuild, reuses existing builder |
| CSS | `.css` files under `public/` | Hot-swap, no rebuild |
| Theme CSS | `.css` files under `themes/<name>/css/` (or the theme dev source tree) | Bundle reassembled in place, no rebuild |
| Static | Everything else (`assets/`, `data/`, other files) | Full build, reuses existing builder |

## CSS hot-swap

When all changes in a batch are CSS files under `public/`, no rebuild runs. Each changed file is copied directly to the output directory, and the browser receives a CSS-specific reload message that swaps the stylesheet in place without a full page reload.

CSS files under `assets/` are excluded from hot-swap because they may be bundled or fingerprinted through esbuild. Changes to those files trigger a full rebuild instead.

## Theme CSS fast path

Theme stylesheets (`themes/<name>/css/`, or the theme source tree when the server runs with `--theme-dev`) feed the single `sarde.css` bundle, so a CSS edit cannot change page HTML, links, or search content. When all changes in a batch are theme CSS files, the server reassembles the bundle from disk, rewrites it in the output directory, and broadcasts the same CSS swap message as the `public/` hot-swap path. No pages are rebuilt, and the link checker and search index do not rerun.

If the refresh fails (for example, no build has completed yet), the server falls back to a full rebuild on a fresh builder. Non-CSS theme files (templates, JS) keep the template behavior above.

## Batch merging

When a batch contains mixed change kinds, the highest-priority kind wins: config (5) > template and plugin (4) > content (3) > static (2) > CSS and theme CSS (1).

Two exceptions:

- If a batch mixes content changes with any non-content kind, the result is escalated to a full build (static kind). The incremental content path would miss the accompanying static/CSS files.
- If a batch mixes theme CSS with content or static changes, it escalates to a template change (fresh builder). A full build on a reused builder would keep the stale CSS bundle, since the template engine assembles CSS only once per builder.

Content file paths across the batch are deduped and unioned. The earliest detection timestamp is preserved for end-to-end timing.

## Rebuild coalescing

At most one rebuild runs at a time. If a file change arrives during an active rebuild, it is merged into a pending slot (kinds escalate, content paths union). When the active rebuild finishes, it checks the pending slot and runs the merged change automatically. This repeats until no more changes accumulated, guaranteeing no lost changes while preventing cascading queued rebuilds from rapid saves.

## Clean URL support

The file server resolves URLs without extensions to their `index.html` equivalents: `/docs/guide/` serves `docs/guide/index.html`, and `/docs/guide` (no trailing slash) does the same. Directory listings are suppressed. All responses use `Cache-Control: no-store` to prevent stale cached output during development.

If no matching file exists, the server tries a language-scoped `404.html` (derived from the first URL path segment), then falls back to the root `404.html`. If neither exists, a minimal HTML skeleton is served so the live-reload client can still attach and display the error overlay.

## Port binding

If the requested port is taken, the server tries up to 10 consecutive ports before giving up, and reports the one it bound to. Code driving the server through the project API gets the same value back from the preview call, so a caller that passed port 0 or a busy port learns the real port rather than assuming its request was honored.

## Timing a rebuild

Run the server with `--verbose` to print per-phase timings on every rebuild. Use it to find which phase a slow save is spending its time in before reaching for the cache settings.

```bash
sarde dev --verbose
```
