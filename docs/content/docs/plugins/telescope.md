---
title: Telescope
description: "A command-palette modal for fuzzy page navigation, opened with Ctrl+/ or Cmd+/"
sidebar:
  order: 43
---

A command-palette style quick-navigation modal. Press ::kbd[Ctrl]+::kbd[/] (::kbd[Cmd]+::kbd[/] on Mac) anywhere on the site to open a fuzzy search over every page, jump to a result with the keyboard, pin favorite pages, and revisit recently viewed ones. Disabled by default.

Telescope complements the [Search](/plugins/search) plugin rather than replacing it. Search indexes full page content and answers "where is X discussed"; Telescope searches only page metadata (titles, paths, tags, descriptions) and answers "take me to page X" with typo-tolerant matching and near-zero latency.

## Enable the plugin

`sarde.yaml`
```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    # ... other default plugins
    - telescope
```

## How it works

At build time, the plugin writes a `telescope-pages.json` index at the site root containing the title, URL, description, tags, collection, language, and version of every content page. Drafts, section listings, and taxonomy pages are excluded.

In the browser, a trigger button appears in the header next to the search button. Opening the palette (via the button or the keyboard shortcut) lazily fetches the index, filters it to the current language, and searches it with the bundled Fuse.js fuzzy matcher. Matches are weighted: title first, then path, tags, collection, and description. Results ranking prefers exact title matches, then title prefixes, then word-start matches, then the fuzzy score, so quick navigation always beats deep fuzzy hits. Matched substrings are highlighted.

Each result row shows the page's collection as a small accent-colored chip before the title, so identically named pages in different collections (a "Setup" page in both docs and a course, for example) are distinguishable at a glance. The chip shows the collection's configured title, falls back to its directory name, and is omitted for pages outside any collection and on very narrow screens. Collection names are also searchable, so typing `blog set` narrows to setup pages in the blog.

Two tabs are available: **Search** over all pages and **Recent** listing recently visited pages. Any result row can be pinned with its pin button or the ::kbd[Space] key; pinned pages appear in their own section at the top of the search tab. Pins and history are stored in `localStorage`, scoped per site, and validated against the fetched index on every load so entries for moved or deleted pages clean themselves up.

### Keyboard controls

| Key | Action |
|-----|--------|
| ::kbd[Ctrl]+::kbd[/] (::kbd[Cmd]+::kbd[/] on Mac) | Open the palette |
| ::kbd[↑] / ::kbd[↓] | Move the selection |
| ::kbd[Enter] | Go to the selected page |
| ::kbd[Space] | Pin or unpin the selected page (when the input is empty or while navigating) |
| ::kbd[Esc] | Close the palette |

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    telescope:
      shortcut_key: "/"
      max_results: 20
      max_recent: 10
      default_tab: search
      exclude:
        - "/changelog/*"
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `shortcut_key` | String | `/` | Key combined with ::kbd[Ctrl] or ::kbd[Cmd] to open the palette. |
| `max_results` | Number | `20` | Maximum results shown in the Search tab (1 to 100). |
| `max_recent` | Number | `10` | Recently visited pages shown (1 to 50). |
| `max_pinned` | Number | `50` | Pinned pages cap (1 to 200). Pinning past the cap evicts the oldest pin. |
| `debounce_ms` | Number | `120` | Delay in milliseconds before a keystroke triggers a search (0 to 1000). |
| `default_tab` | Select | `search` | Tab shown when the palette opens: `search` or `recent`. |
| `exclude` | List | `[]` | Glob patterns matched against page URLs to leave out of the index. Patterns match the language-free path, so one pattern covers all translations. |
| `placeholder` | String | `""` | Custom search input placeholder. Empty uses the localized default. |

## Injection rule

This plugin activates on every page. Its trigger button inserts itself into the header action area at runtime; no template changes are needed.

## Edge cases

- Out-of-range numeric values are clamped to their documented ranges rather than rejected.
- The page index is fetched only when the palette is first opened, so pages where the palette is never used pay no network cost beyond the script itself.
- In multilingual sites the palette only offers pages in the reader's current language. Versioned collections contribute every version to the index; the path in each row disambiguates.
- Pinned and recent lists store URL paths only and are keyed by the site ID, so two Sarde sites served from the same origin (for example during local development) never see each other's history.
- If the index fetch fails, the palette shows no results instead of stale or unvalidated data.
- The shortcut works on international keyboard layouts through a physical-key fallback, and the ::kbd[Ctrl] hint switches to ::kbd[⌘] on Apple devices.
- Opening the palette under the mouse cursor does not falsely select the row beneath it; hover selection only engages after the mouse actually moves.
- The dialog uses the native `dialog` element with proper combobox, listbox, and tab ARIA semantics, announces result counts to screen readers, and disables its animations when reduced motion is requested.
- The trigger button and dialog are hidden in print output.
