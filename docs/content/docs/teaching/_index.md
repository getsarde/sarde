---
title: Teaching
description: "Build slide decks, courses, labs, and presentations with Sarde"
aliases:
  - /guides/slides-and-presentations/
sidebar:
  order: 3
  icon: graduation-cap
---

Build slide decks, labs, and course material from Markdown. Sarde renders presentations
as full-screen slide viewers, organizes decks into SlideShare-style galleries, and turns a
`labs` directory into a step-by-step workbook.

- [Course Template](/teaching/course-template/) starts a complete course website:
  courses with schedules and announcements, labs, a course catalog, and GitHub Pages
  publishing.
- [Slides Collection](/teaching/slides-collection/) covers the auto-detected
  `slides` collection type, gallery pages, and multi-course organization.
- [Presentation Layout](/teaching/presentation-layout/) explains the
  full-viewport layout, per-page overrides, cascade, and dual-output workflows.
- [Writing Slides](/teaching/writing-slides/) covers slide separation syntax,
  authoring tips, and writing content that works as both docs and slides.
- [Viewer Features](/teaching/viewer-features/) documents navigation, themes,
  search, bookmarks, laser pointer, and keyboard shortcuts.
- [Courses with Slides](/teaching/courses-with-slides/) shows how to embed
  presentations inside docs or courses collections with sidebar integration.
- [Labs](/teaching/labs/) covers the `labs` collection type, per-lab sidebars,
  step progress, and learning objectives.

## Start from the course template

To start with a working course site rather than an empty project, create it from the course template. The command downloads the template from the [sarde-templates](https://github.com/getsarde/sarde-templates) repository, so it needs a network connection the first time:

```bash
sarde new site my-college --template course
```

[Course Template](/teaching/course-template/) walks through what the site contains, how to replace the sample courses and labs, and how to publish it on GitHub Pages.
