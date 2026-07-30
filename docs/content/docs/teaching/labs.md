---
title: Labs
description: "Build step-by-step lab and exercise material with the labs collection: per-lab sidebars, progress indicators, and learning objectives."
sidebar:
  order: 6
---

A directory named `labs` becomes a step-by-step workbook: each lab gets its own sidebar, its own progress bar, and prev/next navigation that stays inside the lab.

```
content/
  labs/
    _index.md
    getting-started/
      _index.md
      01-install.md
      02-build.md
    deploying/
      _index.md
      01-ship.md
```

Detection is by directory name, and `labs` is the only name that works. Unlike the docs collection, there are no synonyms.

## Two structures

Sarde picks the structure by looking at the labs themselves. If any directory under `labs/` has its own subdirectories, the collection is treated as a set of courses.

:::tabs
== Lab and step

Two levels. Each directory under `labs/` is a lab, and the files inside it are the steps.

```
labs/
  getting-started/
    _index.md
    01-install.md
    02-build.md
```

== Course, lab, and step

Three levels. Each directory under `labs/` is a course containing labs.

```
labs/
  web101/
    _index.md
    lab-a/
      _index.md
      01-step.md
    lab-b/
      _index.md
```
:::

Both work with no configuration. Lab numbering restarts inside each course, so `web101` and `web201` both begin at Lab 1.

## Layouts

Landing pages and lab pages use different layouts, which is what makes the collection read as a workbook rather than a flat set of docs.

| Page | Layout | Renders as |
|------|--------|------------|
| Collection root (`labs/_index.md`) | `default` | Card grid of courses or labs |
| Course index (three-level only) | `default` | Card grid of labs |
| Lab index (`_index.md`) | `labs` | Lab intro with sidebar and progress |
| Step page | `labs` | Step with sidebar and progress |

Set `layout` in a page's frontmatter to override any of these.

Cards on a landing page show a count taken from what the directory holds: `2 labs` for a course, `1 steps` for a lab. Add `image` to a lab's `_index.md` for a thumbnail, and `description` for the card text.

## Per-lab sidebar

Inside a lab, the sidebar lists only that lab. Other labs in the collection do not appear. The first entry is always **Overview**, linking to the lab's `_index.md`, followed by the steps in `sidebar.order`.

Prev/next stops at the lab boundary. The last step of one lab has no Next link into the following lab, so a reader finishing a lab returns to the landing page rather than falling into unrelated material.

## Progress and lab number

Every page in a lab renders two indicators, both filled in automatically:

- A **lab badge** above the title reading `Lab 1`, numbered by the lab's position among its siblings.
- A **progress bar** reading `Step 2 of 3`.

The step count includes the Overview page. A lab with an `_index.md` and two steps reports three steps, and the Overview is step 1.

To rename the badge, set a label on the collection:

```yaml
collections:
  labs:
    labs:
      label: "Exercise"
```

Pages then read `Exercise 1`. The word is used for the badge only; the progress bar always says "Step".

## Learning objectives

Add `learning_objectives` to a lab's frontmatter to render a highlighted list at the top of the page:

```yaml
---
title: Getting Started
learning_objectives:
  - Install the toolchain
  - Run the first build
---
```

The field takes a list of strings and works on any page in the collection, though it usually belongs on the lab's `_index.md`.

## Defaults

The labs collection is configured on detection. Override any of it under `collections.labs` in `sarde.yaml`.

| Setting | Default |
|---------|---------|
| Sort | `sidebar.order`, ascending |
| Sidebar | Collapsible, max depth 4, search enabled |
| Table of contents | Levels 2 to 4, scroll highlighting on |
| Prev/next | Enabled, labelled "Previous" and "Next" |
| Lab badge label | `Lab` |
