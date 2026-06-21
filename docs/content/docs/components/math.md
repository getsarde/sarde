---
title: "Math"
description: "Mathematical expressions with KaTeX."
sidebar:
  order: 19
---

Math expressions render with [KaTeX](https://katex.org). Use `$` delimiters for inline math and `$$` for display (block) math. KaTeX is loaded on-demand — only pages that contain math expressions load the library.

## Inline Math

Wrap an expression in single `$` delimiters to render it inline:

```md
The energy-mass equivalence is $E = mc^2$.
```

Renders as: The energy-mass equivalence is $E = mc^2$.

More examples:

```md
The quadratic formula gives $x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}$.

For all $n \geq 1$, the sum $\sum_{i=1}^{n} i = \frac{n(n+1)}{2}$.

Use Greek letters like $\alpha$, $\beta$, $\gamma$, and $\Delta$ inline.
```

The quadratic formula gives $x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}$.

For all $n \geq 1$, the sum $\sum_{i=1}^{n} i = \frac{n(n+1)}{2}$.

Use Greek letters like $\alpha$, $\beta$, $\gamma$, and $\Delta$ inline.

## Display Math

Use double `$$` delimiters for centered, block-level equations:

```md
$$
\int_{-\infty}^{\infty} e^{-x^2} dx = \sqrt{\pi}
$$
```

$$
\int_{-\infty}^{\infty} e^{-x^2} dx = \sqrt{\pi}
$$

A matrix:

```md
$$
A = \begin{pmatrix} a & b \\ c & d \end{pmatrix}
$$
```

$$
A = \begin{pmatrix} a & b \\ c & d \end{pmatrix}
$$

A multi-line aligned equation:

```md
$$
\begin{aligned}
  (a + b)^2 &= a^2 + 2ab + b^2 \\
  (a - b)^2 &= a^2 - 2ab + b^2
\end{aligned}
$$
```

$$
\begin{aligned}
  (a + b)^2 &= a^2 + 2ab + b^2 \\
  (a - b)^2 &= a^2 - 2ab + b^2
\end{aligned}
$$

For the full list of supported functions and symbols, see the [KaTeX supported functions reference](https://katex.org/docs/supported.html).
