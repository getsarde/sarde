---
title: Shortcodes
sidebar:
  order: 6
---

Shortcodes are reusable content snippets invoked from Markdown files and backed by Go templates. They use double-brace angle-bracket delimiters, distinct from [extensions](/extensions/using-extensions) (`:::` Markdown syntax) and [components](/reference/ui-components) (`{{ component }}` in layout templates).

```
{{< alert type="tip" title="Before you start" >}}
Save your work before running `sarde build --clean`.
{{< /alert >}}
```

Sarde ships one built-in shortcode: `alert`. It renders valid HTML but has no bundled CSS. Override it or create new shortcodes by dropping `.html` files in `layouts/shortcodes/`.

## Syntax

Two forms are supported. Self-closing shortcodes have no inner content:

```
{{< youtube id="dQw4w9WgXcQ" />}}
```

Paired shortcodes wrap inner content between opening and closing tags:

```
{{< alert type="info" >}}
Inner **Markdown** content is rendered to HTML.
{{< /alert >}}
```

Shortcode names accept word characters and hyphens (`[\w-]+`). Shortcodes are expanded in Markdown content files only, not in layout templates.

## Parameters

Parameters use `key=value` syntax. All values are parsed as strings (`map[string]string`). Missing keys return an empty string.

| Format | Example |
|--------|---------|
| Double-quoted | `type="info"` |
| Single-quoted | `title='My Title'` |
| Bare (no spaces) | `cols=3` |

:::caution
All parameter values are strings. `{{ if .Params.disabled }}` evaluates to true for `disabled="false"` because the value is a non-empty string. Compare explicitly instead: `{{ if eq .Params.disabled "true" }}`.
:::

## The `alert` shortcode

The only built-in shortcode. Renders a `<div>` wrapper with optional title and body.

| Param | Type | Default | Description |
|-------|------|---------|-------------|
| `type` | string | `"info"` | Alert variant. Applied as CSS class `alert--{type}`. |
| `title` | string | - | Optional title text, rendered in a `.alert__title` div. |

Inner content is optional. When present, it is rendered as Markdown and wrapped in a `.alert__body` div.

```
{{< alert type="warning" title="Deprecation notice" >}}
This API will be removed in v2.0.
{{< /alert >}}
```

The `alert` shortcode ships with no default CSS. Style the `.alert`, `.alert__title`, and `.alert__body` classes in your project's CSS.

:::tip
For styled callouts that work out of the box, use the aside extension instead: `:::note`, `:::tip`, `:::caution`, `:::danger`. See [Aside](/extensions/aside).
:::

## Shortcode context

Shortcode templates receive a `ShortcodeContext` with four fields.

| Field | Type | Description |
|-------|------|-------------|
| `.Params` | `map[string]string` | Parsed key=value attributes from the tag |
| `.Inner` | `template.HTML` | Rendered HTML of the body (paired shortcodes only, empty for self-closing) |
| `.Page` | `*Page` | The page being rendered (Title, Slug, Tags, Content, etc.) |
| `.Site` | `*SiteContext` | Site-wide context (Title, BaseURL, Collections, etc.) |

`.Inner` is already-rendered HTML. Emit it directly with `{{ .Inner }}` without re-escaping.

## Writing a custom shortcode

Create a `.html` file in `layouts/shortcodes/`. The filename (minus extension) becomes the shortcode name.

`layouts/shortcodes/callout.html`

```html
{{- $type := or .Params.type "note" -}}
{{- $icon := "" -}}
{{- if eq $type "tip" }}{{ $icon = "lightbulb" }}
{{- else if eq $type "caution" }}{{ $icon = "alert-triangle" }}
{{- else }}{{ $icon = "info" }}
{{- end -}}
<div class="callout callout--{{ $type }}">
  <div class="callout__header">
    {{ icon $icon "callout__icon" }} {{ or .Params.title ($type | title) }}
  </div>
  {{- if .Inner }}
  <div class="callout__body">{{ .Inner }}</div>
  {{- end }}
</div>
```

Invoke it from any Markdown file:

```
{{< callout type="tip" title="Performance" >}}
Enable `build.minify` for smaller output files.
{{< /callout >}}
```

## Overriding shortcodes

Shortcodes resolve through three layers. Each layer fully replaces same-named shortcodes from the layer below.

| Priority | Layer | Directory |
|----------|-------|-----------|
| 1 (lowest) | Embedded | Compiled into the binary |
| 2 | Theme | `themes/<name>/layouts/shortcodes/` |
| 3 (highest) | Project | `layouts/shortcodes/` |

Place an `alert.html` in `layouts/shortcodes/` to override the built-in `alert` with your own markup. The filename minus `.html` must match the shortcode name exactly.

## Nesting and limits

Shortcodes can nest. The expansion pipeline runs up to 10 fixed-point passes (`maxRecursionDepth`). Same-name nesting is supported through depth-balanced matching:

```
{{< tabs >}}
  {{< tabs >}}
    Inner tabs
  {{< /tabs >}}
{{< /tabs >}}
```

Nested shortcodes inside a paired body expand before the body is rendered as Markdown. This prevents Goldmark from HTML-escaping the raw `{{<` syntax before inner shortcodes can be matched.

:::note
Shortcode syntax inside fenced code blocks (`` ``` `` or `~~~`) is never expanded. Examples showing `{{< ... >}}` in code fences render literally.
:::

## Unknown and unclosed shortcodes

An unknown shortcode name produces a build warning and preserves the raw text in the output. An unclosed paired shortcode tag behaves the same way. Neither causes a build failure. A typo in a shortcode name shows up as literal `{{< ... >}}` text on the rendered page.

## Template functions

Most template functions are available inside shortcode templates. A few are unavailable or degraded. See [Template Functions: Shortcode availability](/reference/template-functions#shortcode-availability) for the full list.
