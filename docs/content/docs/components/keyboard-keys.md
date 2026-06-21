---
title: "Keyboard Keys"
description: "Styled keyboard key indicators."
sidebar:
  order: 16
---

Keyboard keys render inline text as styled `<kbd>` elements that resemble physical keys — raised with a bottom-border shadow.

## Basic Usage

```md
::kbd[Ctrl+S]
```

Renders as: ::kbd[Ctrl+S]

## Examples

Common key combinations:

```md
Copy: ::kbd[Ctrl+C]
Paste: ::kbd[Ctrl+V]
Search: ::kbd[Cmd+K]
Switch tabs: ::kbd[Alt+Tab]
Open terminal: ::kbd[Ctrl+`]
```

Use keyboard keys inline in instructions:

```md
Press ::kbd[Ctrl+S] to save the file.
Open the command palette with ::kbd[Cmd+Shift+P].
```

Renders as: Press ::kbd[Ctrl+S] to save the file. Open the command palette with ::kbd[Cmd+Shift+P].
