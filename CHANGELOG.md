# Changelog

All notable changes to Sarde are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.4.0] - 2026-08-28

### Added

- **`sarde plugins` command** prints the plugin catalog: every plugin accepted in `plugins.enabled`, covering built-in server and client plugins plus external plugins found under the project's `plugins/` directory, with descriptions, default state, and configurable fields. `--format json` emits the catalog for tooling. Metadata only: `plugins.disabled` and premium license checks are not applied, since builds enforce both.
- **Selective `sarde theme eject`** accepts paths to copy only those files or directories (`sarde theme eject layouts/_blog/single.html css/blog.css`); everything else keeps falling back to the embedded theme, including fixes in later releases. `--name <slug>` writes into `themes/<slug>/` and sets the theme's `slug`, and `--list` prints every ejectable path. `theme.yaml` is copied whenever the destination has none.
- **JSON error envelope** for `build`, `dev`, and `validate` with `--format json`: a fatal error is emitted to stdout as a single JSON document carrying a `kind`, a `message`, and per-field `details` for configuration validation failures, including the allowed values for enumeration checks. Human-readable error text still goes to stderr in both formats.
- **Ambiguous link detection** reports a bare `name.md` link, which never resolves by design, as an `ambiguous_link` finding with fix advice instead of a generic broken target.
- **Link source positions** are now included in link validation findings: the 1-based line and column of the link in its source file, in both the pretty and JSON reports.
- **Typed plugin field blueprints** publish plugin configuration fields with their type, label, hint, default, numeric range, and select options, so catalogs and settings interfaces can render them without re-parsing the YAML.

### Changed

- **Theme stylesheets fall back per file** instead of all-or-nothing. Each of the 24 stylesheets Sarde looks for is read from the theme's `css/` directory when present and from the embedded theme otherwise. A partial `css/` directory no longer drops the stylesheets it omits, so a theme can ship only the ones it changes.
- Server plugin defaults moved to per-plugin blueprint files, so `sarde plugins` and the build resolve the same field list.
- Page cache schema version bumped to carry link positions. The first build after upgrading re-renders every page.

### Fixed

- **`sarde theme eject` now places every template directory under `layouts/`**: the `_taxonomy`, `_labs`, `_presentation`, `_slides`, and `shortcodes` directories were written at the theme root, where the template and shortcode lookups never read them, so the ejected copies had no effect. Eject also no longer leaves empty `_default/`, `_docs/`, `_blog/`, `components/`, and `partials/` directories at the theme root.
- **Install script authenticates GitHub API calls**, so installs no longer fail against the unauthenticated rate limit.

### Docs

- Added a Customization section covering layouts and templates, using themes, creating a preset, and creating a theme, including the blog-layout template lookup chain and template naming conventions.
- Completed the theme token reference: the text-safe hue variants, the aside code mix ratios, inline code text, and the `accent-text` derivation were previously undocumented.

## [1.3.0] - 2026-08-06

### Added

- **Theme CSS hot-swap in dev mode.** Editing a theme stylesheet (under `themes/<name>/css/` or the `--theme-dev` source tree) now reassembles the CSS bundle in place instead of running a full site rebuild. Typical refresh time is 2-7ms instead of ~1.3s. The browser restyles via the existing CSS swap mechanism without a page reload.
- **Draft banner.** Pages with `draft: true` now display a visual banner in dev mode so draft status is immediately visible in the browser.
- **Prose link underline in extensions.** Links inside tabs, details/accordion, steps, and columns now receive the same underline and hover styling as regular prose links. Previously the `not-content` wrapper on these extensions excluded them.
- **Aside link underline.** Links inside galaxy-style asides now show a visible underline at rest and shift to `--sd-text-high` on hover, matching Starlight's aside link treatment.
- **Hero CTA hover effects.** The primary call-to-action button gains an accent glow, a brightness shift that works in both light and dark themes, and a press state on click.

### Changed

- **Kazari updated to v1.2.0.**
- Active TOC link border thickened from 1px to 2px for better visibility.
- Active TOC link text in dark mode is now brighter (mixed 70% accent with white) so it stands out from the muted neighbors.

### Fixed

