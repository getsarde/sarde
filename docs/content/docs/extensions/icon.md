---
title: Icon
sidebar:
  order: 33
  group: Inline
---

Icons render inline SVG glyphs that match the surrounding text size and color. The default icon set is [Lucide](https://lucide.dev), with support for additional Iconify-compatible sets.

## Basic syntax

```markdown
Press the :icon[settings] button to open preferences.
```

Press the :icon[settings] button to open preferences.

→ A small gear icon appears inline, sized to match the text.

## Icon sets

Bare names resolve against the default set (Lucide). Prefix a name with the set identifier to use a different set:

```markdown
:icon[tabler:home] :icon[simple-icons:react] :icon[brands:github]
```

:icon[tabler:home] :icon[simple-icons:react] :icon[brands:github]

→ Each icon renders from its respective set. `brands:` is an alias for `simple-icons:`.

Additional icon sets must be loaded before use. See the [Icons guide](/docs/guides/icons/) for setup with `sarde icons add` and the `icons.sets` configuration.

## Sizing

The default size is `1em`, matching the surrounding text. Apply a CSS class for fixed sizes:

```markdown
:icon[heart class="sarde-icon-sm"] Small
:icon[heart] Default
:icon[heart class="sarde-icon-lg"] Large
```

:icon[heart class="sarde-icon-sm"] Small
:icon[heart] Default
:icon[heart class="sarde-icon-lg"] Large

→ Three hearts render at `1rem`, `1em`, and `2rem` respectively.

Set an explicit pixel width with the `width` attribute. Height auto-derives from the icon's aspect ratio:

```markdown
:icon[heart width="32"]
```

:icon[heart width="32"]

→ The heart renders at 32 pixels wide.

## Colors

Lucide icons use `stroke="currentColor"`, so they inherit the text color of their container. Override inline with `style`:

```markdown
:icon[circle-check style="color: green"] Passed
:icon[circle-x style="color: red"] Failed
```

:icon[circle-check style="color: green"] Passed
:icon[circle-x style="color: red"] Failed

→ The checkmark appears green and the x-circle appears red.

Icons inside colored extensions (asides, badges) automatically pick up the container's text color.

## Transforms

Rotate an icon by a given number of degrees:

```markdown
:icon[arrow-right rotate="90"] Points down
:icon[arrow-right rotate="180"] Points left
```

:icon[arrow-right rotate="90"] Points down
:icon[arrow-right rotate="180"] Points left

→ The arrow rotates to point in the specified direction.

Flip an icon horizontally, vertically, or both:

```markdown
:icon[arrow-right flip="horizontal"] Flipped
```

:icon[arrow-right flip="horizontal"] Flipped

→ The arrow mirrors to point left. Valid values: `horizontal`, `vertical`, `both`.

## Accessibility

Icons are decorative by default (`aria-hidden="true"`). Add a `title` to make an icon meaningful to assistive technology:

```markdown
:icon[triangle-alert title="Warning"]
```

:icon[triangle-alert title="Warning"]

→ The icon receives a `<title>` element and `role="img"`, making it visible to screen readers.

An `aria-label` attribute has the same effect without adding a visible `<title>` element.

## Options

| Attribute | Example | Description |
|-----------|---------|-------------|
| `class` | `class="sarde-icon-lg"` | Appended to the base class. Use `sarde-icon-sm` or `sarde-icon-lg` for size variants. |
| `width` | `width="24"` | SVG width in pixels. Height auto-derives from the icon's aspect ratio. |
| `height` | `height="24"` | SVG height in pixels. Width auto-derives if omitted. |
| `style` | `style="color: red"` | Inline CSS. Use `color` to tint stroke-based icons via `currentColor`. |
| `rotate` | `rotate="90"` | Rotation in degrees. |
| `flip` | `flip="horizontal"` | Mirror the icon: `horizontal`, `vertical`, or `both`. |
| `title` | `title="Warning"` | Adds a `<title>` element and promotes to `role="img"`. |
| `aria-label` | `aria-label="Warning"` | ARIA label. Promotes to `role="img"` without a visible `<title>`. |

Any unrecognized attribute is passed through as a literal SVG attribute.

## Edge cases

- An unknown icon name falls back to the `circle-help` icon (a question mark in a circle).
- Lucide provides semantic shorthands: `warning` resolves to `alert-triangle`, `success` to `check-circle`, `close` to `x`, `gear` to `settings`, among others. These shorthands only work when the default prefix is `lucide`.
- `:icon[]` with an empty name is not parsed as the extension. It renders as literal text.
- The syntax is single-line only. The closing `]` must appear on the same line as the opening `:icon[`.
- Attribute values can be double-quoted (`"value"`), single-quoted (`'value'`), or bare (`value`). Bare values cannot contain spaces.
- If `icons.default_prefix` is set to a prefix with no loaded provider, both the requested icon and the `circle-help` fallback fail. The token disappears from the output.
- Local icons (SVGs in the `icons/` directory) take priority over bundled set icons for bare names.
