---
title: Oi Desktop Quickstart
description: "Create, edit, and preview Sarde content visually with the Oi Desktop app"
sidebar:
  order: 3
---

Oi Desktop is a visual editor for Sarde sites. It wraps the Sarde engine in a native desktop application so content can be created, edited, and previewed without opening a terminal. Available for macOS, Windows, and Linux.

Prefer the command line? See [Getting Started](/start-here/getting-started) for the CLI workflow.

## Download and install

Download the latest release for your platform from [GitHub Releases](https://github.com/getsarde/oi-desktop/releases):

| Platform | File |
|----------|------|
| macOS | `Oi-Desktop-x.x.x.dmg` |
| Windows | `Oi-Desktop-x.x.x.msi` |
| Linux | `Oi-Desktop-x.x.x.AppImage` or `.deb` |

On macOS, drag the app to the Applications folder. On Windows, run the installer and follow the prompts. On Linux, make the AppImage executable (`chmod +x`) and run it, or install the `.deb` package.

Oi Desktop bundles the Sarde engine. No separate installation is required.

## Create a project

Launch Oi Desktop. The welcome screen displays three options: create a new project, open an existing project, or reopen a recent project.

<!-- SCREENSHOT: oi-welcome-screen — welcome screen with Create, Open, and Recent sections -->

To create a new site, click **Create New Project**. The project wizard asks for:

1. A theme mode (docs, blog, or marketing)
2. A site title and description
3. A directory where the project files are stored

→ Oi Desktop scaffolds the project and opens the workspace.

<!-- SCREENSHOT: oi-project-wizard — project creation wizard showing theme mode selection -->

To open an existing Sarde project, click **Open Existing** and select a folder containing a `sarde.yaml` file.

## The workspace

The workspace has three panels:

- **Sidebar** (left): file tree showing content collections, pages, and assets
- **Editor** (center): Markdown editor with tabs for open files
- **Preview** (right): live site preview rendered by the Sarde engine

All panels are resizable and collapsible.

<!-- SCREENSHOT: oi-workspace-overview — three-panel workspace with sidebar, editor, and preview -->

## Create a page

Right-click a collection folder in the sidebar (for example, `docs/`) and select **New Page**. Enter a title. Oi Desktop creates the file with frontmatter pre-filled:

```yaml
---
title: Photosynthesis Overview
---
```

The new page appears in the sidebar and opens in the editor.

→ The sidebar updates to show the new page sorted within its collection.

## Edit content

The editor uses CodeMirror 6 with Markdown syntax highlighting. All Sarde extensions (`:::note`, `:::tabs`, `:::steps`, and others) are highlighted with distinct styling.

Type Markdown content below the frontmatter:

```markdown
---
title: Photosynthesis Overview
---

## How it works

Plants convert light energy into chemical energy through a series
of reactions in the chloroplast.

:::note
This process requires both water and carbon dioxide.
:::
```

Save with **Ctrl+S** (Windows/Linux) or **Cmd+S** (macOS). The preview panel updates automatically.

→ The preview shows the rendered page with the note aside styled as a blue callout.

<!-- SCREENSHOT: oi-editor-preview — editor with Markdown source on the left and rendered preview on the right -->

## Edit frontmatter

Click the frontmatter bar above the editor to expand the visual form. Fields render as typed inputs: text fields for title and description, a date picker for dates, a tag input with autocomplete for tags, and toggles for boolean fields like `draft`.

```yaml
---
title: Photosynthesis Overview
description: How plants convert light to energy
tags: [biology, plants]
draft: false
---
```

Toggle **Raw YAML** to switch between the form view and a YAML text editor. Changes sync between both views.

→ The frontmatter form shows each field with its label, type-appropriate input widget, and validation indicators.

<!-- SCREENSHOT: oi-frontmatter-form — visual frontmatter editor with title, tags, and draft fields -->

## Preview the site

The preview panel on the right renders the full site using the Sarde engine. Navigate between pages by clicking links in the preview, or use the sidebar to switch files.

The preview updates on every save. CSS changes hot-swap without a full page reload.

Resize the preview panel by dragging its left edge, or collapse it entirely to focus on writing.

## Search content

Press **Ctrl+Shift+F** (Windows/Linux) or **Cmd+Shift+F** (macOS) to search across all content files in the project. Results display the file path and matching line. Click a result to open the file in a new editor tab.

## Keyboard shortcuts

| Shortcut | Action |
|----------|--------|
| Ctrl/Cmd+S | Save current file |
| Ctrl/Cmd+Shift+P | Open command palette |
| Ctrl/Cmd+Shift+F | Search across files |
| Ctrl/Cmd+W | Close current tab |
| Ctrl/Cmd+B | Toggle sidebar |

The command palette provides quick access to all actions. Type to filter, then press Enter to execute.

## Build and deploy

To build the site for production, open the command palette (**Ctrl/Cmd+Shift+P**) and run **Build Site**. Oi Desktop runs `sarde build` and displays progress, warnings, and errors in the activity log panel at the bottom of the workspace.

The output is written to the `dist/` directory, ready for deployment. See [Deploying](/start-here/deploying) for platform-specific deployment steps.

## Next steps

- [Getting Started](/start-here/getting-started) for the CLI workflow
- [Content & Collections](/guides/content-and-collections) to understand how collections work
- [Configuration](/reference/configuration) to customize `sarde.yaml`
