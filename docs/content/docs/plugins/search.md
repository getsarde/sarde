---
title: Search
description: "Build a JSON search index at build time with embedded, offline Orama search"
sidebar:
  order: 39
---

Builds a JSON search index at build time and ships the Orama client-side search runtime. Every page gets the search script injected, and the search UI is available via ::kbd[Ctrl+K] (or ::kbd[Cmd+K] on macOS). Enabled by default.

## How it works

During the build, the plugin extracts searchable content from every non-draft page and writes a JSON index file per language. At runtime, the embedded Orama client loads the index and handles queries entirely client-side, with no server required.

## Search index structure

Each page produces one primary document plus one document per heading. This enables both page-level and section-level search results.

### Page document fields

| Field | Source |
|-------|--------|
| `title` | Page title |
| `url` | Page URL |
| `description` | Page description |
| `content` | HTML-stripped page content, truncated to `max_content_length` |
| `section` | Collection name |
| `tags` | Page tags |
| `version` | Page version ID (if versioned) |
| `breadcrumb` | Collection > Section > Subsection path |

### Heading documents

Each heading on the page generates an additional document with the heading text as `title` and the anchor URL (`page-url#heading-id`) as `url`. Heading documents include the breadcrumb extended with the page title (e.g., "Docs > Guides > Writing Content").

## Output files

| File | Description |
|------|-------------|
| `search-index.en.json` | Search index for English (or the default language) |
| `search-index.fr.json` | Search index for French (one file per language) |
| `assets/vendor/orama/orama.esm.js` | Orama search engine (ES module) |
| `assets/js/static-search.js` | Search initialization and UI script |

## Configuration

`sarde.yaml`
```yaml
plugins:
  config:
    search:
      max_content_length: 5000
      exclude:
        - /admin/*
        - /drafts/*
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `max_content_length` | Number | `5000` | Maximum characters of page content stored in the index. Higher values increase index size but improve result relevance for long pages. |
| `exclude` | List | `[]` | URL glob patterns for pages to exclude from the index. |

Individual pages can opt out with `pagefind: false` in their frontmatter, and a section can opt out all of its descendants via `cascade: { pagefind: false }` in its `_index.md`. See [Search](/guides/search/#excluding-individual-pages) and the [frontmatter reference](/reference/frontmatter/).

The global `search` config section controls whether search is enabled site-wide:

```yaml
search:
  enabled: true
  provider: "orama"
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `search.enabled` | Boolean | `true` | Enable or disable search site-wide. |
| `search.provider` | String | `"orama"` | Search provider. Currently only `"orama"` is supported. |

## Multi-language support

On multi-language sites, the plugin groups pages by their `Lang` field and writes a separate index file for each language (`search-index.en.json`, `search-index.fr.json`, etc.). Pages without a language tag fall into the `en` index by default. The search UI loads the index matching the current page's language.

## Version-scoped search

When a collection uses versioning, each page carries a `version` field in its search document. The search UI filters results to the currently viewed version, so readers searching within `v2` documentation do not see results from `v1`.

## Content extraction

Page content is extracted by parsing the rendered HTML and collecting its visible text with whitespace collapsed. Regions that render as UI rather than prose are skipped: `script`, `style`, and `svg` elements, Mermaid diagram sources, raw math (KaTeX) sources, and code block line-number gutters. Code text itself stays searchable. HTML entities are decoded, so searching for `R&D` matches pages containing `R&amp;D` markup. The raw HTML is first truncated to 3x `max_content_length` bytes (to limit processing), then extracted, then truncated to the final `max_content_length`. Truncation is rune-safe (it never splits a multi-byte UTF-8 character).

## Incremental rebuild caching

The plugin caches extracted search documents per page across incremental rebuilds. When a page's content digest has not changed, its cached documents are reused without re-extraction. On a full build, the cache is repopulated from scratch (because breadcrumb inputs from `_index.md` section titles may have changed even if page content did not).

## Disabling search

Remove `search` from the enabled list, or set the global config:

```yaml
search:
  enabled: false
```

## Edge cases

- Draft pages are excluded from the index.
- Duplicate URLs (permalink collisions) are deduplicated: only the first page encountered keeps its entry.
- Heading documents within a page are also deduplicated by their anchor URL.
- The Orama runtime and search UI script are only copied on full builds. Incremental rebuilds skip the asset copy since the files are identical.
- Pages excluded via `exclude` patterns are matched against their permalink using glob semantics.
