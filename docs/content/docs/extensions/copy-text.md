---
title: Copy Text
description: "Turn a text snippet into a one-click clipboard copy styled like inline code"
sidebar:
  order: 32
  group: Inline
---

Copy text turns a snippet into a one-click clipboard copy, styled like inline code with a small copy button attached.

## Basic syntax

```markdown
Install the package with ::copy[npm install sarde] and configure the dev server.
```

Install the package with ::copy[npm install sarde] and configure the dev server.

→ "npm install sarde" appears in a monospace pill with a clipboard icon button. Clicking the button copies the text.

Multiple copy snippets can appear in the same line:

```markdown
Use ::copy[--port 8080] and ::copy[--host 0.0.0.0] flags together.
```

Use ::copy[--port 8080] and ::copy[--host 0.0.0.0] flags together.

→ Each snippet renders independently with its own copy button.

## Standalone values

Place a copy snippet on its own line for file paths, commands, or configuration values:

```markdown
::copy[~/.config/sarde/config.yaml]
```

::copy[~/.config/sarde/config.yaml]

→ The path appears as a standalone copyable pill.

Backslashes and special characters pass through literally:

```markdown
::copy[C:\Users\You\AppData\Local\sarde\config.yaml]
```

::copy[C:\Users\You\AppData\Local\sarde\config.yaml]

→ The Windows path renders verbatim with no escape processing.

## Copy feedback

Clicking the copy button triggers a brief visual confirmation. The clipboard icon swaps to a green checkmark for 1.5 seconds, then reverts.

| State | Icon | Duration |
|-------|------|----------|
| Default | Clipboard | Until clicked. |
| Copied | Green checkmark | 1.5 seconds, then reverts to default. |

<!-- SCREENSHOT: copy-text-copied-state — the copy-text widget showing the green checkmark after a click -->

The copy operation uses the Clipboard API (`navigator.clipboard.writeText`). No tooltip or toast message appears.

## Edge cases

- The first `]` closes the content. There is no escaping mechanism for literal `]` characters inside the snippet.
- `::copy[]` with empty brackets is not parsed as the extension. It falls through as literal text.
- Content is single-line only. The syntax cannot span multiple lines.
- No attributes or parameters are supported. Everything between `[` and `]` is the copied text.
- The Clipboard API requires a secure context (HTTPS or localhost). On insecure pages, the button click does nothing.
- In print, the copy button is hidden. Only the code text remains visible.
- The button's color transition respects `prefers-reduced-motion`.
