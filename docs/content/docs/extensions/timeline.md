---
title: Timeline
sidebar:
  order: 26
  group: Block
---

Timelines display a vertical sequence of entries with markers and a connecting line. Use them for changelogs, project milestones, or historical sequences.

## Separator syntax

Use `== Title` lines to define entries inside a `:::timeline` block:

````
:::timeline
== September 2024
Course materials published. Students receive access to the learning platform.

== October 2024
First assignments due. Office hours begin.

== December 2024
Final exams and course evaluations.
:::
````

→ A vertical timeline appears with a connecting line. Each entry shows its title in a marker beside the line, with the body content to the right.

<!-- SCREENSHOT: timeline-separator — a three-entry vertical timeline with dates as markers -->

## Heading syntax

`###` headings also work as entry delimiters:

````
:::timeline
### Phase 1: Research
Conduct literature review and define research questions.

### Phase 2: Data collection
Survey participants and gather observational data.

### Phase 3: Analysis
Statistical analysis and interpretation of results.
:::
````

→ The same vertical timeline layout appears, with each `###` heading becoming an entry title.

Both syntaxes can be used interchangeably. The `==` separator and `###` heading styles produce the same visual output.

## Rich content

Each timeline entry supports any Markdown content: paragraphs, lists, code blocks, and nested extensions.

````
:::timeline
== v3.0.0
- Added versioning support
- New search plugin with per-heading indexing
- Responsive image pipeline

== v2.5.0
- Dark mode three-way toggle
- RTL layout support
:::
````

→ Each entry displays its bullet list below the version marker.

## Edge cases

- Content before the first `==` or `###` delimiter creates an implicit entry with no title.
- The `==` separator must have a space after the double equals sign. `==NoSpace` is not recognized as a boundary.
- Markdown formatting in `==` separator body text (after the title line within the same paragraph) is preserved through the inline parser.
- Only `###` (H3) headings are recognized as entry delimiters. `##` and `####` headings render as normal headings inside the current entry.
