---
title: External Plugins
description: "Install, enable, and configure declarative plugins packaged outside the Sarde binary"
sidebar:
  order: 2
---

External plugins are packages installed into a project's `plugins/` directory. Each one adds CSS, JavaScript, or templates to the site, described entirely by a `plugin.yaml` manifest. No code runs at build time: Sarde reads the manifest and wires the plugin's assets and templates into the regular build pipeline.

This page covers installing and managing external plugins as a site owner. To build one, read [Authoring External Plugins](/advanced/external-plugin-authoring/). For plugins that require a license, read [Premium Plugins](/plugins/premium-plugins/).

An installed plugin looks like this:

```
plugins/
  cohort-banner/
    plugin.yaml          # manifest (required)
    config.yaml          # default option values (optional)
    assets/              # CSS/JS copied to the build output
      css/cohort-banner.css
      js/cohort-banner.js
    templates/           # partials, components, shortcodes (optional)
```

External plugins differ from the [built-in plugins](/plugins/using-plugins/) that ship inside the `sarde` binary: they are installed per project, updated independently of Sarde releases, and can come from third parties.

## Installing a plugin

```
sarde plugin install <source>
```

Four source forms are supported:

```
sarde plugin install cohort-banner.zip
sarde plugin install ./local-plugins/cohort-banner
sarde plugin install github.com/example-org/grade-sync
sarde plugin install https://example.com/downloads/grade-sync.tar.gz
```

GitHub sources accept a branch and subdirectory: `github.com/example-org/plugin-monorepo/tree/main/plugins/grade-sync`.

The destination directory is always `plugins/<slug>/`, where the slug comes from the plugin's own manifest. The source file or folder name does not matter, and there is no rename flag: license lookup and disabling both key off the slug.

```
$ sarde plugin install ./local-plugins/cohort-banner
Installed plugin "Cohort Banner" 1.0.0 to plugins/cohort-banner/
```

Installing a premium plugin additionally prints license instructions:

```
$ sarde plugin install grade-sync.zip
Installed plugin "Grade Sync" 2.1.0 to plugins/grade-sync/

"Grade Sync" is a premium plugin and needs a license to activate.
Purchase a license at https://example.com/buy/grade-sync
Then install it with: sarde license install <license-file>
License locations checked:
  .sarde/licenses/grade-sync.license
  ~/.sarde/licenses/grade-sync.license
```

## Installing manually

The CLI is optional. Unzipping an archive into `plugins/`, or creating the directory by hand, works the same way. The one hard rule: the directory name must exactly match the `slug` declared in `plugin.yaml`. A mismatch produces a build warning and the plugin is skipped.

Manual installs suit local plugin development and vendoring plugins into version control.

## Managing installed plugins

### Listing

```
sarde plugin list
```

```
SLUG                 VERSION    LICENSE              STATUS
cohort-banner        1.0.0      free                 enabled
grade-sync           2.1.0      premium (no license) enabled
```

The LICENSE column shows `free` for regular plugins and the license state for premium ones. See [Premium Plugins](/plugins/premium-plugins/#checking-license-status) for the possible values.

### Inspecting

```
sarde plugin info <slug>
```

```
$ sarde plugin info grade-sync
Name:         Grade Sync
Slug:         grade-sync
Version:      2.1.0
Description:  Sends quiz results to an external gradebook
Author:       example-author
Homepage:     https://example.com/grade-sync
Premium:      true
License:      licensed
Purchase URL: https://example.com/buy/grade-sync
Injects when: collection courses
Styles:       css/grade-sync.css
Scripts:      js/grade-sync.js
Output:       assets/vendor/grade-sync/
```

### Removing

```
sarde plugin remove <slug>
```

```
$ sarde plugin remove cohort-banner
Removed plugin "cohort-banner"
```

Removal deletes `plugins/<slug>/` only. License files live outside the plugin directory and are kept, so reinstalling a premium plugin later does not require re-licensing. See [Premium Plugins](/plugins/premium-plugins/#licenses-survive-plugin-removal).

## Enabling and disabling plugins

An external plugin is active as soon as it exists in `plugins/`. There is no `plugins.enabled` entry for external plugins, and adding one has no effect.

To turn a plugin off without uninstalling it, add its slug to `plugins.disabled` in `sarde.yaml`:

`sarde.yaml`
```yaml
plugins:
  disabled:
    - cohort-banner
    - reading-progress
```

`plugins.disabled` works for every plugin type: external, built-in server-side, and client-side. Unlike `plugins.enabled`, it never replaces anything; it only turns off the named entries.

:::note
`plugins.enabled` continues to govern built-in and client-side plugins only, and setting it still replaces the whole default list. See [Using Plugins](/plugins/using-plugins/#enabling-and-disabling-plugins) for that behavior.
:::

Unknown names in `plugins.disabled` or `plugins.enabled` fail config validation with an "unknown plugin" error, so typos surface at build time instead of silently doing nothing.

## Configuring a plugin

Options resolve through three layers, later layers winning:

1. Field defaults from the plugin's `blueprint.yaml`
2. The plugin's shipped `config.yaml`
3. `plugins.config.<slug>` in the site's `sarde.yaml`

`sarde.yaml`
```yaml
plugins:
  config:
    grade-sync:
      gradebook_url: "https://lms.example.edu/api"
      retry_count: 3
```

Each plugin documents its own options. Unset options fall back to the plugin's shipped defaults.

### Forcing injection

A plugin's manifest declares when its assets load (a specific layout, a collection, pages with images, and so on). Setting `always: true` in the plugin's config bypasses that condition and injects the assets on every page:

`sarde.yaml`
```yaml
plugins:
  config:
    grade-sync:
      always: true
```

This is the same mechanism the [SlideViewer](/plugins/slideviewer) plugin (itself distributed as an external plugin) uses for its `always` option.

## What external plugins can't do

External plugins handle presentation: assets, templates, and configuration. They cannot:

- Add Markdown syntax or Goldmark extensions
- Transform content or inject pages during the build
- Register dev-server routes
- Run build-time computation (image generation, search indexes, feeds)

Those capabilities require Go code and the built-in plugin lifecycle hooks. See [Writing Plugins](/advanced/writing-plugins/).

## Edge cases

- A plugin whose slug matches a built-in plugin name (`sitemap`, `search`, `reading-progress`, and so on) is rejected: `sarde plugin install` fails with an error, and a manually placed directory produces a build warning and is skipped.
- If two installed plugins ship a template with the same filename, the alphabetically last slug wins and the build emits a warning naming both plugins.
- `sarde dev` watches `plugins/` and runs a full rebuild on any change under it. There is no incremental path for plugin changes.
- Unknown fields in `plugin.yaml` are rejected rather than ignored, so a misspelled manifest key fails loudly instead of silently doing nothing.
- A malformed manifest never breaks the build: that plugin is skipped with a warning and every other plugin still loads.
