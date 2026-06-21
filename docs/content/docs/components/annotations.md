---
title: "Annotations"
description: "Inline tooltips that reveal explanations on hover."
sidebar:
  order: 15
---

Annotations add a dotted underline to a word or phrase. Hovering over the text reveals an explanation in a tooltip.

## Basic Usage

```md
::annotation[Label]{Explanation text shown in tooltip}
```

The text in square brackets appears inline with a dotted underline. The text in curly braces appears in the tooltip.

## Example

```md
Sarde uses ::annotation[Goldmark]{A CommonMark-compliant Markdown parser written in Go} to process Markdown files.
```

Renders as: Sarde uses ::annotation[Goldmark]{A CommonMark-compliant Markdown parser written in Go} to process Markdown files.

## Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| Label | string | The text displayed inline with a dotted underline (inside `[]`) |
| Explanation | string | The tooltip text shown on hover (inside `{}`) |
