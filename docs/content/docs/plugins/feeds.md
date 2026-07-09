---
title: Feeds
sidebar:
  order: 32
  group: Server-Side
---

Generates RSS 2.0 and Atom 1.0 feed files for collections that have feeds enabled. Both the `rss` and `atom` plugins are enabled by default and share the same configuration structure.

## Output files

Each feed-enabled collection gets two XML files in the build output:

| File | Format | Location |
|------|--------|----------|
| `feed.xml` | RSS 2.0 | `<collection>/feed.xml` |
| `atom.xml` | Atom 1.0 | `<collection>/atom.xml` |

For a blog collection, the output paths are `blog/feed.xml` and `blog/atom.xml`.

## Which collections get feeds

By default, blog-type collections (`blog/`, `posts/`, `articles/`, `news/`) have feeds enabled through auto-detection (`Feed: true` in the inferred config). Docs-type collections do not.

Override per collection in `sarde.yaml`:

```yaml
collections:
  docs:
    feed: true
  blog:
    feed: false
```

Alternatively, restrict feeds to specific collections via the plugin config:

```yaml
plugins:
  config:
    rss:
      collections:
        - blog
        - news
    atom:
      collections:
        - blog
```

When `collections` is set in the plugin config, it takes precedence over the per-collection `feed` flag.

## Configuration

The `rss` and `atom` plugins accept the same options.

`sarde.yaml`
```yaml
plugins:
  config:
    rss:
      limit: 20
      collections:
        - blog
    atom:
      limit: 20
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `limit` | Number | `20` | Maximum number of entries per feed. |
| `collections` | List | `[]` | Restrict feeds to these collection names. Empty uses collections with `feed: true`. |

## Feed content

### RSS items

Each RSS item includes:

| Field | Source |
|-------|--------|
| `title` | Page title |
| `link` | Absolute page URL |
| `description` | Page `description`, falling back to `summary` |
| `pubDate` | Page date in RFC 1123 format |
| `guid` | Absolute page URL |

The channel's `lastBuildDate` is the `pubDate` of the most recent item.

### Atom entries

Each Atom entry includes:

| Field | Source |
|-------|--------|
| `title` | Page title |
| `link` | Absolute page URL |
| `id` | Absolute page URL |
| `updated` | Page `updated` date, falling back to `date`, in RFC 3339 format |
| `summary` | Page `description`, falling back to `summary` |
| `author` | `author` field from page `params`, if present |

The feed's `updated` timestamp uses the most recent entry's date.

## Multi-language sites

On multi-language sites, the plugins generate one feed per (collection, language) pair. Feed URLs follow the localized URL layout. For example, with French as a non-default language:

- `blog/feed.xml` (default language)
- `fr/blog/feed.xml` (French)

Pages are grouped by their `Lang` field. The default language omits the prefix, matching the site's `prefix-except-default` URL strategy.

## Disabling feeds

Remove `rss` and/or `atom` from the enabled list:

```yaml
plugins:
  enabled:
    - search
    - seo
    - sitemap
    - robots
    # rss and atom removed
    - content_lint
    - link_validator
```

## Edge cases

- Draft pages are excluded from all feeds.
- If a page has no `date`, the RSS `pubDate` field is omitted. The Atom `updated` field falls back to the current build time.
- If a page has no `description` or `summary`, the description/summary field is omitted.
- Each XML file includes the standard `<?xml version="1.0" encoding="UTF-8"?>` header.
- On incremental rebuilds, feeds are regenerated for all feed-enabled collections (not only changed pages) to ensure completeness.
