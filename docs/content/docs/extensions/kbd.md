---
title: Kbd
description: "Render keyboard keys and key combinations as styled keycaps"
sidebar:
  order: 34
  group: Inline
---

Keyboard keys render as styled keycaps that visually match physical keyboard buttons. Key combinations split on `+` and display each key separately with a separator.

## Basic syntax

```markdown
Press ::kbd[Ctrl+S] to save the document.
```

Press ::kbd[Ctrl+S] to save the document.

→ Two keycaps appear ("Ctrl" and "S") joined by a `+` separator.

A single key renders without a separator:

```markdown
Press ::kbd[Esc] to close the dialog.
```

Press ::kbd[Esc] to close the dialog.

→ A single "Esc" keycap appears inline with the text.

## Key combinations

The `+` character splits keys into individual keycaps. Three or more keys work the same way:

```markdown
Open the command palette with ::kbd[Ctrl+Shift+P].
```

Open the command palette with ::kbd[Ctrl+Shift+P].

→ Three keycaps appear joined by separators.

Unicode symbols work directly as key labels:

```markdown
On macOS, use ::kbd[⌘+K] instead.
```

On macOS, use ::kbd[⌘+K] instead.

→ The command symbol renders inside a keycap alongside "K".

## Sizes

Set `size="sm"` or `size="lg"` in parentheses after the closing bracket:

```markdown
::kbd[Ctrl+S](size="sm") Small
::kbd[Ctrl+S] Default
::kbd[Ctrl+S](size="lg") Large
```

::kbd[Ctrl+S](size="sm") Small
::kbd[Ctrl+S] Default
::kbd[Ctrl+S](size="lg") Large

→ The same key combination renders at three different sizes.

## Wide keys

Add `wide` for keys that are physically wider on a keyboard (Space, Enter, Backspace, Tab):

```markdown
::kbd[Space](wide) ::kbd[Enter](wide) ::kbd[Backspace](wide)
```

::kbd[Space](wide) ::kbd[Enter](wide) ::kbd[Backspace](wide)

→ Each keycap renders with a wider minimum width.

Combine `size` and `wide` in the same attribute block:

```markdown
::kbd[Space](size="lg" wide)
```

::kbd[Space](size="lg" wide)

→ A large, wide Space keycap.

## Options

| Option | Syntax | Default | Description |
|--------|--------|---------|-------------|
| `size` | `size="sm"` or `size="lg"` | (default) | Keycap size. Only `sm` and `lg` are accepted. |
| `wide` | `wide` | `false` | Wider minimum width for keys like Space or Enter. |

## Edge cases

- The `+` character splits keys into a combo. A lone `::kbd[+]` renders as a single keycap showing the `+` symbol.
- Content inside `[...]` is plain text, not parsed as Markdown. HTML characters are escaped.
- The first `]` closes the key content. There is no escaping mechanism for literal `]` characters.
- `::kbd[]` with empty brackets is not parsed as the extension. It falls through as literal text.
- The syntax is single-line only. The closing `]` must appear on the same line as the opening `::kbd[`.
- An unrecognized size value (e.g., `size="xl"`) is silently ignored. The keycap renders at the default size.
- Keycaps have a subtle pressed effect on click (CSS `:active` state), with no JavaScript involved.
- Each key in a combo receives the same size and wide modifiers.
