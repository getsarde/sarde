---
title: Changelog
description: "Notable changes to Sarde, grouped by release, covering fixes, improvements, and new features"
sidebar:
  order: 1
---

Notable changes to Sarde, grouped by release. Bug fixes, new features, and breaking changes are listed separately within each release.

## Unreleased

### Added

- **Catalog page for tabbed collections:** when the root `_index.md` of a tabbed collection has a body, the collection root (for example `/courses/`) renders as a full-width catalog: the root's title, description, and body, then one card per tab with its tile, description, and sidebar badge. A root `_index.md` with only frontmatter still redirects to the first tab. Override the markup with `layouts/<collection>/catalog.html`. See [Tabbed Navigation](/guides/tabbed-navigation/#the-collection-root).
- **Course site template:** `sarde new site my-college --template course` (or `-t course`) creates a course site with two sample courses (lessons, assignments, and per-course announcements and schedule pages), hands-on labs grouped by course, and a site-wide announcements page. The site includes a GitHub Actions workflow that publishes it to GitHub Pages. Scroll to top, reading progress, and focus mode are enabled for long lessons and labs. An unknown template name fails before anything is written. See [CLI Commands](/reference/cli-commands/#course-template).
- **`initials` template function:** returns up to two uppercase letters from a string, for monogram tiles. `initials "Go Essentials"` gives `GE`. See [Template Functions](/reference/template-functions/).
- **Netlify, Cloudflare Pages and Vercel deploys:** `sarde deploy` now publishes to all three through their APIs, with no provider CLI needed. Tokens come from `NETLIFY_AUTH_TOKEN`, `CLOUDFLARE_API_TOKEN` and `VERCEL_TOKEN`, never from `sarde.yaml`. Only files the provider does not already have are uploaded, uploads retry on rate limits, and the command prints the live URL. New `deploy.account_id` (Cloudflare) and `deploy.team_id` (Vercel) settings; `CLOUDFLARE_ACCOUNT_ID` and `VERCEL_ORG_ID` override them. See [Deploying](/start-here/deploying/).
- **`sarde deploy --check`** verifies the token and access to the configured site or project without uploading anything.
- **`sarde deploy --format json`** streams newline-delimited progress events (steps, upload progress, the result with the live URL and deploy ID) for tools such as Sarde Studio. Failures end with the shared error envelope, which now carries an optional `code` (`auth`, `not_found`, `rate_limited`, `limit`, `network`, `canceled`, `config`, `provider`).
- **GitHub Pages custom domain:** `deploy.cname` writes the `CNAME` file on every deploy, so a force-push no longer drops a custom domain.
- **Heading font:** `theme.font_heading` (or the new `font-heading` token) sets the font for `h1` to `h6`. Headings follow `theme.font_family` until it is set. See [Theme Tokens](/reference/theme-tokens/#fonts).
- **Font size scale:** `theme.font_scale` (or the new `text-scale` token) multiplies the whole `text-xs` to `text-5xl` scale, so body text and headings grow or shrink together. Accepts `0.5` to `2`; a value outside that range is a config error.
- **Web fonts:** `theme.web_fonts: google` or `bunny` loads the theme's fonts that Sarde does not bundle from Google Fonts or Bunny Fonts, so the `clean` and `academic` presets, and any Google Fonts family set in `font_family`, `font_heading`, or `font_mono`, look the same for every visitor. Sarde requests only families in the Google Fonts library, in one stylesheet after a preconnect. Off by default; the service receives each visitor's IP address. See [Web fonts](/guides/themes-and-styling/#web-fonts).
- **Plugin requirements in the catalog:** `sarde plugins --format json` gives each plugin a `requires` list of plugin ids it needs enabled. `social_cards` requires `seo` and `search_highlighter` requires `search`. The field is omitted when empty, and the catalog version is unchanged.
- **Icon lists in timelines and details:** a list inside a timeline entry or a details panel whose items each open with an `:icon[...]` uses the icons as markers instead of bullets. See [Timeline](/extensions/timeline/#icon-lists).

### Fixed

- **Telescope finds section pages:** the index now includes `_index.md` pages such as course overviews, lab introductions, and single-page labs, which it skipped before. Sections with `render: false` and a tabbed collection root that only redirects stay out.
- **Redirect-only tabbed roots stay out of the sitemap and search:** a tabbed collection root that only redirects to its first tab is no longer listed in `sitemap.xml` or the search index.
- **Tabbed collection roots behave the same in every language:** the root of a tabbed collection redirected only in the default language; translated roots rendered as a list page with the first tab's sidebar. Every language now follows the same rule.
- **List items ending in an icon or key stay aligned:** in a list that holds paragraphs or other blocks, a list item whose last element was inline, such as an `:icon[...]`, a `::kbd[...]` key, or a highlight, got the spacing meant for blocks. That pushed the element above its line of text, as in the course template's schedule items. Only block elements now get that spacing.
- **Site-wide `head.tags` are rendered:** tags listed under `head.tags` in `sarde.yaml` were parsed and never written to any page, and `head` tags set through a section's `cascade` were dropped the same way. Both now render on every page they apply to, site tags before a page's own. A `Head` component ejected from an earlier version needs `{{ siteHeadTags . }}` before its `renderHeadTags` line. See [Head tags](/reference/configuration/site-and-branding/#head-tags).
- **Head tag scripts and styles keep their quotes:** the content of a `script`, `style`, or `noscript` head tag was HTML-escaped, so quotes and `&&` became entities and broke the CSS or JavaScript. That content is now written as given, with any closing tag inside it neutralized.
- **Clear error outside a site:** `sarde build`, `dev`, `check-links`, and `validate` run in a folder with no `sarde.yaml` and no `content/` folder now stop with `not a Sarde site` and the commands to fix it, including `cd` into a subfolder that holds a site. Previously `sarde dev` printed a `discovering content` error and kept serving 404 pages. See [Troubleshooting](/resources/troubleshooting#not-a-sarde-site).
- **`sarde new site` says to `cd` first:** creating a site in a subfolder now ends with `Run 'cd <path>' then 'sarde dev'` instead of only `Run 'sarde dev'`, which failed from the parent folder.
- **Links in timelines and cards are underlined:** plain Markdown links inside a timeline entry or a card were told apart from the text by color alone, because both extensions opt out of prose styles. They now get the same subtle underline as links in body text, tabs, and steps.
- **Lab pages show their tags:** tags set in a lab page's frontmatter now appear under the title and link to their tag pages, as they already did on docs pages. The labs layout never rendered them, so they only appeared on the tag pages themselves.
- **Code blocks follow dark mode:** sites created with `sarde new site` set `darkMode.selector` to `.dark` in `kazari.config.yaml`, a class Sarde never sets, so code blocks stayed light after switching to dark mode. Dark mode for code blocks now always comes from `markdown.codeblocks.dark_mode_selector` in `sarde.yaml`, which matches the theme toggle by default, so existing sites are fixed on their next build. A `darkMode` entry in `kazari.config.yaml` is ignored and the build warns about it; delete it. New sites no longer write one.
- **Section badges show in the sidebar:** a `sidebar.badge` set in a section's `_index.md` never appeared on the section's group row, because only page entries copied the badge. Groups now show it, and a `sidebar.yaml` badge override still takes precedence. Sidebar badges also use a compact, normal-case style and wrap under a long label instead of overlapping it; badges in page content are unchanged.
- **The labs Overview entry is translatable:** the first sidebar entry of a lab was hardcoded English. It now uses the `labs.overview` UI string. Templates can read the new `NavNode.LabelKey` field for generated entries like this one.
- **Deploys no longer publish `.sarde.lock`:** the GitHub Pages deployer copied the build's lock file into the published branch. Every deployer now skips it.
- **Custom deploy commands on Windows run as typed:** a command that quoted a path, such as `echo done > "C:\My Site\log.txt"`, failed with "The filename, directory name, or volume label syntax is incorrect", because the quotes were escaped for a program other than `cmd.exe`. The command now reaches `cmd.exe` unchanged, and a site path in Windows' `\\?\` form no longer makes the command run in `C:\Windows`.
- **GitHub Pages deploys read the remote from the site root:** the deployer looked up the `origin` remote in the process working directory, so `sarde deploy path/to/site` run from elsewhere pushed to the wrong repository or failed. Git and custom-command output no longer write straight to stdout.
- **GitHub Pages deploys keep files that start with an underscore:** the branch GitHub Pages serves had no `.nojekyll` file, so Jekyll ran over the published site and dropped any file or directory whose name starts with `_`. The `github` deployer now adds an empty `.nojekyll` file on every deploy.
- **`robots.txt` no longer points at a missing sitemap:** the robots plugin wrote a `Sitemap:` line even when the sitemap plugin was off. The line now appears only when the sitemap plugin runs.

### Changed

- **Announcement banners appear above the site header:** banners from the announcements plugin now render in a full-width, one-line strip above the navigation bar on every layout (two lines on phones), instead of inside the page content, and stay with the header while scrolling until dismissed. The strip shows one banner at a time, so `stack` mode behaves like `first`, and the new `banner-height` theme token (`3rem`) sets its height. Banners that are dismissed, outside their date window, or targeted at other pages no longer flash on load. A base layout ejected from an earlier version keeps the banner where it calls `{{ announcementBanner }}`; wrap that call and the Header component in `<div class="sarde-masthead">` to get the new placement. See [Announcements](/plugins/announcements/#placement).
- **Tabbed sidebars lead with an Overview entry:** the sidebar inside a docs tab no longer wraps every page in a group named after the tab, which the tab switcher already shows. The tab's pages start at the top level, and its `_index.md` becomes an **Overview** entry at the top. Rename it with `sidebar.label` or hide it with `sidebar.hidden` in the tab's `_index.md`. Tabs with their own `nav.yaml` are unchanged. See [Tabbed Navigation](/guides/tabbed-navigation/#the-sidebar-inside-a-tab).
- **Docs tab switcher redesign:** the tab dropdown at the top of the sidebar is now a tinted card with a tile showing the tab's icon (or its initials), the collection title as a small label, and up/down chevrons. Menu entries use the same tiles, and the sidebar collapse button sits further out so it no longer covers the switcher. Sites that override `DocsTabSwitcher` keep their own markup.
- **Deploy dependency:** `github.com/zeebo/blake3` (and its dependency `github.com/klauspost/cpuid/v2`) for the Cloudflare Pages asset hash.
- **Inferred descriptions no longer appear under the page title:** a page without a `description` in frontmatter used to show its first paragraph under the title and again as the start of the body. The header now shows only descriptions you write; the inferred one is still used for meta tags, search, social cards, and listings. To show text under the title, set `description` in frontmatter. Templates can check the new `.Page.DescriptionInferred` field.
- **Lab progress bar moved to the top:** the "Step X of Y" bar on lab pages now sits between the lab badge and the page title instead of below the content, and is no longer repeated at the bottom.
- **Single font names get quotes and a fallback:** a font field that holds one family name, such as `font_family: Plus Jakarta Sans`, is now written as `'Plus Jakarta Sans', system-ui, sans-serif`. A name with a digit word, such as `Source Sans 3`, was invalid CSS unquoted, and with no fallback a missing font fell back to the browser default. The fallback follows the family's Google Fonts category. Values that are already stacks are unchanged. See [Shortcut fields](/guides/themes-and-styling/#shortcut-fields).


### Fixed

- **`search.enabled: false` now disables search:** the documented switch was never read: the index was still built, the runtime script still injected, and the header button still rendered. It now unregisters the search plugin and hides the button and modal, without warning about an unused `plugins.config.search` block. The undocumented `search.provider: "disabled"` value the template used to honor is gone; `provider` accepts only `orama`. `header.search: false`, also previously inert, now hides the header button while keeping the index and modal.
- **Search results reach the highlighter:** result links never carried the `?q=` parameter that the `search_highlighter` plugin reads, so highlighting never triggered from search. Links now append `?q=<query>` before any `#anchor` whenever the highlighter plugin is enabled, and stay clean otherwise.
- **Search modal strings are translatable:** twenty-one runtime labels ("Recent", "Type to start searching", the result count, the full-search preview labels, and others) were hard-coded English and two of them overwrote labels the template had already translated. They now resolve through new `search.*` i18n keys with `_one`/`_other` plural forms and `{count}`, `{term}`, `{min}` placeholders, injected per page language, with English fallbacks in the script. The loading spinner gained an accessible label.
- **Client plugin config no longer clobbers other plugins:** the inline `window.__SARDE__.pluginConfig` script assigned the object wholesale; it now merges, and every injected plugin gets an entry even with an empty config, so `pluginConfig[slug]` reliably signals that a plugin is active on the page.
- **Mobile sidebar drawer recovers from the back/forward cache:** a docs or lab page cached with the drawer open came back with the header and content still inert. The drawer now resets to a closed, interactive state on restore, and backdrop cleanup no longer leaves a stray `transitionend` listener after each close.

### Added

- **Plugin translation capability**: `BeforeRenderContext.T(lang, key)` resolves UI strings through the site's i18n layers for any plugin's per-page hook.

### Docs

- Added a Course Template page under Teaching: creating a course site from the template, what it contains, courses, schedules, announcements, labs, replacing the sample content, and publishing on GitHub Pages.
- Added a Template API page that maps what a template receives, the base shell contract, and where components, partials, and functions are documented, and a Route Data reference listing every field available as the dot context with the conditions under which each is populated. A test in `internal/engine` fails when a struct field is added without a docs entry.

## 1.4.0 - 2026-08-28

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

## 1.3.0 - 2026-08-06

### Added

- **Theme CSS hot-swap in dev mode:** editing a theme stylesheet (under `themes/<name>/css/` or the `--theme-dev` source tree) now reassembles the CSS bundle in place instead of running a full site rebuild. Typical refresh time is 2-7ms instead of ~1.3s. The browser restyles via the existing CSS swap mechanism without a page reload.
- **Draft banner:** pages with `draft: true` now display a visual banner in dev mode so draft status is immediately visible in the browser.
- **Prose link underline in extensions:** links inside tabs, details/accordion, steps, and columns now receive the same underline and hover styling as regular prose links. Previously the `not-content` wrapper on these extensions excluded them.
- **Aside link underline:** links inside galaxy-style asides now show a visible underline at rest and shift to high-contrast text on hover, matching Starlight's aside link treatment.
- **Hero CTA hover effects:** the primary call-to-action button gains an accent glow, a brightness shift that works in both light and dark themes, and a press state on click.

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

- **`scroll_to_top` threshold is now pixels instead of a percentage:** the default changed from `30` (meaning 30% of total scroll) to `300` (meaning 300 pixels from the top). Sites with a custom `threshold` value will see different behavior. A new `position` option (`left`, `center`, `right`) controls button placement, defaulting to center.
- **Plugin slugs and config keys standardized to snake_case:** all 11 client plugin slugs changed from kebab-case to snake_case (e.g. `scroll-to-top` to `scroll_to_top`). Legacy kebab-case spellings in `plugins.enabled`, `plugins.disabled`, and `plugins.config` are accepted as deprecated aliases with a build-time warning. The `scroll_to_top` plugin's config field keys also moved from camelCase to snake_case (e.g. `showTooltip` to `show_tooltip`); old spellings are similarly aliased. Update your `sarde.yaml` to the new names to silence the warnings.

### Added

- **Telescope command-palette plugin** for fuzzy page navigation. Opens with a keyboard shortcut and lets readers search and jump to any page instantly. Enable it with `telescope` in `plugins.enabled`.
- **Custom `:::` directives:** sites and themes can define new Markdown `:::` block directives with a YAML schema, an HTML template, and an optional CSS sidecar placed in a `directives/` directory. `sarde new directive <name>` scaffolds the starter files. `sarde directives --check` validates definitions. The dev server live-reloads directive changes.
- **Plugin-shipped directives:** external plugins can now include `:::` directives by placing definitions in `plugins/<slug>/directives/`.
- **Social card redesign:** cards now use a bottom-anchored editorial layout with a background gradient, corner logo mark, and optional watermark. New config options: `logo`, `watermark`, `watermark_opacity`, `accent_color_2`, `bg_image`, `bg_gradient`, `logo_size`, and custom `fonts`. Per-page `og_card` frontmatter overrides colors and toggles. A disk cache under `.cache/social_cards/` makes repeat builds near-instant.
- **Site logo in the header:** `site.logo` now renders an image before the site title, with separate light and dark variants. `replaces_title: true` hides the text and shows only the logo. Raster logos get `width`/`height` attributes to prevent layout shift.
- **Signed releases:** release archives are now signed with ed25519. `sarde update` verifies the signature chain before applying an update.
- **Passive update notices:** `sarde build` and `sarde dev` print a one-line notice when a newer release is available. Suppressed in CI, with `--quiet`, and when stderr is not a terminal. Set `SARDE_NO_UPDATE_CHECK=1` to disable.
- **`sarde update` improvements:** the update command now detects package-manager installs (Homebrew, Scoop, Chocolatey, winget) and prints the matching upgrade command instead of self-replacing. Release notes are shown before the confirmation prompt. A `--yes` flag supports non-interactive use. Permission errors print actionable advice.
- **Benchmark harness:** new `sarde-bench` tool measures build performance with median wall time, pages/sec, and per-phase breakdowns. `sarde build --format json` emits machine-readable build results.
- **Hero CTA icons:** hero call-to-action buttons accept an optional `icon` field that renders a Lucide icon after the label.
- **Keyboard navigation enhancements:** side navigation arrows are now mdBook-sized with reserved page margin so they never overlap content. A styled tooltip shows the target page title and keyboard shortcut on hover and focus. Below 1280px, the arrows restyle as compact floating buttons. Buttons auto-hide after inactivity and reappear on scroll. New options: `side_nav_size`, `show_tooltip`, `show_compact_nav`, `auto_hide`, `hide_delay`, `scroll_threshold`.
- **Galaxy aside style:** set `markdown.asides.style: galaxy` for a rounded-card look with accent ring, gradient glow, and softer hover. The `caution` and `important` aside variants now render with proper accent colors (amber and purple) instead of being unstyled.
- **Inline code chips:** inline code now renders with an accent-tinted text color and a subtle border. Inside asides, the chip tints to match the aside accent.
- **`fontUsed` template function:** conditionally preloads a font file only when the current page layout actually uses it.
- **Blog list card restyle:** blog list pages use updated card styling with a wider layout for sidebar pages.
- **Sidebar hover accent:** sidebar links change to the accent text color on hover.
- **Docs preset accent:** the docs theme preset now uses Starlight's default accent palette, with a dark-mode variant.

### Changed

- Sidebar groups now use an accessible button with `aria-expanded` instead of a `details/summary` element.
- Prose links in body text are now underlined so they are not identified by color alone.
- The docs sidebar drawer uses `inert` instead of `aria-hidden` when closed.

### Fixed

- **SEO:** JSON-LD is now rendered as raw script content instead of being double-encoded. Added a description fallback for pages without one, self-referencing hreflang links, and pagination indexing hints.
- **Accessibility:** the image lightbox and text highlighter are now operable by keyboard. ARIA roles are corrected and expanded state stays in sync across interactive components. Missing focus indicators are restored. Copy confirmations are announced to screen readers. The reading-position toast is pausable. `prefers-reduced-motion` is respected in JavaScript-driven scrolling.
- **Theme contrast:** all accent-colored text, subtle text, aside titles, and code block line numbers now meet WCAG AA contrast ratios. The `<meta charset>` tag is emitted as the first element in `<head>`. Accent-fill foregrounds stay white in dark mode instead of flipping to black. Inline icons flow with text and honor explicit sizes. Long ToC headings stay inside the column on narrow desktop widths. List card titles are promoted to `h2` to avoid skipped heading levels. Missing labs tokens are defined so progress and badge styles render. Logical properties are used for lightbox, image-compare, and theme-toggle positioning. Mobile and icon-only controls meet the minimum touch target size.
- **Search:** the `pagefind: false` frontmatter opt-out is now honored. Documented field boosts (title 5x, tags 2.5x, description 2x) are applied. Per-language stemming and stopwords are active for 14 languages. Mermaid diagrams, KaTeX math, and code-block line numbers are excluded from the search index.
- **Summaries:** raw `:::` directive syntax no longer leaks into auto-generated descriptions, social card text, SEO meta tags, or RSS/Atom feeds. Pages built entirely from directives fall back to rendered-text extraction.
- **Font tokens:** removed a self-referencing `var()` cycle in font token fallbacks that caused the browser to ignore the declaration.
- **Sidebar collapse toggle** enlarged and its hover state made visible in both light and dark themes.
- **Search trigger** uses the accent hover color and applies correctly in dark mode.

### Improved

- Body font is preloaded with a metrics-matched fallback to reduce layout shift.
- The first image on each page is loaded eagerly; subsequent images honor the lazy-loading setting.
- The `scroll_to_top` plugin uses `requestAnimationFrame`-coalesced scroll updates and hides automatically near the page footer.

## 1.1.0 - 2026-07-29

### Breaking

- **The `last-updated` client plugin has been removed:** if your `sarde.yaml` lists `last-updated` under `plugins.enabled`, **delete that line** or the build will fail with `config validation failed: plugins.enabled[N]`. The date is now rendered by the theme itself on docs, labs, blog, and default layouts, with no plugin and no JavaScript required. The plugin's `date_format` option lives on as [`theme.date_format`](/reference/configuration/theme-and-appearance/#date-format). Relative time ("3 days ago") is no longer available; the date is always absolute.

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
