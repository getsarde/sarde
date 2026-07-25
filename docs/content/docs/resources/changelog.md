---
title: Changelog
description: "Notable changes to Sarde, grouped by release, covering fixes, improvements, and new features"
sidebar:
  order: 1
---

Notable changes to Sarde, grouped by release. Bug fixes, new features, and breaking changes are listed separately within each release.

## Unreleased

### Changed

- `build.last_updated` now defaults to `git` instead of `mtime`. Page timestamps come from each file's last commit rather than its filesystem modification time, which in CI is the checkout time and made every page report the same moment on every deploy. Outside a git repository the behavior is unchanged, since `git` falls back to `mtime` automatically. Set `build.last_updated: mtime` to keep the old behavior.
- `show_updated: false` now hides only the "last updated" badge. The timestamp is still resolved, so sitemap `lastmod`, SEO `dateModified`, and feed timestamps for that page remain correct. Previously it suppressed the date entirely, stripping that metadata as a side effect.

### Fixed

- "Edit this page" links now render on docs pages. `site.edit_url` was previously honored only by blog and default layouts, so a docs site could configure it correctly and see links on blog posts alone. Sites that do not set `site.edit_url` are unaffected.
- `show_updated: false` was ignored on pages that set `updated:` explicitly in frontmatter.
- The "last updated" badge never appeared on blog posts.
- The badge's `position: bottom` setting placed it at the end of the page content instead of above the previous/next navigation.
- The badge is now localized: relative times use the page's language via `Intl.RelativeTimeFormat`, and absolute dates and the label follow the page language rather than the visitor's browser locale.
- Sarde now warns once when `build.last_updated: git` cannot be used (git missing, not a repository, or a shallow clone) instead of silently falling back to file modification times.
- The "Made with Sarde" footer credit linked to a domain that does not resolve. It now points to the documentation site. Because the footer template is compiled into the binary, sites built with 1.0.0 or 1.0.1 keep the old link until you upgrade and rebuild.

## 1.0.1 - 2026-07-25

### Fixed

- Added `linux/arm64` binaries, which were missing from the 1.0.0 release. ARM Linux servers and CI runners can now install with the standard script.

## 1.0.0 - 2026-07-24

The first stable release. Everything below has accumulated since the 0.1.x previews.

### Breaking

- The `static/` project directory is renamed to `public/`. Files placed in `public/` are copied as-is to the output directory, exactly as `static/` worked before. Rename your project's `static/` directory to `public/` before your next build.

### Fixed

- Frontmatter date fields (`date`, `updated`, `publish_date`, `expiry_date`) may now be left empty. An empty value means "not set" instead of aborting the build, which is what an editor writes when a date field is cleared.
- Frontmatter parse errors now name the file they came from, instead of reporting a bare parse failure with no location.
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

- External plugin system: install declarative `plugin.yaml` packages into `plugins/<slug>/` with `sarde plugin install`, supporting conditional CSS/JS injection, template contributions (partials, components, shortcodes), and asset copying to the build output. No plugin code executes at build time.
- New `plugins.disabled` config list that turns off any plugin (built-in, client-side, or external) without replacing the `plugins.enabled` list.
- Offline license verification for premium external plugins, managed with `sarde license install` and `sarde license list`. A missing or invalid license deactivates the plugin with a warning; the build never fails.
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
