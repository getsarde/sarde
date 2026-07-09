---
title: LLMs.txt
sidebar:
  order: 35
  group: Server-Side
---

Generates an `llms.txt` file in the build output, listing every content page with its title and absolute URL. This file follows the llms.txt convention for making site content discoverable by large language models. Enabled by default.

## Output

The generated file is written to `llms.txt` at the site root. The format is:

```text
# Site Title

> Site description from sarde.yaml

## Pages

- [Getting Started](https://example.com/docs/getting-started/)
- [Configuration](https://example.com/docs/reference/configuration/)
- [Writing Content](https://example.com/docs/guides/writing-content/)
```

The heading uses the `site.title` from `sarde.yaml`. The blockquote uses `site.description`, if set. Each page is a Markdown-formatted link with the page's title and absolute URL.

## Configuration

This plugin uses the global `llms_txt` config section in `sarde.yaml`.

`sarde.yaml` (defaults shown)
```yaml
llms_txt:
  enabled: true
  include_blog: true
```

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | Boolean | `true` | Generate the `llms.txt` file. |
| `include_blog` | Boolean | `true` | Include pages from blog-type collections (`blog/`, `posts/`, `articles/`, `news/`). |

## Filtering blog content

Set `include_blog: false` to exclude blog posts from the output. This keeps the file focused on documentation and reference content.

```yaml
llms_txt:
  include_blog: false
```

Blog-type collections are identified by directory name: `blog`, `posts`, `articles`, and `news`.

## Excluded pages

The following pages are always excluded from the output:

- Draft pages
- Section index pages (`_index.md`)
- The homepage

## Disabling the plugin

Remove `llms_txt` from the enabled list, or set `enabled: false`:

```yaml
llms_txt:
  enabled: false
```

## Edge cases

- If `site.title` is not set, the heading falls back to "Site".
- If `site.description` is empty, the blockquote line is omitted.
- The file uses absolute URLs constructed from `site.url`. If no URL is configured, URLs are relative to the build root.
- Pages are listed in build order (the order they appear in the page list), not alphabetically.
