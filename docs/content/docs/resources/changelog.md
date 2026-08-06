---
title: Changelog
description: "Notable changes to Sarde, grouped by release, covering fixes, improvements, and new features"
sidebar:
  order: 1
---

Notable changes to Sarde, grouped by release. Bug fixes, new features, and breaking changes are listed separately within each release.

## Unreleased

## 1.3.0 - 2026-08-06

### Added

- **Theme CSS hot-swap in dev mode.** Editing a theme stylesheet (under `themes/<name>/css/` or the `--theme-dev` source tree) now reassembles the CSS bundle in place instead of running a full site rebuild. Typical refresh time is 2-7ms instead of ~1.3s. The browser restyles via the existing CSS swap mechanism without a page reload.
- **Draft banner.** Pages with `draft: true` now display a visual banner in dev mode so draft status is immediately visible in the browser.
- **Prose link underline in extensions.** Links inside tabs, details/accordion, steps, and columns now receive the same underline and hover styling as regular prose links. Previously the `not-content` wrapper on these extensions excluded them.
- **Aside link underline.** Links inside galaxy-style asides now show a visible underline at rest and shift to high-contrast text on hover, matching Starlight's aside link treatment.
- **Hero CTA hover effects.** The primary call-to-action button gains an accent glow, a brightness shift that works in both light and dark themes, and a press state on click.

### Changed

- **Kazari updated to v1.2.0.**
- Active TOC link border thickened from 1px to 2px for better visibility.
- Active TOC link text in dark mode is now brighter so it stands out from the muted neighbors.

### Fixed

- **`color-scheme: light` override** added so `light-dark()` CSS functions resolve correctly when the OS prefers dark mode.
- **Hardcoded dark-mode OKLCH values** in banners and TOC links replaced with theme tokens so custom accent palettes apply everywhere.
- **Telescope collection badge** moved after the description for better scannability.

### Docs

- Added a dedicated [Updating Sarde](/docs/start-here/updating-sarde/) page covering `sarde update`, signed releases, package manager detection, and passive update notices.
- Rewrote the Getting Started opening, dev server, build, and scaffold sections with warmer lead-ins for the educator audience.
- Documented the theme CSS fast path in the [dev server](/docs/advanced/dev-server/) reference page.

## 1.2.0 - 2026-08-04

### Breaking

- **`scroll_to_top` threshold is now pixels instead of a percentage.** The default changed from `30` (meaning 30% of total scroll) to `300` (meaning 300 pixels from the top). Sites with a custom `threshold` value will see different behavior. A new `position` option (`left`, `center`, `right`) controls button placement, defaulting to center.
- **Plugin slugs and config keys standardized to snake_case.** All 11 client plugin slugs changed from kebab-case to snake_case (e.g. `scroll-to-top` to `scroll_to_top`). Legacy kebab-case spellings in `plugins.enabled`, `plugins.disabled`, and `plugins.config` are accepted as deprecated aliases with a build-time warning. The `scroll_to_top` plugin's config field keys also moved from camelCase to snake_case (e.g. `showTooltip` to `show_tooltip`); old spellings are similarly aliased. Update your `sarde.yaml` to the new names to silence the warnings.

### Added

- **Telescope command-palette plugin** for fuzzy page navigation. Opens with a keyboard shortcut and lets readers search and jump to any page instantly. Enable it with `telescope` in `plugins.enabled`.
- **Custom `:::` directives.** Sites and themes can define new Markdown `:::` block directives with a YAML schema, an HTML template, and an optional CSS sidecar placed in a `directives/` directory. `sarde new directive <name>` scaffolds the starter files. `sarde directives --check` validates definitions. The dev server live-reloads directive changes.
- **Plugin-shipped directives.** External plugins can now include `:::` directives by placing definitions in `plugins/<slug>/directives/`.
- **Social card redesign.** Cards now use a bottom-anchored editorial layout with a background gradient, corner logo mark, and optional watermark. New config options: `logo`, `watermark`, `watermark_opacity`, `accent_color_2`, `bg_image`, `bg_gradient`, `logo_size`, and custom `fonts`. Per-page `og_card` frontmatter overrides colors and toggles. A disk cache under `.cache/social_cards/` makes repeat builds near-instant.
- **Site logo in the header.** `site.logo` now renders an image before the site title, with separate light and dark variants. `replaces_title: true` hides the text and shows only the logo. Raster logos get `width`/`height` attributes to prevent layout shift.
- **Signed releases.** Release archives are now signed with ed25519. `sarde update` verifies the signature chain before applying an update.
- **Passive update notices.** `sarde build` and `sarde dev` print a one-line notice when a newer release is available. Suppressed in CI, with `--quiet`, and when stderr is not a terminal. Set `SARDE_NO_UPDATE_CHECK=1` to disable.
- **`sarde update` improvements.** The update command now detects package-manager installs (Homebrew, Scoop, Chocolatey, winget) and prints the matching upgrade command instead of self-replacing. Release notes are shown before the confirmation prompt. A `--yes` flag supports non-interactive use. Permission errors print actionable advice.
- **Benchmark harness.** New `sarde-bench` tool measures build performance with median wall time, pages/sec, and per-phase breakdowns. `sarde build --format json` emits machine-readable build results.
- **Hero CTA icons.** Hero call-to-action buttons accept an optional `icon` field that renders a Lucide icon after the label.
- **Keyboard navigation enhancements.** Side navigation arrows are now mdBook-sized with reserved page margin so they never overlap content. A styled tooltip shows the target page title and keyboard shortcut on hover and focus. Below 1280px, the arrows restyle as compact floating buttons. Buttons auto-hide after inactivity and reappear on scroll. New options: `side_nav_size`, `show_tooltip`, `show_compact_nav`, `auto_hide`, `hide_delay`, `scroll_threshold`.
- **Galaxy aside style.** Set `markdown.asides.style: galaxy` for a rounded-card look with accent ring, gradient glow, and softer hover. The `caution` and `important` aside variants now render with proper accent colors (amber and purple) instead of being unstyled.
- **Inline code chips.** Inline code now renders with an accent-tinted text color and a subtle border. Inside asides, the chip tints to match the aside accent.
- **`fontUsed` template function.** Conditionally preloads a font file only when the current page layout actually uses it.
- **Blog list card restyle.** Blog list pages use updated card styling with a wider layout for sidebar pages.
- **Sidebar hover accent.** Sidebar links change to the accent text color on hover.
- **Docs preset accent.** The docs theme preset now uses Starlight's default accent palette, with a dark-mode variant.

