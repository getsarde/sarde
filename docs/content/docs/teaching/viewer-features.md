---
title: Viewer Features
description: "Navigation, themes, search, bookmarks, laser pointer, and keyboard shortcuts"
sidebar:
  order: 4
---

The SlideViewer is the client-side runtime that turns rendered HTML into an interactive slide presentation. It is injected by the [`slideviewer` plugin](/plugins/slideviewer/) on pages with the `presentation` layout.

## Navigation

Navigate between slides using the toolbar buttons or keyboard:

| Action | Keyboard | Touch |
|--------|----------|-------|
| Next slide | `Arrow Right`, `Arrow Down`, `Page Down` | Swipe left |
| Previous slide | `Arrow Left`, `Arrow Up`, `Page Up` | Swipe right |
| First slide | `Home` | - |
| Last slide | `End` | - |
| Go to slide N | `G`, then type the number | Tap slide counter |

The slide counter in the toolbar is clickable. Tap it to type a slide number directly.

## Slide overview

Press the grid button or `O` to show a thumbnail overview of all slides. Click any thumbnail to jump to that slide.

## Table of contents

A collapsible sidebar on the left lists every slide by its first heading. The active slide is highlighted. Click any entry to jump to that slide.

The TOC updates automatically when switching between slide mode and reading mode.

## Search

Press `/` to open the search modal. Type a keyword to filter slides by content. Results show the slide number and a text snippet with the match highlighted.

The sidebar also has a quick-filter input that narrows the TOC entries.

## Reading mode

Toggle reading mode with the book icon in the toolbar. Reading mode replaces paginated slides with a single scrollable view of all content, similar to a normal documentation page. A progress bar tracks scroll position.

Reading mode keeps the TOC sidebar, themes, and search functional.

## Themes

Six color themes are available, each with light and dark variants:

| Theme | Key colors |
|-------|------------|
| Default | Neutral grays |
| Dracula | Purple/pink on dark |
| GitHub | Blue accents |
| Discord | Blurple/dark |
| One Dark | Atom editor palette |
| Ocean | Teal/sea green |

Press `T` to cycle through themes. The theme dropdown in the toolbar shows all options. A separate light/dark toggle switches the active theme's variant.

Theme and light/dark preferences are saved to `localStorage` and persist across sessions.

## Bookmarks

Press `B` to bookmark the current slide. A star icon appears on bookmarked slides in both the toolbar and the TOC sidebar. Press `Shift+B` to open the bookmarks panel, which lists all starred slides.

Bookmarks are stored per-URL in `localStorage` and persist across browser sessions.

## Font scaling

The `+`/`-` buttons in the toolbar adjust slide text from 60% to 500% of the base size. Double-click the `-` button to reset to 100%. The setting is saved to `localStorage`.

## Laser pointer

Press `L` to activate the laser pointer overlay. Draw annotations directly on slides:

| Tool | Description |
|------|-------------|
| Freehand | Draw with the cursor |
| Circle | Click and drag |
| Rectangle | Click and drag |
| Line | Click and drag |

Press `S` to open the laser pointer settings modal. Configure color, thickness, fade duration, and reverse-fade behavior. Settings are saved to `localStorage`.

## Fullscreen

Press `F` to toggle fullscreen mode. In standalone mode (slides collection pages), the browser's fullscreen API is used. The viewer exits fullscreen when closing the presentation.

## Keyboard shortcuts

Press `?` to show the full keyboard shortcuts modal.

| Key | Action |
|-----|--------|
| `Arrow Left/Right` | Previous / Next slide |
| `Arrow Up/Down` | Previous / Next slide |
| `Page Up/Down` | Previous / Next slide |
| `Home` / `End` | First / Last slide |
| `G` | Go to slide number |
| `/` | Open search |
| `F` | Toggle fullscreen |
| `L` | Toggle laser pointer |
| `S` | Laser pointer settings |
| `B` | Toggle bookmark on current slide |
| `Shift+B` | Toggle bookmark panel |
| `T` | Cycle theme |
| `?` | Show keyboard shortcuts |
| `Escape` | Close viewer / Close open modal |

## Standalone vs modal mode

The viewer operates in two modes depending on context:

| | Standalone | Modal |
|---|-----------|-------|
| **Used in** | Slides collection pages, `layout: presentation` pages | Docs/courses pages with an optional "Start Slideshow" button |
| **Opens** | Automatically on page load | On button click |
| **Close behavior** | Navigates back to the gallery/section | Hides the overlay |
| **Fullscreen** | Manual (press `F`) | Auto-requested on open |

Standalone mode reads `data-standalone="true"` and `data-exit-url` from the viewer container. These are set by the theme's presentation templates.
