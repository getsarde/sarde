---
title: Using Extensions
description: "How block and inline extension syntax works, including fenced directives and nesting rules"
sidebar:
  order: 1
---

Extensions add custom syntax to Markdown. Sarde includes block extensions (asides, tabs, steps, cards, and more) and inline extensions (icons, badges, keyboard shortcuts). All extensions are built into the binary and active by default.

## Block syntax

Block extensions use a `:::` fenced directive. The directive name follows the opening colons:

````
:::note
Mitochondria are the powerhouse of the cell.
:::
````

:::note
Mitochondria are the powerhouse of the cell.
:::

→ A blue callout box appears with a "Note" label and an info icon.

Some block extensions wrap child content with a specific structure, like tabs wrapping individual tab panels:

````
::::tabs
:::tab[Biology]
Photosynthesis converts light energy into chemical energy.
:::
:::tab[Chemistry]
The reaction produces glucose and oxygen from water and CO₂.
:::
::::
````

Use four or more colons (`::::`) for the outer fence when nesting extensions inside each other.

## Inline syntax

Inline extensions use a `:name[content]` pattern embedded directly in paragraph text:

```markdown
Press :kbd[Ctrl+S] to save the file.
```

→ The key combination renders as styled keyboard keys inline with the text.

Some inline extensions accept attributes after the content:

```markdown
:badge[New]{type="success"}
```

## Attributes

Block extensions accept parameters in square brackets or as space-separated key-value pairs on the opening fence:

```markdown
:::aside[Custom Title] icon=sparkles
Content here.
:::
```

Inline extensions accept attributes in curly braces after the closing bracket:

```markdown
:badge[Beta]{type="warning" size="sm"}
```

Attribute values use `key="value"` or `key='value'` syntax. Bare flags (no value) are also supported for boolean options:

```markdown
:::details[Click to expand] open
Hidden content revealed on page load.
:::
```

## Nesting

Block extensions can be nested. Use additional colons on the outer fence to distinguish it from the inner fence:

````
::::card-grid
:::card[Lesson 1]
Introduction to cellular biology.
:::
:::card[Lesson 2]
DNA replication and protein synthesis.
:::
::::
````

The parser tracks nesting depth internally. Mismatched closing fences are ignored until the correct depth is reached. Named closing fences (`:::/aside`) are also supported for clarity in deeply nested structures.

## Standard Goldmark extensions

In addition to Sarde's custom extensions, the Markdown renderer includes these standard Goldmark extensions:

| Extension | Syntax | Description |
|-----------|--------|-------------|
| GFM | Tables, task lists, strikethrough, autolinks | GitHub Flavored Markdown features. |
| Footnotes | `[^1]` and `[^1]: text` | Footnote references and definitions. |
| Definition Lists | `Term` followed by `: Definition` | HTML `<dl>` definition lists. |

These are always enabled and require no configuration.

## Extension list

Each extension is documented on its own page. Block extensions handle multi-line containers. Inline extensions handle text-level markers.

**Block extensions:** [Aside](/extensions/aside), [Accordion](/extensions/accordion), [Badges](/extensions/badges) (group), [Cards](/extensions/cards), Code Group, Columns, [Details](/extensions/details), [Figure](/extensions/figure), [File Tree](/extensions/file-tree), [Gallery](/extensions/gallery), [Image Compare](/extensions/image-compare), [Link Buttons](/extensions/link-buttons) (group), [Link Card](/extensions/link-card), [Math](/extensions/math) (display), [Mermaid](/extensions/mermaid), [Steps](/extensions/steps), [Tabs](/extensions/tabs), [Terminal](/extensions/terminal), [Timeline](/extensions/timeline), [Video](/extensions/video).

**Inline extensions:** [Annotation](/extensions/annotation), Badge, [Copy Text](/extensions/copy-text), [Highlight](/extensions/highlight), [Icon](/extensions/icon), [Kbd](/extensions/kbd), [Spoiler](/extensions/spoiler).
