---
title: Highlight
description: "Draw attention to key words or phrases with a colored background highlight"
sidebar:
  order: 33
---

Highlighted text draws attention to key words or phrases with a colored background, similar to a marker pen on paper.

## Basic syntax

```markdown
Photosynthesis converts ==light energy== into chemical energy stored in glucose.
```

Photosynthesis converts ==light energy== into chemical energy stored in glucose.

→ "light energy" appears with a yellow background highlight.

Multiple highlights can appear in the same sentence:

```markdown
The ==most important== part of any routing algorithm is ==path normalization==.
```

The ==most important== part of any routing algorithm is ==path normalization==.

→ Each phrase receives an independent highlight background.

## Combining with other extensions

Highlights work alongside other inline extensions in the same sentence:

```markdown
The ==recommended== command is ::copy[sarde build --minify].
```

The ==recommended== command is ::copy[sarde build --minify].

→ "recommended" is highlighted while the command renders as a copyable snippet.

Highlights also work inside block extensions such as asides:

````
:::tip
You can use ==highlighted text== and `inline code` inside callout blocks.
:::
````

:::tip
You can use ==highlighted text== and `inline code` inside callout blocks.
:::

→ The highlighted phrase renders with the yellow background inside the green tip callout.

## Edge cases

- Content between `==` markers is plain text. Markdown formatting (bold, italic, links, code spans) inside a highlight renders as literal characters, not formatted output.
- Highlights are single-line only. An opening `==` on one line cannot be closed on the next. Unmatched markers render as literal `==` text.
- Empty highlights (`====`) are not parsed. At least one character must appear between the markers.
- No attributes or parameters are supported. The syntax is always `==text==`.
- There is no escaping mechanism for literal `==` inside highlighted text.
- In dark mode, the background shifts to an amber tone with lighter text for contrast.
