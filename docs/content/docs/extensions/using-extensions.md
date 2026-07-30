---
title: Using Extensions
description: "How block and inline extension syntax works, including fenced directives and nesting rules"
sidebar:
  order: 1
---

Extensions add custom syntax to Markdown. Sarde includes block extensions (asides, tabs, steps, cards, and more) and inline extensions (icons, keyboard shortcuts, annotations). All extensions are built into the binary and active by default.

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

Some block extensions divide their content into sections with marker lines rather than nested fences. Tabs use `== Label` to start each panel, and a panel can contain other block extensions:

````
::::tabs
== Biology

:::note
Photosynthesis converts light energy into chemical energy.
:::

== Chemistry

The reaction produces glucose and oxygen from water and CO₂.
::::
````

Use four or more colons (`::::`) for the outer fence when nesting extensions inside each other.

## Inline syntax

Inline extensions embed directly in paragraph text. Most use a `::name[content]` pattern:

```markdown
Press ::kbd[Ctrl+S] to save the file.
```

→ The key combination renders as styled keyboard keys inline with the text.

The prefix is not the same for every inline extension. Use the form shown here:

| Extension | Syntax |
|-----------|--------|
| [Icon](/extensions/icon/) | `:icon[settings]` (one colon) |
| [Kbd](/extensions/kbd/) | `::kbd[Ctrl+S]` |
| [Annotation](/extensions/annotation/) | `::annotation[term]` |
| [Copy text](/extensions/copy-text/) | `::copy[npm install]` |
| [Highlight](/extensions/highlight/) | `==marked text==` |
| [Spoiler](/extensions/spoiler/) | `\|\|hidden text\|\|` |

## Attributes

Block extensions accept parameters in square brackets, in parentheses, or as space-separated key-value pairs on the opening fence:

```markdown
:::aside[Custom Title] icon=sparkles
Content here.
:::
```

```markdown
:::badge(success icon="rocket")
Shipped
:::
```

Inline extensions that take attributes accept them in parentheses after the closing bracket:

```markdown
::kbd[Esc](size="lg" wide)
```

Icon is the exception: its attributes go inside the brackets, after the icon name.

```markdown
:icon[heart class="sarde-icon-sm"]
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

**Block extensions:** [Aside](/extensions/aside), [Accordion](/extensions/accordion), [Badges](/extensions/badges) (group), [Cards](/extensions/cards), Code Group, [Columns](/extensions/columns), [Details](/extensions/details), [Figure](/extensions/figure), [File Tree](/extensions/file-tree), [Gallery](/extensions/gallery), [Image Compare](/extensions/image-compare), [Link Buttons](/extensions/link-buttons) (group), [Link Card](/extensions/link-card), [Math](/extensions/math) (display), [Mermaid](/extensions/mermaid), [Steps](/extensions/steps), [Tabs](/extensions/tabs), [Terminal](/extensions/terminal), [Timeline](/extensions/timeline), [Video](/extensions/video).

**Inline extensions:** [Annotation](/extensions/annotation), [Copy Text](/extensions/copy-text), [Highlight](/extensions/highlight), [Icon](/extensions/icon), [Kbd](/extensions/kbd), [Spoiler](/extensions/spoiler).
