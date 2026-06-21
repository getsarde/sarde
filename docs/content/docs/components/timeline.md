---
title: "Timeline"
description: "Vertical timelines with date badges and content cards."
sidebar:
  order: 26
---

Timelines display events in a vertical layout with dot markers and optional date badges. Entry boundaries use `== Entry Title` lines — the same syntax as [Tabs](/docs/components/tabs/).

## Basic Usage

```md
:::timeline

== v3.0.0 (2025-06)
Rewrote the rendering pipeline. Build times dropped by 60%. Added the plugin system with four lifecycle hooks.

== v2.0.0 (2024-11)
Introduced i18n support with localized URLs. Added versioned documentation collections.

== v1.1.0 (2024-06)
Added Kazari code blocks with syntax highlighting, line numbers, and diff markers.

== v1.0.0 (2024-01)
Initial release. Markdown to static site with zero configuration.

:::
```

:::timeline

== v3.0.0 (2025-06)
Rewrote the rendering pipeline. Build times dropped by 60%. Added the plugin system with four lifecycle hooks.

== v2.0.0 (2024-11)
Introduced i18n support with localized URLs. Added versioned documentation collections.

== v1.1.0 (2024-06)
Added Kazari code blocks with syntax highlighting, line numbers, and diff markers.

== v1.0.0 (2024-01)
Initial release. Markdown to static site with zero configuration.

:::

## Heading Mode

Use `###` headings as entry boundaries instead of `==`:

```md
:::timeline

### Project kickoff

Defined the architecture. Chose Go for the engine and Tauri for the desktop app.

### First prototype

Rendered a 200-page docs site in under one second.

### Public beta

Released to early adopters. Collected feedback on the plugin API.

:::
```

:::timeline

### Project kickoff

Defined the architecture. Chose Go for the engine and Tauri for the desktop app.

### First prototype

Rendered a 200-page docs site in under one second.

### Public beta

Released to early adopters. Collected feedback on the plugin API.

:::

Entry bodies accept full Markdown — paragraphs, lists, code blocks, and inline formatting all render normally.
