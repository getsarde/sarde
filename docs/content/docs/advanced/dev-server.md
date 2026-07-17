---
title: Dev Server
description: "Dev server internals, WebSocket live reload, CSS hot-swap, and draft content handling."
sidebar:
  order: 5
---

`sarde dev` starts a local development server with file watching, live reload, and incremental rebuilds. It serves the built output directory with clean URL support and injects a WebSocket client for instant browser updates.

## Live reload

The server maintains WebSocket connections to all open browser tabs via a connection hub. When a build completes, the hub broadcasts a reload message and every connected tab refreshes.

Each HTML response is intercepted before serving: the server injects a `<script>` tag before `</body>` containing the live-reload client and a build ID (a monotonic timestamp). When the client receives a reload message with a newer build ID, it triggers a page reload. CSS-only changes use a lighter mechanism that swaps stylesheets without a full page reload.

A ping/pong keepalive runs every 30 seconds to detect stale connections.

## File watching

The server watches these project directories for changes: `content/`, `layouts/`, `assets/`, `data/`, `public/`, `themes/`, `plugins/`. It also watches the project root non-recursively for config file changes (`sarde.yaml`, `theme.yaml`, `nav.yaml`, `sidebar.yaml`).

Changes are debounced (150ms) and batched before processing.

## Change classification

Each changed file is classified into one of six kinds:

| Kind | Trigger | Behavior |
|------|---------|----------|
| Config | `sarde.yaml`, `theme.yaml`, `nav.yaml`, `sidebar.yaml` | Fresh builder, full build |
| Template | Files under `layouts/` or `themes/` | Fresh builder, full build |
| Plugin | Files under `plugins/` | Fresh builder, full build |
| Content | Files under `content/` | Incremental rebuild, reuses existing builder |
| CSS | `.css` files under `public/` | Hot-swap, no rebuild |
| Static | Everything else (`assets/`, `data/`, other files) | Full build, reuses existing builder |

## CSS hot-swap

When all changes in a batch are CSS files under `public/`, no rebuild runs. Each changed file is copied directly to the output directory, and the browser receives a CSS-specific reload message that swaps the stylesheet in place without a full page reload.

CSS files under `assets/` are excluded from hot-swap because they may be bundled or fingerprinted through esbuild. Changes to those files trigger a full rebuild instead.

## Batch merging

When a batch contains mixed change kinds, the highest-priority kind wins: config (5) > template and plugin (4) > content (3) > static (2) > CSS (1).

One exception: if a batch mixes content changes with any non-content kind, the result is escalated to a full build (static kind). The incremental content path would miss the accompanying static/CSS files.

Content file paths across the batch are deduped and unioned. The earliest detection timestamp is preserved for end-to-end timing.

## Rebuild coalescing

At most one rebuild runs at a time. If a file change arrives during an active rebuild, it is merged into a pending slot (kinds escalate, content paths union). When the active rebuild finishes, it checks the pending slot and runs the merged change automatically. This repeats until no more changes accumulated, guaranteeing no lost changes while preventing cascading queued rebuilds from rapid saves.

## Clean URL support

The file server resolves URLs without extensions to their `index.html` equivalents: `/docs/guide/` serves `docs/guide/index.html`, and `/docs/guide` (no trailing slash) does the same. Directory listings are suppressed. All responses use `Cache-Control: no-store` to prevent stale cached output during development.

If no matching file exists, the server tries a language-scoped `404.html` (derived from the first URL path segment), then falls back to the root `404.html`. If neither exists, a minimal HTML skeleton is served so the live-reload client can still attach and display the error overlay.
