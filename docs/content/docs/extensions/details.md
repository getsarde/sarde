---
title: Details
sidebar:
  order: 14
  group: Block
---

Details creates a collapsible section that readers can expand and collapse. It renders as an HTML `<details>` element with a clickable summary.

## Basic syntax

````
:::details[What are mitochondria?]
Mitochondria are membrane-bound organelles found in eukaryotic cells. They generate most of the cell's supply of adenosine triphosphate (ATP), the molecule used as energy currency.
:::
````

:::details[What are mitochondria?]
Mitochondria are membrane-bound organelles found in eukaryotic cells. They generate most of the cell's supply of adenosine triphosphate (ATP), the molecule used as energy currency.
:::

→ A collapsed section appears with "What are mitochondria?" as the clickable summary. Clicking it reveals the content below.

## Open by default

Add the `open` flag to expand the section on page load:

````
:::details[Course syllabus] open
- Week 1: Introduction to molecular biology
- Week 2: DNA structure and replication
- Week 3: Gene expression and regulation
- Week 4: Genetic engineering techniques
:::
````

:::details[Course syllabus] open
- Week 1: Introduction to molecular biology
- Week 2: DNA structure and replication
- Week 3: Gene expression and regulation
- Week 4: Genetic engineering techniques
:::

→ The section appears expanded when the page loads. Readers can still collapse it by clicking the summary.

The `open` flag can also be written in parentheses: `:::details(open)[Course syllabus]`.

## Default summary

Omitting the summary text produces a default label of "Details":

````
:::details
Additional technical notes that most readers can skip.
:::
````

:::details
Additional technical notes that most readers can skip.
:::

→ A collapsed section appears with "Details" as the summary text.

## Rich content

Details sections support any Markdown content, including headings, code blocks, lists, and nested extensions:

````
:::details[Implementation notes]
The authentication flow uses three steps:

1. Client sends credentials to `/auth/login`
2. Server validates and returns a JWT
3. Client includes the token in subsequent requests

```python
response = requests.post("/auth/login", json={"user": "admin", "pass": "secret"})
token = response.json()["token"]
```
:::
````

→ The collapsed section contains formatted text, a numbered list, and a syntax-highlighted code block.

## Using with accordion

Wrap multiple `:::details` blocks in an `:::accordion` container to create mutually exclusive panels where opening one closes the others. See the Accordion extension page for details.

## Options

| Option | Syntax | Default | Description |
|--------|--------|---------|-------------|
| Summary | `:::details[Text]` | `"Details"` | Clickable summary text shown in the collapsed state. |
| `open` | `:::details[Text] open` or `:::details(open)[Text]` | `false` | Expand the section on page load. |

## Edge cases

- The `open` flag is accepted in parentheses before or after the summary brackets, or as a trailing bare word. All three forms are equivalent.
- Details blocks cannot interrupt a paragraph. A blank line must precede the opening `:::`.
- The native `<details>` element handles expand/collapse without JavaScript. The extension works even with scripts disabled.
- Nested `:::details` blocks inside a details section require four colons (`::::`) on the outer fence.
