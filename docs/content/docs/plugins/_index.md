---
title: Plugins
description: "Server-side and client-side plugins that extend the build process and generated site"
sidebar:
  order: 5
  icon: puzzle
---

Plugins add behavior to the build process or the generated site. Sarde ships
with 27 built-in plugins, and sites can install external plugins into their
`plugins/` directory.

Read [Using Plugins](/plugins/using-plugins/) for enable/disable configuration,
plugin options, lifecycle hooks, and page injection rules. Read
[External Plugins](/plugins/external-plugins/) for installing and managing
plugins distributed outside the Sarde binary, and
[Premium Plugins](/plugins/premium-plugins/) for licensed ones.

**Server-side plugins** run during `sarde build` and produce files or inject
metadata: [Search](/plugins/search), [SEO](/plugins/seo),
[Sitemap](/plugins/sitemap), [Feeds](/plugins/feeds),
[Social Cards](/plugins/social-cards), [SlideViewer](/plugins/slideviewer),
[Link Validator](/plugins/link-validator),
and [Content Lint](/plugins/content-lint).

**Client-side plugins** inject JavaScript and CSS into the generated pages:
[Reading Progress](/plugins/reading-progress),
[Scroll to Top](/plugins/scroll-to-top),
[Image Lightbox](/plugins/image-lightbox),
[Focus Mode](/plugins/focus-mode),
[External Links](/plugins/external-links), and more.
