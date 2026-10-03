---
title: Page Banners
description: "Display contextual banners at the top of individual pages using frontmatter."
sidebar:
  order: 21
---

A page banner is a colored notice at the top of a page, such as "This page is under construction" or "This feature is deprecated". Add one with a `banner` field in frontmatter. A banner can also be applied to a whole section through `cascade`.

## Add a banner

Add a `banner` field to the page's frontmatter:

```yaml title="content/docs/migration-guide.md"
---
title: "Migration Guide"
banner:
  content: "This page is under construction."
---
```

→ A blue notice with an info icon and the text appears at the top of the page. In the docs and labs layouts it sits above the page title, below the breadcrumbs. In the default layout it sits above the page content. The presentation layout does not render banners.

## Options

The `banner` field accepts these keys:

| Key | Type | Default | Description |
|---|---|---|---|
| `content` | `string` | | The text to display. Required. The banner is hidden when empty or absent. Rendered as plain text, so Markdown and HTML are shown literally. |
| `variant` | `string` | `"note"` | Visual style. One of `note`, `tip`, `caution`, `danger`. |
| `icon` | `string` | per variant | Lucide icon name. Overrides the variant's default icon. |

## Variants

Each variant sets an accent color and a default icon:

| Variant | Use for | Accent color | Default icon |
|---|---|---|---|
| `note` | General information. The default. | Blue | `info` |
| `tip` | Helpful suggestions or recommendations | Green | `lightbulb` |
| `caution` | Warnings that deserve attention | Amber | `alert-triangle` |
| `danger` | Breaking changes or destructive actions | Red | `alert-octagon` |

Set the variant and, optionally, a different icon:

```yaml
banner:
  content: "New release available."
  variant: "tip"
  icon: "rocket"
```

→ A green notice with a rocket icon appears at the top of the page.

Use only the four variants. Sarde does not validate the value, and any other name renders without color styling and with an untranslated accessible label.

## Cascade inheritance

Apply a banner to every page in a section by setting it under `cascade` in the section's `_index.md`:

```yaml title="content/docs/experimental/_index.md"
---
title: "Experimental Features"
cascade:
  banner:
    content: "Everything in this section is experimental and subject to change."
    variant: "caution"
---
```

Every page under the section inherits the banner. A page that sets its own `banner` with `content` in frontmatter keeps its own and ignores the cascaded one.

## Customize the template

The banner renders through the `PageBanner` component. Override it by adding `layouts/components/PageBanner.html` to the project root. The component runs on every page, including pages without a banner, so wrap the markup in a check. The banner is available as `.PageBanner` with the fields `.Content`, `.Variant`, and `.Icon`:

```html title="layouts/components/PageBanner.html"
{{ if .PageBanner }}
<aside class="my-banner my-banner-{{ or .PageBanner.Variant "note" }}">
  {{ .PageBanner.Content }}
</aside>
{{ end }}
```

Without the `{{ if .PageBanner }}` check, the build fails with a nil pointer error on the first page that has no banner. See [Frontmatter](/reference/frontmatter/#banner) for the field reference.
