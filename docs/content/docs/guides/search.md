---
title: Search
description: "Understand the offline, client-side Orama search index and how to configure it"
sidebar:
  order: 11
---

Every Sarde site ships with full-text search that runs in the reader's browser. The build writes the index, and the [Orama](https://orama.com/) library queries it client-side, so search works on any host, including static file hosts with no server.

## How it works

During `sarde build`, the search plugin extracts text from every page and writes one JSON index file per language (`search-index.<lang>.json`). The browser loads the index when the reader first opens search.

Each page produces a primary document (title, description, content, tags, section) plus one document per `h2` to `h4` heading. Titles weigh 5x, tags 2.5x, and descriptions 2x against body content at 1x.

## Opening search

Press **Ctrl+K** (Windows and Linux) or **Cmd+K** (macOS), or click the search button in the header.

→ A modal appears with a search input, keyboard hints, and a results list.

<!-- SCREENSHOT: search-modal - search modal with results list and keyboard hints -->

Type a query to see results. Use the arrow keys to move through them, Enter to open one, and Escape to close the modal.

### Full-search mode

Press **Ctrl+Space** (Windows and Linux) or **Cmd+Space** (macOS) inside the search modal to toggle full-search mode.

→ The modal expands into a split pane with the result list on the left and a preview of the selected result on the right.

<!-- SCREENSHOT: search-full-mode - split-pane search with result list and content preview -->

## Search scope

Search results are scoped to where the reader is:

- **Versioned collections:** results are filtered to the current version. A reader on `/docs/v2/...` only sees v2 pages.
- **Multi-language sites:** each language has its own index, so a reader on `/fr/docs/...` only sees French pages.
- **Sections:** each result shows its collection and breadcrumb path, and the modal offers one filter chip per collection to narrow results.

### Language support

Each language gets language-aware tokenization when Sarde ships a stemmer for it: Arabic, Danish, Dutch, English, Finnish, French, German, Italian, Norwegian, Portuguese, Russian, Spanish, Swedish, and Turkish. For these, search applies the language's stemming rules (searching *manger* matches *mangeons*) and filters common stopwords. Regional codes fall back to their base language (`pt-BR` uses the Portuguese stemmer). Languages without a stemmer still work with default tokenization. The built output includes only the stemmers for the languages the site uses.

## Per-heading indexing

Every `h2`, `h3`, and `h4` heading generates its own search document with an anchor link. A result for a heading opens the page at that section instead of at the top. The page title (`h1`) and `h5` and `h6` headings are not indexed as separate results.

For example, searching "photosynthesis" might return:

- **Photosynthesis** (page result, `/biology/photosynthesis/`)
- **Light Reactions** (heading result, `/biology/photosynthesis/#light-reactions`)
- **Calvin Cycle** (heading result, `/biology/photosynthesis/#calvin-cycle`)

## Configuration

Search needs no configuration. These settings turn it off, translate its labels, and control which pages reach the index.

### Enable or disable search

Search is enabled by default. Disable it in `sarde.yaml`:

```yaml title="sarde.yaml"
search:
  enabled: false
```

Disabling search removes the header button, the search modal, the runtime script, and the index files from the build. A `plugins.config.search` block is kept without a warning, so toggling search off and on leaves the rest of the config untouched.

To hide only the header button and keep the index and modal, use `header.search`. This suits a custom element with the `data-search-trigger` attribute that opens search instead:

```yaml title="sarde.yaml"
header:
  search: false
```

### Translating the search UI

Every label in the search modal, including the runtime strings such as "Recent", "Type to start searching", the result count, and the full-search preview labels, resolves through the [UI strings](/guides/internationalization/#translation-strings) cascade under the `search.*` keys. Override any of them in your project `i18n/<lang>.yaml`:

```yaml
search:
  recent: "Récents"
  results_count_one: "{count} résultat pour '{term}'"
  results_count_other: "{count} résultats pour '{term}'"
```

Keys ending in `_one` and `_other` are plural forms picked from the page language's plural rules. Keep the `{count}`, `{term}`, and `{min}` placeholders in translations. The full key list is in the [search plugin reference](/plugins/search/#ui-strings).

### Index size and excluded URLs

Limit how many characters of each page are indexed, or keep URL patterns out of the index, in the search plugin config:

```yaml title="sarde.yaml"
plugins:
  config:
    search:
      max_content_length: 5000
      exclude:
        - "/internal/*"
        - "/drafts/*"
```

`exclude` patterns match the page URL. A `*` matches within a single path segment, so `/internal/*` excludes `/internal/notes/` but not `/internal/notes/old/`. List each depth you need to exclude.

The [search plugin options](/plugins/search/#configuration) list each key with its type and default.

### Excluding individual pages

Exclude a single page from the search index with frontmatter:

```yaml
---
title: Internal Notes
pagefind: false
---
```

Set `pagefind: false` under `cascade` in a section's `_index.md` to exclude every page in the section.

## Search highlighting

The `search_highlighter` plugin highlights the search terms on the target page after a reader clicks a result. It is off by default and must be added to `plugins.enabled`. When it is enabled, result links carry the query as a `?q=` parameter, placed before any `#heading` anchor. When it is not, result links stay clean.

See [Search Highlighter](/plugins/search-highlighter/) to enable it and for its options.