- **`color-scheme: light` override** added so `light-dark()` CSS functions resolve correctly when the OS prefers dark mode.
- **Hardcoded dark-mode OKLCH values** in banners and TOC links replaced with theme tokens so custom accent palettes apply everywhere.
- **Telescope collection badge** moved after the description for better scannability.

### Docs

- Added a dedicated "Updating Sarde" page covering `sarde update`, signed releases, package manager detection, and passive update notices.
- Rewrote the "Getting Started" opening, dev server, build, and scaffold sections with warmer lead-ins for the educator audience.
- Documented the theme CSS fast path in the dev server reference page.

## [1.2.0] - 2026-08-04

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

## [1.1.0] - 2026-07-29

### Breaking

- **The `last-updated` client plugin has been removed.** If your `sarde.yaml` lists `last-updated` under `plugins.enabled`, delete that line or the build will fail. The date is now rendered by the theme itself on docs, labs, blog, and default layouts, with no plugin and no JavaScript required. The plugin's `date_format` option lives on as `theme.date_format`. Relative time ("3 days ago") is no longer available; the date is always absolute.

### Added

- Docs, labs, blog, and default layouts now render a "Last updated" date server-side, so it works without JavaScript and causes no layout shift.
- New `theme.date_format` setting controls how that date is displayed. Accepts `short`, `long`, `iso`, or any Go layout string.

### Changed

- `build.last_updated` now defaults to `git` instead of `mtime`. Page timestamps come from each file's last commit rather than its filesystem modification time, which in CI is the checkout time and made every page report the same moment on every deploy. Outside a git repository the behavior is unchanged, since `git` falls back to `mtime` automatically. Set `build.last_updated: mtime` to keep the old behavior.
- `show_updated: false` now hides only the "last updated" badge. The timestamp is still resolved, so sitemap `lastmod`, SEO `dateModified`, and feed timestamps for that page remain correct. Previously it suppressed the date entirely, stripping that metadata as a side effect.

### Fixed

- "Edit this page" links now render on docs pages. `site.edit_url` was previously honored only by blog and default layouts.
- `show_updated: false` was ignored on pages that set `updated:` explicitly in frontmatter.
- Sarde now warns once when `build.last_updated: git` cannot be used (git missing, not a repository, or a shallow clone) instead of silently falling back to file modification times.
- The "Made with Sarde" footer credit linked to a domain that does not resolve. It now points to the documentation site.

## [1.0.1] - 2026-07-25

### Fixed

- Added `linux/arm64` binaries, which were missing from the 1.0.0 release. ARM Linux servers and CI runners can now install with the standard script.

## [1.0.0] - 2026-07-24

The first stable release. Everything below has accumulated since the 0.1.x previews.

### Breaking

- The `static/` project directory is renamed to `public/`. Files placed in `public/` are copied as-is to the output directory, exactly as `static/` worked before. Rename your project's `static/` directory to `public/` before your next build.

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

### Fixed

- Frontmatter date fields (`date`, `updated`, `publish_date`, `expiry_date`) may now be left empty. An empty value means "not set" instead of aborting the build.
- Frontmatter parse errors now name the file they came from.
- Page cache now re-validates link targets on every build, even for cached pages.
- Custom heading IDs (`## Heading {#custom-id}`) are now preserved instead of being overwritten with auto-generated slugs.
- The `site.heading_links` config option is now functional. Heading IDs are always assigned (required by the TOC, search, and link validation), but the visible anchor element is toggled by this setting.
- Content lint rules no longer trigger false positives inside fenced code blocks or inline code spans.
- Content lint line numbers now match the source file on disk instead of being offset by the frontmatter block's height.
- The `same_site_policy` link validation option works correctly on incremental rebuilds.

### Improved

- Multi-language sites reuse taxonomy structures on body-only incremental rebuilds, skipping redundant taxonomy and data file processing per language.
- WebSocket hub uses ping/pong keepalive (30-second interval) to detect stale connections.
- Unchanged headings are reused during incremental rebuilds, and link validation is scoped to changed pages only.

[Unreleased]: https://github.com/frostybee/sarde/compare/v1.4.0...HEAD
[1.4.0]: https://github.com/frostybee/sarde/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/frostybee/sarde/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/frostybee/sarde/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/frostybee/sarde/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/frostybee/sarde/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/frostybee/sarde/releases/tag/v1.0.0
