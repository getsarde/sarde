---
title: Changelog
description: "Notable changes to Sarde, grouped by release, covering fixes, improvements, and new features"
sidebar:
  order: 1
---

Notable changes to Sarde, grouped by release. Bug fixes, new features, and breaking changes are listed separately within each release.

## Unreleased

### Breaking

- The `static/` project directory is renamed to `public/`. Files placed in `public/` are copied as-is to the output directory, exactly as `static/` worked before. Rename your project's `static/` directory to `public/` before your next build.

### Fixed

- Page cache now re-validates link targets on every build, even for cached pages. Warm builds report the same link coverage as cold builds, and renaming a linked page is detected without editing the source file.
- Custom heading IDs (`## Heading {#custom-id}`) are now preserved. Previously, all heading IDs were overwritten with auto-generated slugs, causing links to custom anchors to be falsely reported as broken.
- The `site.heading_links` config option is now functional. It controls whether clickable anchor links appear next to headings. Heading IDs are always assigned (required by the TOC, search, and link validation), but the visible anchor element is now toggled by this setting.
- Content lint rules no longer trigger false positives inside fenced code blocks or inline code spans. Example syntax shown in documentation (e.g., `![](...)` in a code block) is skipped.
- Content lint line numbers now match the source file on disk. Previously, line numbers were offset by the frontmatter block's height.
- The `same_site_policy` link validation option works correctly on incremental rebuilds.

### Improved

- Multi-language sites reuse taxonomy structures on body-only incremental rebuilds, skipping redundant taxonomy and data file processing per language.
- WebSocket hub uses ping/pong keepalive (30-second interval) to detect stale connections.
- Unchanged headings are reused during incremental rebuilds, and link validation is scoped to changed pages only.

### Added

- Announcement banners plugin with three display modes (stack, first, rotate), date scheduling, page targeting via glob patterns, and i18n message resolution.
- Social card auto-generation (1200x630 PNG/JPEG) with theme-aware colors, Inter font embedding, and title auto-sizing.
- `llms.txt` generation for LLM discoverability, with optional blog content exclusion.
- RTL CSS layout support.
- Language switcher and fallback notice components for multi-language sites.
- Tab state persistence across page navigation via `localStorage`.
- Collapsible sidebar groups with `collapsed_by_default` configuration.
- Mobile sidebar drawer.
- 16 client-side plugins: scroll to top, copy section link, external links, image lightbox, focus mode, reading progress, reading preferences, reading position memory, search highlighter, text highlighter, and last updated badge.
- 24 Markdown extensions: aside, accordion, badges, cards, details, figure, file tree, gallery, image compare, link buttons, link card, math, mermaid, steps, tabs, terminal, timeline, video, annotation, copy text, highlight, icon, kbd, and spoiler.
