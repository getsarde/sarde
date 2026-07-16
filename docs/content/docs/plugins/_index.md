---
title: Plugins
description: "Server-side and client-side plugins that extend the build process and generated site"
sidebar:
  order: 4
  icon: puzzle
---

Plugins add behavior to the build process or the generated site. Sarde ships
with 26 built-in plugins.

Read [Using Plugins](/plugins/using-plugins/) for enable/disable configuration,
plugin options, lifecycle hooks, and page injection rules.

**Server-side plugins** run during `sarde build` and produce files or inject
metadata: [Search](/plugins/search), [SEO](/plugins/seo),
[Sitemap](/plugins/sitemap), [Feeds](/plugins/feeds),
[Social Cards](/plugins/social-cards), [Link Validator](/plugins/link-validator),
and [Content Lint](/plugins/content-lint).

**Client-side plugins** inject JavaScript and CSS into the generated pages:
[Reading Progress](/plugins/reading-progress),
[Scroll to Top](/plugins/scroll-to-top),
[Image Lightbox](/plugins/image-lightbox),
[Focus Mode](/plugins/focus-mode),
[External Links](/plugins/external-links), and more.
