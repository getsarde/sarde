---
title: Course Template
description: "Start a course website from the course template: courses with schedules and announcements, hands-on labs, a course catalog, and publishing on GitHub Pages."
sidebar:
  order: 0
---

The course template is a complete course website to copy and fill with your own material. It has two sample courses with lessons, assignments, schedules, and announcements, hands-on labs grouped by course, a course catalog, and a GitHub Actions workflow that publishes the site to GitHub Pages. See the [live demo](https://getsarde.github.io/course-template/).

There are two ways to start from it, and both produce the same files:

- Run `sarde new site` with `--template course`.
- Clone the [`getsarde/course-template`](https://github.com/getsarde/course-template) repository, which also has a README and a license.

## Create a course site

1. Create the site and move into it:

   ```bash
   sarde new site my-college --template course
   cd my-college
   ```

   → The command prints the next step and explains how to turn on the GitHub Pages workflow. See [Publish on GitHub Pages](#publish-on-github-pages).

2. Start the dev server:

   ```bash
   sarde dev
   ```

   Open `http://localhost:4727`. The site rebuilds and the browser reloads each time you save a file.

To start from the repository instead, clone it and run the dev server inside it:

```bash
git clone https://github.com/getsarde/course-template.git my-college
cd my-college
sarde dev
```

## What's in the template

```
my-college/
  sarde.yaml                   # site title, homepage hero, header links, plugins
  kazari.config.yaml           # code block options
  .github/
    workflows/
      deploy.yml               # publishes the site to GitHub Pages
  content/
    _index.md                  # homepage, with a link to each course
    announcements.md           # news for every course
    courses/
      _index.md                # course catalog at /courses/
      web-fundamentals/
        _index.md              # course overview and Course info card
        announcements.md       # news for this course
        schedule.md            # weekly plan
        html-basics.md         # lesson
        css-layout.md          # lesson
        assignments/
          _index.md
          build-a-page.md
          style-a-card.md
      python-essentials/       # a second course with the same shape
    labs/
      _index.md                # one card per course
      web-fundamentals/        # the labs for one course
      python-essentials/
  public/
    images/                    # homepage hero images
```

Sarde detects `courses` as a docs collection and `labs` as the labs collection from their directory names, so neither needs configuration.

## Courses

Each directory in `content/courses/` is one course. With two or more courses, each one becomes a tab in the course switcher at the top of the sidebar, and the sidebar shows only the current course's pages. [Tabbed Navigation](/guides/tabbed-navigation/) explains how tabs work.

With a single course, tabs turn off, so the course switcher and the catalog disappear. To keep them for one course, turn tabs on in `sarde.yaml`:

`sarde.yaml`
```yaml
collections:
  courses:
    tabs: true
```

### Course overview

A course's `_index.md` is its overview page, listed as **Overview** at the top of the course sidebar. Its fields also appear outside the page:

| Field | Shown in |
|-------|----------|
| `title` | Course switcher and catalog card |
| `description` | Course switcher menu, catalog card, and under the page title |
| `icon` | Tile in the switcher and on the catalog card, in place of the course's initials |
| `sidebar.badge` | Catalog card, for example **Beginner** |

The overview ends with a **Course info** card for the instructor's contact details and office hours. It uses the [card](/extensions/cards/) extension:

```markdown
:::card[Course info](icon="user")
- **Instructor:** Your Name, [you@example.edu](mailto:you@example.edu)
- **Office hours:** Mondays 12:00 to 13:30 and Fridays 12:00 to 13:00, online
:::
```

### Pages in a course

The course sidebar lists **Overview** first, then the course's pages in `sidebar.order`:

| Order | Page |
|-------|------|
| 1 | `announcements.md` |
| 2 | `schedule.md` |
| 3, 4 | The lessons, `html-basics.md` and `css-layout.md` |
| 5 | The `assignments/` group |

Give each new lesson a `sidebar.order` to place it. `assignments/` is a sidebar group with its own `_index.md`, which sets the **Assignment** badge on the group. The assignments inside it are ordered the same way.

### The course catalog

`content/courses/_index.md` is the catalog at `/courses/`, where the header's **Courses** link and the homepage's **Browse Courses** button lead. Its text appears above one card per course, in tab order. Each card shows the course's tile, description, and badge.

The catalog exists because that `_index.md` has a body. With frontmatter only, `/courses/` redirects to the first course instead. See [The collection root](/guides/tabbed-navigation/#the-collection-root).

## Schedules

Each course's `schedule.md` is a [timeline](/extensions/timeline/) with one entry per week:

- **Entry title:** the week and its topic.
- **Heading:** a guiding question in bold on the first line, which the timeline styles as the entry's heading.
- **Items:** a list where each item opens with an icon for its type.

`content/courses/web-fundamentals/schedule.md`
```markdown
:icon[book-open] Lesson · :icon[flask-conical] Lab · :icon[pencil] Assignment · :icon[users] In class

:::timeline
== Week 1: HTML structure
**How is a web page put together?**

- :icon[calendar] Sep 7 to 13
- :icon[book-open] [HTML Basics](/courses/web-fundamentals/html-basics/)
- :icon[flask-conical] [Hello World](/labs/web-fundamentals/hello-world/)
- :icon[users] Course kickoff and setup check
:::
```

→ The first line is the legend. In each entry, the icons replace the list bullets. See [Icon lists](/extensions/timeline/#icon-lists).

### Long courses

A timeline is always open, which reads well for a few weeks. For a full semester, a list of collapsible weeks keeps the page short. Put each week in a [`:::details`](/extensions/details/) block titled with its topic, inside an [`:::accordion(independent)`](/extensions/accordion/), and add `open` to the current week:

```markdown
:::accordion(independent)
:::details[Week 1: HTML structure]
- :icon[calendar] Sep 7 to 13
- :icon[book-open] [HTML Basics](/courses/web-fundamentals/html-basics/)
:::
:::details[Week 2: CSS layout] open
- :icon[calendar] Sep 14 to 20
- :icon[book-open] [CSS Layout](/courses/web-fundamentals/css-layout/)
:::
:::
```

Icon lists work inside details panels too.

## Announcements

The template posts announcements in three places:

- **Course announcements:** each course's `announcements.md` holds its news, newest first, one `###` heading per item.
- **Site-wide announcements:** `content/announcements.md` holds news for every course and links to each course's announcements and schedule. The header's **Announcements** link opens it.
- **Banner:** a one-line strip above the header, from the [announcements plugin](/plugins/announcements/).

The sample banner appears only on one course's lessons and labs, through `show_on`:

`sarde.yaml`
```yaml
plugins:
  config:
    announcements:
      items:
        - id: web-fundamentals-assignment-1
          message: "Web Fundamentals: Assignment 1 is due at the end of week 2."
          type: info
          show_on:
            - /courses/web-fundamentals/**
            - /labs/web-fundamentals/**
```

The template's `sarde.yaml` also lists every default plugin under `plugins.enabled` before `announcements` and `telescope`, because a `plugins.enabled` list replaces the default set rather than adding to it. [Placement](/plugins/announcements/#placement) explains how the banner strip behaves.

## Labs

`content/labs/` groups the labs by course, with one directory per course and one directory per lab inside it. Lab numbers restart in each course. `web-fundamentals/hello-world/` is a single-page lab: it has only an `_index.md`. [Labs](/teaching/labs/) covers the structure, step order, progress bars, and learning objectives.

## Homepage and navigation

- **Hero:** `homepage.hero` in `sarde.yaml` sets the homepage title, subtitle, images, and two buttons, **Browse Courses** (`/courses`) and **View Labs** (`/labs`).
- **Header links:** the header lists the **Courses** and **Labs** collections, followed by the `header.links` entries for **Announcements** and **Tags**.
- **Homepage content:** `content/_index.md` links to each course with a card.
- **Quick navigation:** the [Telescope plugin](/plugins/telescope/) is on, so ::kbd[Ctrl]+::kbd[/] (::kbd[Cmd]+::kbd[/] on Mac) opens a quick search that jumps to any page by name.

## Replace the sample content

1. In `sarde.yaml`, set `site.title`, `site.description`, and `site.url`, then change the hero text and buttons under `homepage.hero`.
2. Replace the sample courses in `content/courses/`, and update the catalog text in `content/courses/_index.md`.
3. Replace the sample labs in `content/labs/`.
4. Update `content/announcements.md`, each course's `announcements.md` and `schedule.md`, and the banners under `plugins.config.announcements` in `sarde.yaml`.
5. Update the course links in `content/_index.md`.
6. Search the `content/` directory for the word "Replace". Most sample pages open with a sentence that starts with it, which marks text you have not replaced yet.

The sample lessons and labs also show Markdown extensions and code-block options in context. [Using Extensions](/extensions/using-extensions/) and [Code Blocks](/guides/code-blocks/) document each one.

## Publish on GitHub Pages

`.github/workflows/deploy.yml` builds the site and publishes it to GitHub Pages on every push to `main`.

1. Push the site to a GitHub repository.
2. In the repository, open **Settings > Pages** and set **Source** to **GitHub Actions**.
3. Push to `main`, or run the workflow from the **Actions** tab.

→ The site is published at `https://<owner>.github.io/<repository>/`.

The workflow reads the site's address and base path from GitHub Pages and passes them to Sarde, so `sarde.yaml` needs no deployment settings, and a custom domain set under **Settings > Pages** works the same way. Until Pages uses GitHub Actions as its source, the workflow fails on every push. If you host the site elsewhere, delete the workflow file. [GitHub Pages](/deployment/github-pages/) and [Deploying](/start-here/deploying/) cover other options.
