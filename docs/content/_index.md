---
title: "Sarde"
description: "A zero-config static site generator. One binary. Markdown in, website out."
layout: splash
hero:
  title: "Sarde"
  tagline: "Zero config. One binary. Markdown in, website out."
  actions:
    - text: "Get Started"
      link: /docs/getting-started/
      variant: primary
    - text: "GitHub"
      link: https://github.com/frostybee/sarde
      variant: secondary
      attrs:
        target: _blank
        rel: noopener noreferrer
  image:
    light: /images/hero-light.svg
    dark: /images/hero-dark.svg
    alt: "Sarde architecture diagram"
  background: gradient
---

:::card-grid

:::card{title="Zero Config" icon="zap"}
Drop Markdown files into `content/` and run `sarde dev`. Collections, navigation, and theming are inferred from directory names.
:::

:::card{title="Rich Markdown" icon="puzzle"}
26 extensions — callouts, tabs, code groups, steps, file trees, galleries, math, diagrams — all rendered without client JavaScript.
:::

:::card{title="Instant Search" icon="search"}
Offline-first Orama search. The index is generated at build time with zero external services.
:::

:::card{title="Production Ready" icon="rocket"}
30 built-in plugins handle sitemap, feeds, SEO, social cards, link validation, and content linting out of the box.
:::

:::