### Changed

- Sidebar groups now use an accessible button with `aria-expanded` instead of a `details/summary` element.
- Prose links in body text are now underlined so they are not identified by color alone.
- The docs sidebar drawer uses `inert` instead of `aria-hidden` when closed.

### Fixed

- **SEO.** JSON-LD is now rendered as raw script content instead of being double-encoded. Added a description fallback for pages without one, self-referencing hreflang links, and pagination indexing hints.
- **Accessibility.** The image lightbox and text highlighter are now operable by keyboard. ARIA roles are corrected and expanded state stays in sync across interactive components. Missing focus indicators are restored. Copy confirmations are announced to screen readers. The reading-position toast is pausable. `prefers-reduced-motion` is respected in JavaScript-driven scrolling.
- **Theme contrast.** All accent-colored text, subtle text, aside titles, and code block line numbers now meet WCAG AA contrast ratios. The `<meta charset>` tag is emitted as the first element in `<head>`. Accent-fill foregrounds stay white in dark mode instead of flipping to black. Inline icons flow with text and honor explicit sizes. Long ToC headings stay inside the column on narrow desktop widths. List card titles are promoted to `h2` to avoid skipped heading levels. Missing labs tokens are defined so progress and badge styles render. Logical properties are used for lightbox, image-compare, and theme-toggle positioning. Mobile and icon-only controls meet the minimum touch target size.
- **Search.** The `pagefind: false` frontmatter opt-out is now honored. Documented field boosts (title 5x, tags 2.5x, description 2x) are applied. Per-language stemming and stopwords are active for 14 languages. Mermaid diagrams, KaTeX math, and code-block line numbers are excluded from the search index.
- **Summaries.** Raw `:::` directive syntax no longer leaks into auto-generated descriptions, social card text, SEO meta tags, or RSS/Atom feeds. Pages built entirely from directives fall back to rendered-text extraction.
- **Font tokens.** Removed a self-referencing `var()` cycle in font token fallbacks that caused the browser to ignore the declaration.
- **Sidebar collapse toggle** enlarged and its hover state made visible in both light and dark themes.
- **Search trigger** uses the accent hover color and applies correctly in dark mode.

### Improved

- Body font is preloaded with a metrics-matched fallback to reduce layout shift.
- The first image on each page is loaded eagerly; subsequent images honor the lazy-loading setting.
- The `scroll_to_top` plugin uses `requestAnimationFrame`-coalesced scroll updates and hides automatically near the page footer.

## 1.1.0 - 2026-07-29

### Breaking

- **The `last-updated` client plugin has been removed.** If your `sarde.yaml` lists `last-updated` under `plugins.enabled`, **delete that line** or the build will fail with `config validation failed: plugins.enabled[N]`. The date is now rendered by the theme itself on docs, labs, blog, and default layouts, with no plugin and no JavaScript required. The plugin's `date_format` option lives on as [`theme.date_format`](/reference/configuration#date-format). Relative time ("3 days ago") is no longer available; the date is always absolute.

### Added

- Docs, labs, blog, and default layouts now render a "Last updated" date server-side, so it works without JavaScript and causes no layout shift.
- New `theme.date_format` setting controls how that date is displayed. Accepts `short`, `long`, `iso`, or any Go layout string.

### Changed

- `build.last_updated` now defaults to `git` instead of `mtime`. Page timestamps come from each file's last commit rather than its filesystem modification time, which in CI is the checkout time and made every page report the same moment on every deploy. Outside a git repository the behavior is unchanged, since `git` falls back to `mtime` automatically. Set `build.last_updated: mtime` to keep the old behavior.
- `show_updated: false` now hides only the "last updated" badge. The timestamp is still resolved, so sitemap `lastmod`, SEO `dateModified`, and feed timestamps for that page remain correct. Previously it suppressed the date entirely, stripping that metadata as a side effect.

### Fixed

- "Edit this page" links now render on docs pages. `site.edit_url` was previously honored only by blog and default layouts, so a docs site could configure it correctly and see links on blog posts alone. Sites that do not set `site.edit_url` are unaffected.
- `show_updated: false` was ignored on pages that set `updated:` explicitly in frontmatter.
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
- 11 client-side plugins: scroll to top, copy section link, external links, image lightbox, focus mode, keyboard nav, reading progress, reading preferences, reading position memory, search highlighter, and text highlighter.
- 25 Markdown extensions: aside, accordion, badges, cards, columns, details, figure, file tree, gallery, image compare, link buttons, link card, math, mermaid, steps, tabs, terminal, timeline, video, annotation, copy text, highlight, icon, kbd, and spoiler.
