---
title: Annotation
sidebar:
  order: 30
  group: Inline
---

Annotations attach hoverable, keyboard-accessible tooltips to terms inline without breaking up the surrounding sentence.

## Basic syntax

```markdown
The ::annotation[hydration](The process of attaching event handlers to server-rendered HTML) step runs once on page load.
```

The ::annotation[hydration](The process of attaching event handlers to server-rendered HTML) step runs once on page load.

→ The word "hydration" appears with a dotted underline and a help cursor. Hovering or focusing it reveals a tooltip with the explanation.

<!-- SCREENSHOT: annotation-tooltip-hover — an annotation with the tooltip visible on hover -->

Labels can contain code-like tokens:

```markdown
The function ::annotation[render()](Triggers a full re-render of the component tree, reconciling virtual DOM nodes) is called on every state change.
```

The function ::annotation[render()](Triggers a full re-render of the component tree, reconciling virtual DOM nodes) is called on every state change.

→ The label "render()" appears with the same dotted underline and tooltip behavior.

## Styles

Three styles control the visual appearance of the annotated term. Set the style at the start of the explanation text:

| Style | Syntax | Visual effect |
|-------|--------|---------------|
| `underline` (default) | `::annotation[term](explanation)` | Dotted underline, help cursor. |
| `highlight` | `::annotation[term](style="highlight" explanation)` | Tinted accent background, no underline. |
| `plain` | `::annotation[term](style="plain" explanation)` | No underline, no background. Help cursor only. |

### Highlight

```markdown
The ::annotation[virtual DOM](style="highlight" An in-memory representation of the real DOM tree, used for efficient diffing and minimal updates) pattern enables fast UI updates.
```

The ::annotation[virtual DOM](style="highlight" An in-memory representation of the real DOM tree, used for efficient diffing and minimal updates) pattern enables fast UI updates.

→ "virtual DOM" appears with a tinted accent background instead of an underline.

<!-- SCREENSHOT: annotation-highlight — an annotation with the highlight style applied -->

### Plain

```markdown
Check the ::annotation[SSR](style="plain" Server-Side Rendering generates HTML on the server instead of the client) configuration for production deployments.
```

Check the ::annotation[SSR](style="plain" Server-Side Rendering generates HTML on the server instead of the client) configuration for production deployments.

→ "SSR" appears visually identical to surrounding text. The help cursor on hover is the only indicator.

## Options

| Option | Syntax | Required | Description |
|--------|--------|----------|-------------|
| Label | `[label]` | Yes | The visible text. Cannot contain `]`. |
| Explanation | `(explanation text)` | No | Tooltip content shown on hover and focus. |
| `style` | `style="value"` at start of `()` | No | One of `underline`, `highlight`, or `plain`. Defaults to `underline`. |

## Edge cases

- The first `]` closes the label. Labels cannot contain literal `]` characters.
- Omitting the parenthesized explanation renders the label with the annotation styling but produces an empty tooltip.
- `style="..."` must appear at the very start of the parenthesized content, before the explanation text.
- An unrecognized style value (e.g., `style="custom"`) is silently ignored. The annotation falls back to the default underline style.
- `style="underline"` is functionally identical to omitting the style parameter.
- Label and explanation are plain text, not further parsed as Markdown.
- Annotations are keyboard accessible. Each annotation receives `tabindex="0"`, so pressing Tab moves focus to it and reveals the tooltip.
- In print, the underline changes to solid and the tooltip is hidden.
