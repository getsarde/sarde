---
title: Build Pipeline
description: "The six-phase engine pipeline: Initialize, Discover, Parse, Assemble, Render, Write."
sidebar:
  order: 2
---

Sarde builds a site through six sequential phases. Some phases run their work items in parallel when the page count warrants it.

## Phase 1: Initialize

- Resolves content and output directories
- Determines parallel/worker settings
- Loads the i18n string table
- On the first build only: registers plugins and runs `ConfigSetup` hooks, loads icon sources

## Phase 2: Discover

- Scans the content directory to enumerate all files
- Classifies each file by node kind (home, section, page, bundle, standalone)
- Detects page bundles (an `index.md` with sibling non-Markdown files)

## Phase 3: Parse

- Builds collections from discovered files (parallel-capable)
- Parses frontmatter (YAML, TOML, JSON)
- Applies the DefaultsInferrer (title from H1/filename, date from git/mtime, weight from numeric prefix)
- Validates frontmatter against collection schemas
- Builds standalone pages

## Phase 4: Assemble

- Collects all pages across collections
- Runs `ContentLoaded` plugin hooks (plugins can inject virtual pages)
- Links i18n translations and generates fallback pages
- Links versioned pages across version lanes
- Builds taxonomy structures (per-language for multi-lang sites)
- Collects sidebar-override warnings

## Phase 5: Assets and rendering

This is the largest phase, running several sub-steps in order:

1. **Syntax checking** (opt-in): scans for unclosed fenced code blocks
2. **Asset pipeline setup**: builds the asset pipeline, enhances page resources (dimensions, media types, permalinks), builds the PageIndex for link validation, loads the 3-layer shortcode registry (embedded, theme, user)
3. **Markdown rendering** (parallel): converts Markdown to HTML using a pooled renderer worker pool. Consults a content-hash-keyed PageCache to skip unchanged pages.
4. **Link validation**: validates internal links, anchors, and optionally external URLs. In check-only mode (`sarde check-links`), the build stops here.
5. **Asset bundling**: bundles CSS/JS through esbuild, processes theme JS, generates favicon and theme token CSS
6. **Template engine wiring**: loads templates and hands the assembled site context to the template engine

## Phase 6: Write

- Writes bundle assets, processed images, bundled CSS/JS, and theme assets (parallel-capable)
- Runs `BuildDone` plugin hooks (parallel, one goroutine per plugin)
- Writes all rendered HTML pages and alias redirects
- Copies `public/` files to the output directory
- Prunes orphaned files from previous builds (skipped in dev mode)
- Snapshots builder state for incremental rebuilds

## Incremental rebuilds

During `sarde dev`, content-only changes trigger an incremental `ContentRebuild` instead of a full build. This reuses the existing builder (template engine, plugin state) and only re-processes changed files.

### Body-only fast path

A two-layer digest gate determines whether a change is body-only:

1. **Content digest**: if the raw file bytes haven't changed, the file is skipped entirely
2. **Frontmatter digest**: if frontmatter and title are unchanged, the change is classified as body-only

When all changes in a batch are body-only, the rebuild takes a cheaper path: reuses taxonomy structures, skips re-walking `public/` for the asset index, and skips version re-linking.

### Fallback to full build

Structural changes (new/deleted files, bundle content changes, draft/publish status flips, tabbed/versioned/multi-lang navigation changes) fall back to a full build automatically.

## Parallelism

Parallel execution is decided per-phase based on the item count and available CPU cores. The parallel stages are: content parsing, Markdown rendering, template rendering, asset writes, and `BuildDone` plugin hooks.
