---
title: Write Your First Pages
description: "Turn the scaffolded site into your own: write Markdown, add callouts and images, link pages, order the sidebar, group pages into sections, and edit the homepage"
sidebar:
  order: 4
---

Turn the site from [Getting Started](/start-here/getting-started/) into your own. By the end, a docs page has formatted text, a callout, tabs, an image, and a link to another page, the sidebar lists pages in a chosen order with a section of lessons, and the homepage shows your own title.

Start from the `my-site` folder with the dev server running (`sarde dev`) and `http://localhost:4727/docs/my-first-page/` open in the browser. Every step edits a file: save it, and the browser refreshes on its own.

## Write Markdown

Open `content/docs/my-first-page.md`. Replace everything below the closing `---` of the frontmatter with the following Markdown:

```markdown title="content/docs/my-first-page.md"
## Welcome

This page covers the **three stages** of the water cycle:

1. Evaporation
2. Condensation
3. Precipitation

| Stage | Where it happens |
|---|---|
| Evaporation | Oceans, lakes, rivers |
| Condensation | Clouds |
```

→ The page shows a heading, bold text, a numbered list, and a table. The heading also appears in the table of contents beside the page.

Start sections at `##`. Sarde shows the page `title` from the frontmatter as the page heading, so a `#` heading in the body would repeat it. [Markdown Basics](/extensions/markdown-basics/) shows every standard formatting option.

## Add a callout and tabs

Sarde adds its own Markdown syntax, called *extensions*, for content that plain Markdown cannot express. An extension block opens with `:::` and a name, and closes with `:::`. Add the following below the table:

```markdown title="content/docs/my-first-page.md"
:::note
Evaporation is fastest in warm weather.
:::

:::tabs

== Summer

Rain falls as showers.

== Winter

Precipitation often falls as snow.

:::
```

→ The note renders as a colored callout, and the tabs switch between Summer and Winter on click.

Other callout types are `:::tip`, `:::caution`, and `:::danger`. [Using Extensions](/extensions/using-extensions/) lists every extension: cards, steps, file trees, diagrams, math, video, and more.

## Add an image

Copy an image file, for example a PNG or JPEG named `diagram.png`, into the `public/images/` folder of the site. Then add it to the page:

```markdown title="content/docs/my-first-page.md"
![Water cycle diagram](/images/diagram.png)
```

→ The image appears on the page.

Sarde copies everything in `public/` to the built site as is, so `public/images/diagram.png` is served at `/images/diagram.png`. The text in square brackets is the image's alternative text, which screen readers announce and browsers show when the image cannot load. To keep a page's images in the same folder as its Markdown, make the page a *page bundle* instead. [Images and Assets](/guides/images-and-assets/) covers both options.

## Link to another page

Link to the sample page that came with the site, `content/docs/getting-started.md`, by its path inside the docs folder:

```markdown title="content/docs/my-first-page.md"
Read the sample [Getting Started](/getting-started/) page first.
```

→ The link opens the sample page at `/docs/getting-started/`.

A link that starts with `/` is relative to the collection folder, here `content/docs/`. Sarde adds the `/docs/` prefix, and checks that the target page exists. A link to a page that does not exist stops `sarde build` with an error like this one:

```text
[links] checked 2 links: 1 error
  ERROR  content/docs/my-first-page.md:38:5  broken target    /lessons/second-lesson/
Error: build failed: link validation errors found
```

The error names the file, line, and column of the broken link. [Internal Links](/guides/internal-links/) covers relative links, links between collections, and links to headings.

## Order pages in the sidebar

The sidebar lists pages by their `sidebar.order` value, lowest first. The sample Getting Started page sets `order: 1`. Place your page after it by adding `sidebar.order` to the frontmatter:

```yaml title="content/docs/my-first-page.md"
---
date: "2026-03-15T09:00:00-05:00"
title: My First Page
sidebar:
  order: 2
---
```

→ The sidebar lists Getting Started first, then My First Page.

A number at the start of a filename sets the same value without frontmatter: `02-my-first-page.md` gets order 2 and keeps the URL `/docs/my-first-page/`. [Navigation and Sidebar](/guides/navigation-and-sidebar/) covers labels, icons, and collapsed groups.

## Group pages into a section

A folder inside `content/docs/` becomes a collapsible group in the sidebar. Its `_index.md` file is the group's landing page and sets the group's title and position.

Create the folder `content/docs/lessons/` with two files:

```markdown title="content/docs/lessons/_index.md"
---
title: Lessons
sidebar:
  order: 3
---

Each lesson in this section covers one topic.
```

```markdown title="content/docs/lessons/first-lesson.md"
---
title: First Lesson
---

Lesson content.
```

→ The sidebar shows a **Lessons** group after My First Page, with First Lesson inside it. The group title links to `/docs/lessons/`, and the lesson is at `/docs/lessons/first-lesson/`.

`sarde new` creates pages only at the top of a collection, so create files inside a section in the text editor. [Content and Collections](/guides/content-and-collections/) explains sections in more depth.

## Change the site title and homepage

Open `sarde.yaml` at the root of the site. Change the site title and the homepage headline:

```yaml title="sarde.yaml"
site:
  title: "Water Cycle Notes"

homepage:
  hero:
    title: "The Water Cycle"
    subtitle: "Course notes for Earth Science 101."
```

Keep the other keys in the file as they are.

→ The site title in the navigation bar and the browser tab reads Water Cycle Notes, and the homepage at `http://localhost:4727` shows the new headline and subtitle.

The text below the hero comes from `content/_index.md`. [Homepage](/guides/homepage/) lists the homepage templates and every `homepage.hero` key, and [Configuration](/reference/configuration/) documents every setting in `sarde.yaml`.

## Next steps

- [Core Concepts](/start-here/core-concepts/) explains the model behind these steps: URLs, frontmatter, collections, configuration, and themes.
- [Writing Content](/guides/writing-content/) covers frontmatter fields, drafts, scheduled pages, and heading links.
- [Deploying](/start-here/deploying/) publishes the site.
