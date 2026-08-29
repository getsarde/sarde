---
title: Steps
description: "Display a numbered sequence of instructions with a connecting vertical progress line"
sidebar:
  order: 23
---

Steps display a numbered sequence of instructions with a vertical progress line connecting each step. Two input modes are supported: heading mode and ordered-list mode.

## Heading mode

Use `##` or `###` headings inside a `:::steps` block. Each heading becomes a numbered step:

````
:::steps
## Install dependencies

Run the following command to install all required packages:

```bash
npm install
```

## Configure the project

Create a `sarde.yaml` file in the project root with the site title and theme preset.

## Start the dev server

Run `sarde dev` and open `http://localhost:4727` in a browser.
:::
````

→ Three numbered steps appear with a vertical line connecting them. Each step shows its number, title, and body content below.

<!-- SCREENSHOT: steps-heading-mode — three numbered steps with vertical progress line -->

## Ordered-list mode

Write a standard Markdown ordered list inside `:::steps`. CSS handles the numbering and step styling:

````
:::steps
1. Clone the repository from GitHub.
2. Install the required dependencies with `npm install`.
3. Run `sarde dev` to start the local development server.
:::
````

:::steps
1. Clone the repository from GitHub.
2. Install the required dependencies with `npm install`.
3. Run `sarde dev` to start the local development server.
:::

→ The ordered list renders as styled steps with numbering and a connecting line.

The extension auto-detects which mode to use. If the block contains `##` or `###` headings, it uses heading mode. If no headings are found, the block renders its content directly (ordered-list mode relies on CSS for step styling).

## Heading level

The first `##` or `###` heading found determines the step delimiter level. All subsequent headings at that level start a new step. Headings at other levels render as normal headings within the current step's body.

````
:::steps
### Create the project

Run `sarde new site my-course`.

### Add course content

Create Markdown files in `content/docs/`:

#### Lesson structure

Each lesson is a single `.md` file with frontmatter.

### Build and deploy

Run `sarde build` to generate the production output.
:::
````

→ Three steps appear (delimited by `###`). The `#### Lesson structure` heading renders as a sub-heading inside step 2.

## Rich content

Each step can contain any Markdown content: paragraphs, code blocks, images, lists, and nested extensions.

## Edge cases

- Content before the first heading in heading mode creates an implicit step with no title.
- In ordered-list mode, the `:::steps` container adds the visual step styling. The list items themselves are standard Goldmark ordered-list nodes.
- Each step includes a screen-reader-only label ("Step 1.", "Step 2.", etc.) for accessibility.
- Steps cannot be nested inside other steps.
