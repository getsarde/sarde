---
title: CLI Commands
description: "Reference for every sarde CLI command, its flags, and global options"
sidebar:
  order: 3
---

```
sarde <command> [flags] [project-dir]
```

Every command accepts an optional project directory as the first positional argument. If omitted, the current working directory is used.

## Global flags

These flags are inherited by all commands.

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config`, `-c` | string | `sarde.yaml` | Path to the site config file. |
| `--baseURL` | string | `""` | Override the site base URL. |
| `--drafts`, `-D` | bool | `false` | Include draft content. |
| `--future` | bool | `false` | Include future-dated content. |
| `--verbose`, `-v` | bool | `false` | Enable verbose output. |
| `--quiet`, `-q` | bool | `false` | Suppress non-error output. |

## `build`

Build the static site from `content/` to the output directory.

```
sarde build [flags] [project-dir]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output`, `-o` | string | `""` | Override the output directory. Defaults to [`build.output`](/reference/configuration#build). |
| `--base-path` | string | `""` | Override the URL base path (for subdirectory hosting). |
| `--content` | string | `""` | Override the content directory path. |
| `--strict-i18n` | bool | `false` | Warn on missing translation keys per language. |
| `--format` | string | `pretty` | Output format: `pretty` or `json`. With `json`, the build result (page counts, duration, per-phase timings, warnings) is printed to stdout as a single JSON object and the human-readable summary is suppressed. |

```
sarde build
sarde build --output public --drafts
sarde build /path/to/project
```

A build lock prevents concurrent `sarde build` or `sarde dev` processes from writing to the same output directory. The second process exits with `another sarde process (pid N, ...) is already writing to output directory`. See [Troubleshooting](/resources/troubleshooting#another-sarde-process-is-already-writing) if no such process is running.

## `dev`

Start a local development server with live reload.

```
sarde dev [flags] [project-dir]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--port`, `-p` | int | `0` | Server port. Defaults to [`server.port`](/reference/configuration#server) or 4727. |
| `--host` | string | `""` | Host to bind to. Defaults to `127.0.0.1`. Use `0.0.0.0` for LAN access. |
| `--no-drafts` | bool | `false` | Exclude draft content. By default, drafts are included in dev mode. |
| `--base-path` | string | `""` | Override the URL base path. |
| `--content` | string | `""` | Override the content directory path. |
| `--watch-stdin` | bool | `false` | Exit when stdin closes (for child-process mode). |
| `--theme-dev` | string | `""` | Path to a theme source directory for live-reload during framework development. |
| `--check-syntax` | bool | `false` | Enable syntax checking for unclosed fenced blocks during rebuilds. |

```
sarde dev
sarde dev --port 3000 --host 0.0.0.0
```

Draft and expired content are included by default in dev mode. Pass `--no-drafts` to exclude drafts.

If the requested port is taken, the server tries up to 10 consecutive ports before giving up, and prints the one it bound to. `sarde dev` takes the same output-directory build lock as `sarde build`.

## `check-links`

Run link validation without building the site. Checks internal links and optionally probes external URLs.

```
sarde check-links [flags] [project-dir]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--strict` | bool | `false` | Treat all link issues as errors (exit code 1). |
| `--external` | bool | `false` | Also probe external URLs. |
| `--report` | string | `""` | Report format. `pretty`, `json`, or `github-actions`. |
| `--base-path` | string | `""` | Override the URL base path. |
| `--content` | string | `""` | Override the content directory path. |

```
sarde check-links
sarde check-links --external --report github-actions
```

Aliased as `sarde check` for backward compatibility.

## `check-syntax`

Scan Markdown files for unclosed or mismatched `:::` fenced block tags.

```
sarde check-syntax [flags] [project-dir]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--content` | string | `""` | Override the content directory path. |
| `--format` | string | `"pretty"` | Output format. `pretty` or `json`. |

```
sarde check-syntax
sarde check-syntax --format json
```

Piping Markdown via stdin activates a single-file mode that outputs JSON diagnostics (used for editor/tooling integration).

## `validate`

Validate site configuration and content without rendering or writing output. Runs discovery, parsing, schema validation, and optionally content linting.

```
sarde validate [flags] [project-dir]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--lint` | bool | `true` | Run content lint rules after validation. Pass `--lint=false` to disable. |
| `--strict` | bool | `false` | Exit with code 1 if any warnings exist. |

```
sarde validate
sarde validate --strict --lint=false
```

## `deploy`

Deploy the built site to a hosting provider. Run `sarde build` first.

```
sarde deploy [flags] [project-dir]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--provider` | string | `""` | Override the deploy provider. `github`, `netlify`, `cloudflare`, `vercel`, or `custom`. |
| `--output`, `-o` | string | `""` | Override the output directory. |

```
sarde build
sarde deploy --provider github
```

The output directory must exist. If it does not, an error prompts to run `sarde build` first.

Only the `github` and `custom` providers are implemented. The `netlify`, `cloudflare`, and `vercel` values are accepted but exit with a "not yet implemented" error; use the platform CLI through the `custom` provider instead. See [Deploying](/start-here/deploying/) for working per-platform commands.

## `version`

Print version, Go runtime, and OS/architecture information.

```
sarde version
```

Output:

```
sarde v0.1.0
Go: go1.21.0
OS/Arch: linux/amd64
```

## `update`

Check for and install the latest version from GitHub Releases. For the full update workflow, see [Updating Sarde](/docs/start-here/updating-sarde/).

```
sarde update [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--check` | bool | `false` | Only check for updates without installing. |
| `--yes`, `-y` | bool | `false` | Skip the confirmation prompt. |

```
sarde update
sarde update --check
sarde update --yes
```

Shows the release notes and release page, then prompts for confirmation before installing. When stdin is not a terminal (scripts, CI), the prompt cannot be answered, so `sarde update` fails with an error unless `--yes` is passed.

Releases are cryptographically signed. Every downloaded artifact is verified against the release checksums, and the checksums file itself is verified against an ed25519 signature before the binary is replaced. A release that fails verification is never installed.

If the binary was installed through a package manager, `sarde update` does not replace it. Instead it suggests the matching upgrade command:

| Install method | Suggested command |
|----------------|-------------------|
| Homebrew | `brew upgrade sarde` |
| Scoop | `scoop update sarde` |
| Chocolatey | `choco upgrade sarde` |
| winget | `winget upgrade sarde` |
| System package manager (binary under `/usr/bin` or `/usr/lib`) | your distribution's package manager |

### Passive update notice

`sarde build` and `sarde dev` show a one-line notice when a newer release is available. The lookup runs at most once per 24 hours and caches its result in `~/.sarde/update-check.json`. On `build`, the lookup runs alongside the build and the notice appears after the build summary in the same run. On `dev`, the notice appears at startup from the previous lookup's cache while a background refresh updates it for next time. Each release is announced only once.

The notice and the background lookup are skipped entirely when any of these apply:

- the `CI` environment variable is set
- the `SARDE_NO_UPDATE_CHECK` environment variable is set (permanent opt-out)
- `--quiet` is passed
- stderr is not a terminal
- the binary is a dev build

## `new`

Create new content and project scaffolding.

### `new <collection> <title>`

Create a new content file in the specified collection.

```
sarde new docs "Getting Started"
sarde new blog "My First Post"
```

Creates `content/<collection>/<slugified-title>.md` with frontmatter.

### `new site [path]`

Scaffold a new Sarde project with starter files and example content.

```
sarde new site mysite
sarde new site .
```

Creates the following structure:

```
mysite/
  sarde.yaml
  kazari.config.yaml
  .gitignore
  content/
    _index.md
    blog/
      _index.md
      hello-world.md
    docs/
      _index.md
      getting-started.md
  public/
    .gitkeep
    images/
      hero-light.svg
      hero-dark.svg
```

### `new course <name>`

Scaffold a new course directory under `content/courses/`.

```
sarde new course web-development
```

Creates `content/courses/web-development/` with `config.yaml` and `_index.md`.

### `new lesson <course> <name>`

Create a new auto-numbered lesson inside a course.

```
sarde new lesson web-development "Introduction"
```

Creates `content/courses/web-development/01-introduction.md`. The numeric prefix is automatically incremented based on existing lessons.

### `new directive <name>`

Scaffold a custom `:::` block directive.

```
sarde new directive pullquote
```

Creates `directives/pullquote.yaml` (schema), `directives/pullquote.html` (template), and `directives/pullquote.css` (styles, bundled into the site stylesheet automatically). Names must be lowercase letters, digits, and hyphens starting with a letter; names that collide with a built-in directive are rejected.

See [Custom Directives](/extensions/custom-directives) for the schema and template data.

## `import`

Import content from external sources.

### `import obsidian <vault-path>`

Convert an Obsidian vault to Sarde content. Converts wikilinks, callouts, image embeds, and strips comments.

```
sarde import obsidian ~/Documents/MyVault
sarde import obsidian ~/Documents/MyVault --collection notes
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--collection`, `-c` | string | `""` | Target collection name. Defaults to the vault folder name. |
| `--content` | string | `"content"` | Content directory path. |

Obsidian allows two images in different folders to share a filename, while Sarde copies them into one flat `assets/` directory. When two files share a name but differ in content, the second is saved as `<name>-2.<ext>`, the third as `<name>-3.<ext>`, and so on. Each rename prints a warning naming both source files. Markdown references to that name resolve to the first copy, so check the warnings and repoint any embed that should have used the renamed file.

## `theme`

Manage themes.

### `theme eject`

Copy the embedded default theme to `themes/default/` for customization.

```
sarde theme eject
sarde theme eject --force
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Overwrite an existing `themes/default/` directory. |

### `theme list`

List all available themes: the built-in default theme and any themes installed in `themes/`.

```
sarde theme list
```

The active theme is marked with `*`.

### `theme add <source>`

Install a theme from a GitHub repository, a URL, or a local directory.

```
sarde theme add github.com/user/sarde-theme-ocean
sarde theme add https://example.com/theme.tar.gz
sarde theme add ./my-local-theme
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--name` | string | `""` | Override the theme directory name. |

Supported sources: GitHub repositories, direct zip/tar.gz URLs, and local directories.

### `theme remove <name>`

Remove an installed theme.

```
sarde theme remove ocean
sarde theme remove ocean --force
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Remove even if the theme is currently active. |

The embedded `default` theme cannot be removed.

### `theme info <name>`

Show details about a theme (name, version, author, license, description, token counts, presets).

```
sarde theme info default
sarde theme info ocean
```

### `theme chromastyles`

Generate CSS for a Chroma syntax highlighting style.

```
sarde theme chromastyles --list
sarde theme chromastyles --style monokai --dark
sarde theme chromastyles --style dracula --output styles.css
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--style` | string | `""` | Chroma style name. Defaults to the configured light or dark theme. |
| `--dark` | bool | `false` | Wrap output in `[data-theme="dark"] { }` scoping. |
| `--list` | bool | `false` | List all available Chroma style names. |
| `--output`, `-o` | string | `""` | Write CSS to a file instead of stdout. |

## `icons`

Manage icon sets.

### `icons add <prefix> [prefix...]`

Download Iconify icon sets from the npm registry.

```
sarde icons add mdi
sarde icons add mdi lucide tabler
sarde icons add mdi --dest icons/sets
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dest`, `-d` | string | `""` | Destination directory. Defaults to [`icons.sets_dir`](/reference/configuration#icons) or `icon-sets`. |

After downloading, reference icons as `:icon[prefix:name]` in content.

### `icons list`

List all Iconify icon sets available for download, with local download status.

```
sarde icons list
sarde icons list --search material
sarde icons list --page-size 50
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--search`, `-s` | string | `""` | Filter by prefix, name, or category. |
| `--page-size`, `-n` | int | `30` | Rows per page. Set to 0 to disable pagination. |

Downloaded sets are marked with `*` in the `DL` column.

## `i18n`

Manage internationalization: languages and translation files.

The `i18n add-language`, `i18n remove-language`, `i18n scaffold`, and `i18n status` commands output a single JSON line to stdout instead of human-readable text, for integration with the desktop app.

### `i18n add-language <project-dir> <code>`

Add a language to the project.

```
sarde i18n add-language . fr
sarde i18n add-language . zh-hans --name "Chinese (Simplified)" --weight 2
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--name` | string | `""` | Display name for the language. |
| `--weight` | int | `0` | Sort weight. Lower values sort first. |
| `--dir` | string | `"ltr"` | Text direction. `ltr` or `rtl`. |

The language code must match `[a-z]{2,3}(-[a-z0-9]+)*` (e.g., `fr`, `es`, `zh-hans`). Adds the language to `i18n.languages` in `sarde.yaml`. If this is the first language added, sets `i18n.default_language` to the existing `site.language` value, or `en` if unset.

### `i18n remove-language <project-dir> <code>`

Remove a language from the project.

```
sarde i18n remove-language . fr
sarde i18n remove-language . fr --delete-content
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--delete-content` | bool | `false` | Also delete `content/<code>/`. |

The default language cannot be removed.

### `i18n scaffold <project-dir> <code>`

Scaffold content directories and a translation file for a language.

```
sarde i18n scaffold . fr
```

Creates `content/<code>/<collection>/` for each collection found in the default language's content directory, with a stub `_index.md` in each. Creates `i18n/<code>.yaml`, seeded from the default language's translation file if one exists.

### `i18n status <project-dir>`

Show translation coverage across all languages and collections.

```
sarde i18n status .
```

Reports, per collection and language, the total number of Markdown files in the default language versus the number present in each other language.

## `doc-version`

Manage docs versioning for a collection.

### `doc-version create <project-dir> <collection> <version-id>`

Cut a new version for a collection. Takes a trailing `<label>` argument, the human-readable name shown in the version switcher.

```
sarde doc-version create . docs v1 "Version 1.0"
```

Enables versioning for the collection if not already enabled. Archives the collection's current content into `content/<collection>/<old-version>/` (the previous `last_version`, if any), appends the new version entry, and sets `last_version` to the new version ID. The version ID cannot contain `/`, `\`, or `..`.

### `doc-version delete <project-dir> <collection> <version-id>`

Delete a version from a collection.

```
sarde doc-version delete . docs v1
```

Removes the version entry from `sarde.yaml` and deletes `content/<collection>/<version-id>/`. The current latest version (`last_version`) cannot be deleted.

### `doc-version update <project-dir> <collection> <version-id>`

Update a version entry's label, banner, or redirect behavior.

```
sarde doc-version update . docs v1 --label "Version 1.0 (EOL)" --banner unmaintained
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--label` | string | `""` | New display label. |
| `--banner` | string | `""` | Version banner type. `none`, `unmaintained`, or `unreleased`. |
| `--redirect` | string | `""` | Redirect behavior. `same-page` or `root`. |

Only the flags explicitly passed are updated; omitted flags leave the existing value unchanged.

## `plugin`

Manage [external plugins](/plugins/external-plugins/) installed in the project's `plugins/` directory.

### `plugin install <source>`

Install a plugin from a local zip file, a local directory, a GitHub repository, or a direct zip/tar.gz URL.

```
sarde plugin install cohort-banner.zip
sarde plugin install ./local-plugins/cohort-banner
sarde plugin install github.com/example-org/grade-sync
sarde plugin install https://example.com/downloads/grade-sync.tar.gz
```

GitHub sources accept a branch and subdirectory: `github.com/example-org/plugin-monorepo/tree/main/plugins/grade-sync`.

The destination directory always takes the slug declared in the plugin's manifest; there is no rename flag. Installing a premium plugin prints its purchase URL and the license file locations. Installation fails if the plugin is already installed, the manifest is invalid, or the slug collides with a built-in plugin name.

### `plugin list`

List installed external plugins with version, license status, and enabled state.

```
sarde plugin list
```

```
SLUG                 VERSION    LICENSE              STATUS
cohort-banner        1.0.0      free                 enabled
grade-sync           2.1.0      premium (licensed)   enabled
```

### `plugin info <slug>`

Show a plugin's manifest details: name, version, author, premium status, license state, injection condition, assets, and output prefix.

```
sarde plugin info grade-sync
```

### `plugin remove <slug>`

Delete `plugins/<slug>/` from the project. License files live outside the plugin directory and are kept.

```
sarde plugin remove cohort-banner
```

## `license`

Manage licenses for [premium plugins](/plugins/premium-plugins/).

### `license install <license-file>`

Copy a license file into the license directory and verify it.

```
sarde license install grade-sync.license
sarde license install grade-sync.license --project
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--project` | bool | `false` | Install into the project's `.sarde/licenses/` instead of the user home directory (`~/.sarde/licenses/`). |

A license that fails verification is still installed, with a warning that the plugin stays inactive until a valid license is present.

### `license list`

List installed licenses from both the project and user home locations, with validity status.

```
sarde license list
```

```
PLUGIN               LICENSEE                       EXPIRES      STATUS
grade-sync           jane@school.edu                never        valid
quiz-widgets         jane@school.edu                2026-06-30   license expired on 2026-06-30
```

## `catalog`

Print the frontmatter field catalog: every frontmatter field Sarde recognizes, grouped by category, with the layout-to-category mapping. Used by Sarde Studio to offer field pickers.

```
sarde catalog [--format pretty|json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | `"pretty"` | Output format. `pretty` or `json`. |

```
sarde catalog
sarde catalog --format json
```

The `pretty` format prints fields grouped by category with their type and description, followed by a layout-to-category mapping. The `json` format outputs the raw catalog structure. This command does not take a project directory argument; the catalog is embedded in the binary.

## `directives`

Print the catalog of `:::` block directives Sarde recognizes, grouped by category, with syntax templates and key fields. Used by Sarde Studio to offer a directive picker.

```
sarde directives [project-dir] [--format pretty|json] [--check]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | `"pretty"` | Output format. `pretty` or `json`. |
| `--check` | bool | `false` | Validate directive definitions only. Prints warnings and exits 1 if any are found. |

The built-in directives are embedded in the binary. Any [custom directives](/extensions/custom-directives) in the project's `directives/` folder are merged into the output, and every entry carries a `source` field: `builtin` for the embedded catalog, `site` for the project's own. Theme-provided directives are resolved at build time and are not listed here.

`--check` is a fast lint: it loads and validates the definitions without running a build, so it fits an edit-save loop or a pre-commit hook.

## `effective-config`

Print each content collection's fully resolved configuration, after zero-config inference and any `sarde.yaml` overrides. Every field is labeled with where its value came from: `inferred` from the collection directory's name, or `sarde_yaml` when set explicitly.

```
sarde effective-config [project-dir] [--format pretty|json]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--format` | string | `"pretty"` | Output format. `pretty` or `json`. |

Use it to see what a collection is actually doing before overriding anything, which is the quickest way to answer why a directory sorted or laid itself out the way it did.

```
sarde effective-config
sarde effective-config ./my-site --format json
```

## `sidecar`

Start the IPC API server for the desktop app. This is an internal command used by the Tauri-based desktop application.

```
sarde sidecar [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--port`, `-p` | int | `0` | Server port. 0 auto-assigns an available port. |

Outputs a JSON line to stdout with `ready`, `port`, and `token` fields for the host process.

## `render`

Render Markdown from stdin to HTML. This is an internal command used for editor preview integration.

```
echo "# Hello" | sarde render
```

Outputs JSON with `html` and `headings` fields.

## Environment variables

Environment variables override all other configuration layers. See the [Environment variables](/reference/configuration#environment-variables) section of the Configuration Reference for the full list of supported `SARDE_*` variables.
