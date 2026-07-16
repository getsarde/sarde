---
title: Spoiler
description: "Hide text behind a blur until the reader hovers, focuses, or clicks to reveal it"
sidebar:
  order: 36
  group: Inline
---

Spoilers hide text behind a blur until the reader hovers, focuses, or clicks to reveal it. The syntax follows the Discord convention of double pipes.

## Basic syntax

```markdown
The secret ingredient is ||a pinch of concurrency||.
```

The secret ingredient is ||a pinch of concurrency||.

→ The text appears blurred with a gray background. Hovering or clicking reveals the content.

<!-- SCREENSHOT: spoiler-blurred — a spoiler in its default blurred state -->

Multiple spoilers can appear in the same sentence:

```markdown
The answer is ||42|| and the password is ||swordfish||.
```

The answer is ||42|| and the password is ||swordfish||.

→ Each spoiler blurs and reveals independently.

## Revealing content

Three interactions reveal a spoiler:

| Trigger | Behavior |
|---------|----------|
| Hover | Temporarily reveals the text. Blurs again when the cursor leaves. |
| Focus (Tab key) | Temporarily reveals the text. Blurs again when focus moves away. |
| Click, Enter, or Space | Persistently toggles the revealed state. Click again to re-hide. |

Hover and focus work through CSS alone. The persistent click toggle requires JavaScript and adds a `revealed` class to the element.

<!-- SCREENSHOT: spoiler-revealed — a spoiler after being clicked to reveal -->

## Inside other extensions

Spoilers work inside block extensions such as asides:

````
:::tip
The answer is ||hidden behind a spoiler||.
:::
````

:::tip
The answer is ||hidden behind a spoiler||.
:::

→ The spoiler renders with the blur effect inside the green tip callout.

## Edge cases

- No attributes or parameters are supported. The syntax is always `||text||`.
- Content between `||` markers is plain text. Markdown formatting (bold, italic, links, code spans) inside a spoiler renders as literal characters, not formatted output.
- Spoilers are single-line only. An opening `||` on one line cannot be closed on the next. Unmatched markers render as literal `||` text.
- Empty spoilers (`||||`) are not parsed. At least one character must appear between the markers.
- There is no escaping mechanism for literal `||` inside spoiler text.
- In print, spoilers are always fully revealed (blur removed, text visible).
- The reveal transition respects `prefers-reduced-motion` by disabling the animation.
- In dark mode, the blurred background uses darker gray tones while the reveal behavior remains the same.
