---
title: Search
description: "Understand the offline, client-side Orama search index and how to configure it"
sidebar:
  order: 11
---

Sarde builds an offline search index at build time and serves it client-side using Orama. No external search service is needed. Search works on any hosting provider, including static file hosts with no server-side capabilities.

## How it works

During `sarde build`, the search plugin extracts text from every page and writes a JSON index file (`search-index.<lang>.json`). The Orama client-side library loads this index on first search interaction and runs queries entirely in the browser.

Each page produces a primary document (title, description, content, tags, section) plus one sub-document per heading. Field weights prioritize titles (5x), tags (2.5x), and descriptions (2x) over body content (1x) using BM25 ranking.

## Opening search

Press **Ctrl+K** (Windows/Linux) or **Cmd+K** (macOS) to open the search modal. Alternatively, click the search button in the header.

Result: A modal overlay appears with a search input, keyboard navigation hints, and a results list.

<!-- SCREENSHOT: search-modal - search modal with results list and keyboard hints -->

Type a query to see results. Use arrow keys to navigate, Enter to select, and Escape to close.

### Full-search mode

Press **Ctrl+Space** (Windows/Linux) or **Cmd+Space** (macOS) inside the search modal to toggle full-search mode. This expands the modal into a split-pane view with the result list on the left and a preview of the selected result on the right.

<!-- SCREENSHOT: search-full-mode - split-pane search with result list and content preview -->

## Search scope

Search results are scoped automatically:

- **Versioned collections**: results are filtered to the current version. A reader on `/docs/v2/...` only sees v2 pages.
- **Multi-language sites**: results are filtered to the current language. A reader on `/fr/docs/...` only sees French pages.
- **Section filters**: results display the collection and breadcrumb path, making it clear where each result lives.

### Language support

Each language lane gets language-aware tokenization when Sarde ships a stemmer for it: Arabic, Danish, Dutch, English, Finnish, French, German, Italian, Norwegian, Portuguese, Russian, Spanish, Swedish, and Turkish. For these, search applies the language's stemming rules (searching *manger* matches *mangeons*) and filters common stopwords. Regional codes fall back to their base language (`pt-BR` uses the Portuguese stemmer). Languages without a stemmer still work with default tokenization. Only the stemmers for languages your site actually uses are shipped with the built output.

## Per-heading indexing

Every heading in a page generates a separate search document with a direct anchor link. Searching for a term that appears under a specific heading links directly to that section, not the top of the page.

For example, searching "photosynthesis" might return:

- **Photosynthesis** (page result, `/biology/photosynthesis/`)
- **Light Reactions** (heading result, `/biology/photosynthesis/#light-reactions`)
- **Calvin Cycle** (heading result, `/biology/photosynthesis/#calvin-cycle`)

## Configuration

### Enable or disable search

Search is enabled by default. Disable it in `sarde.yaml`:

```yaml
search:
  enabled: false
```

### Plugin options

Configure the search index via the search plugin config:

```yaml
plugins:
  config:
    search:
      max_content_length: 5000
      exclude:
        - "/internal/*"
        - "/drafts/*"
```

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `max_content_length` | int | `5000` | Maximum characters of body content indexed per page |
| `exclude` | string[] | `[]` | Glob patterns for page URLs to exclude from the index |

### Excluding individual pages

Exclude a single page from the search index with frontmatter:

```yaml
---
title: Internal Notes
pagefind: false
---
```

## Search highlighting

The `search-highlighter` plugin highlights search terms on the target page after a reader clicks a search result. This plugin is separate from the search plugin and must be enabled independently.

See [Plugins](/plugins/search-highlighter) for search highlighter configuration.
