---
title: Custom Directives
description: "Author your own ::: block directives with a YAML schema, an HTML template, and an optional CSS sidecar"
sidebar:
  order: 2
---

Custom directives let you add new `:::` block syntax to your site without writing Go or waiting for a release. A directive is three files in a `directives/` folder at your site root (themes can ship them too, under `themes/<name>/directives/`):

- `directives/<name>.yaml` describes the directive: its kind, label, and fields.
- `directives/<name>.html` is a Go `html/template` that renders it.
- `directives/<name>.css` (optional) styles it; the CSS is bundled into your site's stylesheet automatically.

Once the files exist, the directive works in any Markdown page, exactly like built-in extensions.

## Basic syntax

Every custom directive uses one standard form:

````
:::name[optional label] key="value" flag
Body content.
:::
````

The `[label]` bracket and `key="value"` attributes are both optional. Attribute values must be quoted. Close the block with `:::` or a named `:::/name` fence.

## A worked example: pull quote (container)

Run the scaffolder (see below) or create the files by hand. `directives/pullquote.yaml`:

```yaml
name: pullquote        # must equal the filename stem
kind: container         # body is Markdown, rendered recursively
label: "Pull Quote"
description: "A styled pull quote with attribution"
fields:
  - { name: author, label: Author, type: string }
```

`directives/pullquote.html`:

```html
<blockquote class="pullquote">
  {{ .Body }}
  {{- if .Attrs.author }}<cite>{{ .Attrs.author }}</cite>{{ end }}
</blockquote>
```

Use it in any page:

````
:::pullquote author="Ada Lovelace"
The engine weaves **algebraic patterns** just as the Jacquard loom weaves flowers and leaves.
:::
````

Because the kind is `container`, the body is full Markdown: bold text, links, and even other directives (built-in or custom) all render normally.

## Leaf directives

A `leaf` directive captures its body as raw text instead of Markdown. The text is HTML-escaped before it reaches your template, which makes leaves the right choice for verbatim content:

```yaml
name: ascii-box
kind: leaf
label: "ASCII Box"
description: "Preformatted raw text in a box"
```

```html
<pre class="ascii-box">{{ .Body }}</pre>
```

## Template data

Your template executes against this pipeline value:

| Field | Contents |
|-------|----------|
| `.Name` | The directive name. |
| `.Label` | The `[bracket]` label, or empty. |
| `.Attrs` | A map of the opening-fence attributes; missing keys read as empty strings. |
| `.Body` | Container: the rendered body HTML. Leaf: the escaped raw text. |

Templates also get the shortcode helper functions (icons, i18n strings, URL helpers).

## Schema reference

| Key | Required | Notes |
|-----|----------|-------|
| `name` | yes | Lowercase letters, digits, hyphens; must start with a letter and equal the filename stem. |
| `kind` | yes | `container` (Markdown body) or `leaf` (raw text body). |
| `label` | yes | Shown in the Sarde Studio directive picker. |
| `description` | yes | Shown in the picker and in `sarde directives`. |
| `category` | no | Defaults to `custom`. |
| `bracket` | no | Enables `:::name[Label]`; keys: `label`, `required`, `placeholder`. |
| `fields` | no | Opening-fence attributes; each has `name`, `label`, `type` (`string`, `enum`, `boolean`, `number`, `icon`), plus optional `options`, `default`, `required`, `placeholder`. |

## Styling

Put styles in `directives/<name>.css`. The file is appended to the fingerprinted `sarde.<hash>.css` bundle, unlayered, so your rules win over layered theme CSS without extra specificity tricks (see the CSS layers section of [Themes and Styling](/guides/themes-and-styling/)).

Use the `--sd-*` theme tokens so your directive follows theme presets and dark mode automatically:

```css
.pullquote {
  border-inline-start: 3px solid var(--sd-accent);
  color: var(--sd-gray-1);
}
```

Editing the CSS during `sarde dev` rebuilds the bundle and reloads the browser.

## Scaffolding

```
sarde new directive pullquote
```

This creates commented starter versions of all three files and prints usage hints. It rejects names that collide with built-in directives up front.

## Validation

Definitions are validated on every build. A broken directive (bad kind, unknown YAML key, missing template, invalid field type, duplicate field names) is skipped with a warning; the rest of the build continues. Warnings appear:

- in `sarde build` output,
- as failures under `sarde validate --strict`,
- in the dev server's browser warning panel,
- via `sarde directives --check`, a fast lint that loads and validates definitions without building; it exits 1 when anything is wrong.

`sarde directives --format json` lists your site's directives merged with the built-in catalog, each entry stamped with `source: "site"` (or `builtin`). Sarde Studio's directive picker reads this and offers custom directives automatically.

## Edge cases

- **Built-in names win.** A `directives/card.yaml` is ignored with a warning; the built-in `:::card` keeps working.
- **Theme vs site.** A site directive overrides a theme directive of the same name.
- **Unknown names fall through.** `:::something-unregistered` renders as plain paragraph text, matching built-in behavior.
- **Leaf bodies are escaped.** Raw HTML inside a leaf body renders as text, not markup. Use a container if you need rendered content.
