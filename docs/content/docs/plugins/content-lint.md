---
title: Content Lint
sidebar:
  order: 31
  group: Server-Side
---

Checks Markdown content for common quality issues during the build: heading length violations, skipped heading levels, missing image alt text, empty links, and missing frontmatter fields. Enabled by default.

## Default configuration

Content lint runs automatically with all rules active. No configuration is needed for the defaults.

`sarde.yaml` (defaults shown)
```yaml
content_lint:
  enabled: true
  rules:
    heading_max_length: 60
    heading_increment: true
    image_alt_required: true
    no_empty_links: true
    frontmatter_required: []
```

## Rules

### Heading max length

Flags headings whose text exceeds the configured character limit.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `heading_max_length` | Number | `60` | Maximum heading text length in characters. Set to `0` to disable. |

```markdown
## This heading is longer than sixty characters and triggers a warning from the linter
```

→ Warning: `heading exceeds 60 characters (83)`

### Heading increment

Flags headings that skip levels (e.g., jumping from `##` to `####`).

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `heading_increment` | Boolean | `true` | Check that heading levels increase by one at a time. |

```markdown
## Section

#### Subsection
```

→ Warning: `heading level skipped from h2 to h4`

### Image alt text

Flags images with empty alt text (`![](path)`).

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `image_alt_required` | Boolean | `true` | Require non-empty alt text on all images. |

```markdown
![](diagram.png)
```

→ Warning: `image missing alt text`

### Empty links

Flags links with no visible text (`[](url)`).

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `no_empty_links` | Boolean | `true` | Check for links that have no display text. |

```markdown
[](https://example.com)
```

→ Warning: `link has empty text`

### Required frontmatter

Flags pages that are missing specified frontmatter fields.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `frontmatter_required` | List | `[]` | Frontmatter fields that must be present. Supported: `title`, `description`, `date`, `image`, `tags`, `categories`, and any custom field in `params`. |

```yaml
content_lint:
  rules:
    frontmatter_required:
      - title
      - description
```

→ Warning: `required frontmatter field "description" is missing`

## Disabling the plugin

Remove `content_lint` from the enabled list, or set `enabled: false`:

```yaml
content_lint:
  enabled: false
```

## Disabling individual rules

Set a boolean rule to `false`, or set a numeric rule to `0`:

```yaml
content_lint:
  rules:
    heading_increment: false
    heading_max_length: 0
```

## Incremental builds

During incremental rebuilds (e.g., in `sarde dev`), only changed pages are re-linted. Issues on unchanged pages are not re-reported until the next full build or until the file is edited.

## Configuration

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | Boolean | `true` | Enable or disable the plugin entirely. |
| `rules.heading_max_length` | Number | `60` | Maximum heading text length. `0` disables. Must be at least 1 when set. |
| `rules.heading_increment` | Boolean | `true` | Require heading levels to increment by one. |
| `rules.image_alt_required` | Boolean | `true` | Require non-empty image alt text. |
| `rules.no_empty_links` | Boolean | `true` | Flag links with empty display text. |
| `rules.frontmatter_required` | List | `[]` | Frontmatter fields that must be present on every non-draft page. |

## Edge cases

- Draft pages are skipped entirely.
- Headings and images inside fenced code blocks are ignored. The fenced block detector handles both backtick and tilde fences, and requires the closing fence to match the opener's length.
- Image and link patterns inside inline code spans (backtick-delimited) are also ignored.
- Lint issues report line numbers relative to the source file (adjusted for frontmatter offset).
- The `frontmatter_required` check recognizes built-in fields (`title`, `description`, `date`, `image`, `tags`, `categories`) and falls back to the `params` map for custom fields.
