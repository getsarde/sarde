---
title: Page Banners
description: "Display contextual banners at the top of individual pages using frontmatter."
sidebar:
  order: 21
---

Page banners are colored notices that appear at the top of page content, above the article body. Configure them per-page through frontmatter, with four visual variants and support for cascade inheritance across sections.

## Quick example

Add a `banner` field to any page's frontmatter:

```yaml
---
title: "Migration Guide"
banner:
  content: "This page is under construction."
---
```

The banner renders at the top of the content area, before the article body.

## Options

| Key | Type | Default | Description |
|---|---|---|---|
| `content` | `string` | | The text to display. Required. The banner is hidden when empty or absent. |
| `variant` | `string` | `"note"` | Visual style. One of: `note`, `tip`, `caution`, `danger`. |
| `icon` | `string` | per-variant | Lucide icon name. Overrides the variant's default icon. |

Each variant includes a default icon: `note` uses `info`, `tip` uses `lightbulb`, `caution` uses `alert-triangle`, `danger` uses `alert-octagon`. Set `icon` to use any icon from the Lucide set instead:

```yaml
banner:
  content: "New release available."
  variant: "tip"
  icon: "rocket"
```

## Variants

### Note

General-purpose information. This is the default when `variant` is omitted.

```yaml
banner:
  content: "This API is in beta and may change without notice."
```

Renders with a blue accent.

### Tip

Helpful suggestions or recommendations.

```yaml
banner:
  content: "A faster alternative is available. See the Performance guide."
  variant: "tip"
```

Renders with a green accent.

### Caution

Important warnings that deserve attention.

```yaml
banner:
  content: "This page documents a deprecated feature."
  variant: "caution"
```

Renders with an amber accent.

### Danger

Critical alerts about breaking changes or destructive actions.

```yaml
banner:
  content: "Following these steps will delete all existing data."
  variant: "danger"
```

Renders with a red accent.

## Cascade inheritance

Apply a banner to every page in a section by setting it in the section's `_index.md` under `cascade`:

```yaml
---
title: "Experimental Features"
cascade:
  banner:
    content: "Everything in this section is experimental and subject to change."
    variant: "caution"
---
```

Every page under this section inherits the banner. A page that defines its own `banner` in frontmatter overrides the cascaded value. The page-level banner always wins.

## Customizing the template

The banner renders through the `PageBanner` component. Override it by placing a custom template at `layouts/components/PageBanner.html` in the project root. The template receives the full route data context, with `.PageBanner.Content` and `.PageBanner.Variant` available as fields.
