---
title: Math
description: "Render inline and block LaTeX math expressions using KaTeX"
sidebar:
  order: 21
---

Math expressions render LaTeX notation using KaTeX. Inline expressions use single dollar signs. Display (block) expressions use double dollar signs.

## Inline math

Wrap an expression in single `$` delimiters to render it inline with the surrounding text:

```markdown
The quadratic formula is $x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}$ where $a \neq 0$.
```

→ The formula renders inline as typeset math, matching the line height of the surrounding paragraph.

## Display math

Wrap an expression in double `$$` delimiters on their own lines for a centered, full-width block:

```markdown
$$
E = mc^2
$$
```

→ The equation appears centered on its own line in a larger font.

Multi-line expressions are supported:

```markdown
$$
\int_0^\infty e^{-x^2} dx = \frac{\sqrt{\pi}}{2}
$$
```

Single-line display math is also valid:

```markdown
$$ F = ma $$
```

## KaTeX rendering

KaTeX assets (CSS, JS, fonts) are automatically injected on pages that contain math markup. The KaTeX plugin detects `class="sarde-math"` in the rendered HTML and appends the required scripts and stylesheet.

No configuration is needed. KaTeX is enabled by default via `markdown.katex: true` in `sarde.yaml`.

To force KaTeX assets on every page (useful when math is loaded dynamically):

```yaml
plugins:
  config:
    katex:
      always: true
```

To disable math rendering entirely:

```yaml
markdown:
  katex: false
```

## Edge cases

- A single `$` followed by a digit on the closing side (e.g., `costs $9 and $29`) is not treated as math. The parser avoids false matches with currency amounts.
- Empty expressions (`$$`) are ignored.
- LaTeX characters (`{`, `}`, `&`, `\`) are preserved for KaTeX. Only `<` and `>` are HTML-escaped to prevent injection.
- Display math can interrupt a paragraph (no blank line required before `$$`).

See the [KaTeX plugin](/plugins/katex) for runtime asset injection and configuration.
