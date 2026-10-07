---
title: Using Extensions
description: "How block and inline extension syntax works, including fenced directives and nesting rules"
sidebar:
  order: 1
---

Extensions add custom syntax to Markdown. Sarde includes block extensions (asides, tabs, steps, cards, and more) and inline extensions (icons, keyboard shortcuts, annotations). All extensions are built into the binary and active by default.

## Block syntax

Block extensions use a `:::` fenced directive. The directive name follows the opening colons:

::::example
:::note
Mitochondria are the powerhouse of the cell.
:::
::::

→ A blue callout box appears with a "Note" label and an info icon.

Some block extensions divide their content into sections with marker lines rather than nested fences. Tabs use `== Label` to start each panel, and a panel can contain other block extensions:

````
:::tabs
== Biology

:::note
Photosynthesis converts light energy into chemical energy.
:::

== Chemistry

The reaction produces glucose and oxygen from water and CO₂.
:::/tabs
````

When a block sits inside another block, close the outer block with a named fence such as `:::/tabs`. See [Nesting](#nesting).

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

Block extensions nest. Every block opens with `:::name` and closes with a bare `:::` or with a named fence, `:::/name`. Use the named fence on the outer block when nesting: it states which block it closes, so the inner blocks can use plain `:::`.

````
:::card-grid
:::card[Lesson 1]
Introduction to cellular biology.
:::
:::card[Lesson 2]
DNA replication and protein synthesis.
:::
:::/card-grid
````

A named fence closes the nearest open block with that name. If an inner block was left open, `:::/card-grid` still closes the grid, and the inner block closes with it. A named fence that matches no open block, such as `:::/tip` inside a `:::note`, renders as text. Names are case-insensitive. `:::/aside` closes an aside of any type, and `:::/filetree` is accepted for `:::file-tree`.

Blocks of the same name nest too. Each `:::/details` closes the innermost open details block:

````
:::details[Outer]
:::details[Inner]
Hidden twice.
:::/details
:::/details
````

A bare `:::` closes the innermost open block. Longer fences (`::::`, `:::::`) are accepted, so content written for Docusaurus, VitePress, or Pandoc keeps working. The extra colons are a readability convention, not a nesting level.

Run `sarde check-syntax` to find unclosed, mismatched, or malformed fences before building. `sarde dev --check-syntax` runs the same check on every rebuild.

### Code groups

Kazari's `:::code-group` is the one exception. It closes only with a bare `:::` and does not track nesting, so close a code group with `:::` before the closing fence of the block that contains it. `:::/code-group` is not recognized.

## Standard Goldmark extensions

In addition to Sarde's custom extensions, the Markdown renderer includes these standard Goldmark extensions:

| Extension | Syntax | Description |
|-----------|--------|-------------|
| GFM | Tables, task lists, strikethrough, autolinks | GitHub Flavored Markdown features. |
| Footnotes | `[^1]` and `[^1]: text` | Footnote references and definitions. |
| Definition Lists | `Term` followed by `: Definition` | HTML `<dl>` definition lists. |

These are always enabled and require no configuration.

## Extension list

Each extension has its own page. The [Extensions](/extensions/) index groups them by purpose.

## Custom directives

Add custom `:::` directives without writing Go: drop a YAML schema, an HTML template, and optional CSS into a `directives/` folder at your site root. See [Custom Directives](/extensions/custom-directives/).
