---
title: Extensions
description: "Built-in Markdown block and inline extensions for callouts, tabs, cards, media, and more"
sidebar:
  order: 4
  icon: sparkles
---

Extensions add syntax to Markdown for content that plain Markdown cannot express, such as callouts, tabs, and embedded video. Every extension is built into the `sarde` binary and active by default.

Block extensions wrap content in a `:::name` fence. Inline extensions sit inside a sentence, as in `::kbd[Ctrl+S]`. [Using Extensions](/extensions/using-extensions/) covers the shared syntax, attributes, and nesting rules. To add `:::` blocks of your own, see [Custom Directives](/extensions/custom-directives/). The [Kitchen Sink](/extensions/kitchen-sink/) page shows every extension rendered on one page.

Each table below groups extensions by what the reader sees.

## Callouts and status

| Extension | Use it to |
|-----------|-----------|
| [Aside](/extensions/aside) | Set a note, tip, warning, or danger apart from the surrounding text. |
| [Badges](/extensions/badges) | Label a status, version, or category with a colored pill. |
| [Highlight](/extensions/highlight) | Mark a word or phrase with a colored background. |
| [Annotation](/extensions/annotation) | Attach a tooltip to a term without breaking the sentence. |
| [Spoiler](/extensions/spoiler) | Blur text until the reader reveals it. |

## Layout and structure

| Extension | Use it to |
|-----------|-----------|
| [Cards](/extensions/cards) | Put content in a bordered container with a title and icon. |
| [Card Grid](/extensions/card-grid) | Arrange cards in a responsive grid. |
| [Columns](/extensions/columns) | Place content side by side. |
| [Tabs](/extensions/tabs) | Switch between related panels, with the selection synced across the site. |
| [Steps](/extensions/steps) | Number a sequence of instructions. |
| [Timeline](/extensions/timeline) | List dated entries such as milestones or releases. |

## Collapsible content

| Extension | Use it to |
|-----------|-----------|
| [Details](/extensions/details) | Hide content behind a clickable summary. |
| [Accordion](/extensions/accordion) | Group details so that opening one closes the others. |

## Links and actions

| Extension | Use it to |
|-----------|-----------|
| [Link Buttons](/extensions/link-buttons) | Show a link as a button, alone or in a group. |
| [Link Card](/extensions/link-card) | Show a link as a card with a title, description, and image. |
| [Copy Text](/extensions/copy-text) | Let the reader copy a value with one click. |

## Images and media

| Extension | Use it to |
|-----------|-----------|
| [Figure](/extensions/figure) | Add a caption below an image. |
| [Gallery](/extensions/gallery) | Show images in a grid with a full-screen lightbox. |
| [Image Compare](/extensions/image-compare) | Compare two images with a draggable slider. |
| [Video](/extensions/video) | Embed YouTube, Vimeo, or self-hosted video. |
| [Icon](/extensions/icon) | Place an SVG icon inside a sentence. |

## Technical content

| Extension | Use it to |
|-----------|-----------|
| [File Tree](/extensions/file-tree) | Draw a directory structure. |
| [Terminal](/extensions/terminal) | Show commands and their output in a terminal window. |
| [Kbd](/extensions/kbd) | Show keys and key combinations as keycaps. |
| [Math](/extensions/math) | Typeset LaTeX formulas. |
| [Mermaid](/extensions/mermaid) | Draw diagrams from text definitions. |

Standard Markdown, including tables, task lists, and footnotes, is covered in [Markdown Basics](/extensions/markdown-basics/). Code block features are covered in the [Code Blocks](/guides/code-blocks/) guide.
