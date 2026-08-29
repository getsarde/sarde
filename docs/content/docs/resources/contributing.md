---
title: Contributing
description: "How to report issues, set up a development environment, and contribute code, themes, or plugins to Sarde"
sidebar:
  order: 4
---

How to report issues, contribute code, and develop themes or plugins for Sarde.

## Reporting issues

File bug reports and feature requests on the GitHub issue tracker. Include:

- Sarde version (`sarde version`)
- Operating system and architecture
- Steps to reproduce the issue
- Expected vs. actual behavior
- Relevant `sarde.yaml` configuration (redact sensitive values)
- Error output from `sarde build --verbose` or `sarde dev --verbose`

For link validation issues, include the output of `sarde check-links`.

## Development setup

### Prerequisites

- Go 1.21 or later
- Git

### Building from source

```bash
git clone https://github.com/getsarde/sarde.git
cd sarde
go build -o sarde ./cmd/sarde
```

### Running tests

```bash
go test ./...
```

### Project layout

```
cmd/sarde/          Entry point
internal/
  cli/              Cobra CLI commands
  config/           Configuration loading and validation
  content/          Content discovery, parsing, markdown rendering
  collection/       Collection registry and auto-detection
  engine/           SiteBuilder orchestrator and page types
  build/            Build pipeline (phases, caching, incremental)
  template/         Go html/template engine and function map
  theme/            Theme loading, tokens, presets
  plugin/           Plugin system and built-in server-side plugins
  server/           Dev server and WebSocket live reload
  asset/            CSS/JS bundling and image optimization
  navigation/       Sidebar trees, prev/next, breadcrumbs
  links/            Link graph, coverage, and validation
embedded/
  theme/            Default theme (go:embed)
  defaults/         Default configuration values
  i18n/             Default UI translation strings
```

## Code contributions

### Branch workflow

1. Fork the repository.
2. Create a feature branch from `main`.
3. Make changes with tests.
4. Run `go test ./...` and confirm all tests pass.
5. Submit a pull request against `main`.

### Code style

- Follow standard Go conventions (`gofmt`, `go vet`).
- Keep functions focused. Prefer small, testable functions over large methods.
- Add tests for new functionality. Bug fixes should include a regression test.
- Update `my-docs/official/changelog.md` with an entry describing the change. This is the internal, per-change development log, distinct from the root `CHANGELOG.md` and the published [Changelog](/resources/changelog/) page, which are kept in sync with each other and are updated by maintainers when preparing a release.

### Commit messages

Use a short summary line (under 72 characters) followed by a blank line and optional body. Prefix with the affected area:

```
fix: preserve custom heading IDs in TOC extraction
feat: add social card auto-generation plugin
perf: reuse taxonomy structures on body-only rebuilds
docs: write search plugin page
```

## Theme development

### Creating a custom theme

1. Run `sarde theme eject` to copy the default theme to `themes/default/`.
2. Modify templates, partials, components, and CSS in the ejected directory.
3. Override specific files without ejecting the entire theme by placing files in `layouts/` (templates and partials) or `layouts/components/` (components). Sarde resolves templates from the most specific source: project overrides take precedence over theme files, which take precedence over embedded defaults.

### Theme tokens

Themes are styled through design tokens, not raw CSS values. Override tokens in `sarde.yaml` under `theme.overrides` or create a `theme.yaml` in the theme directory. See the Theme Tokens reference for the full list of available tokens.

### Presets

A preset is a named set of token overrides. The built-in presets are `default`, `ocean`, `forest`, `rose`, `clean`, `minimal`, `docs`, and `academic`. Custom presets can be defined in `theme.yaml`.

## Plugin development

### Server-side plugins

Server-side plugins are Go code compiled into the Sarde binary. They use four lifecycle hooks:

| Hook | When it runs | Use case |
|------|-------------|----------|
| `ConfigSetup` | After config is loaded | Register template functions, modify config. |
| `ContentLoaded` | After content is discovered | Inspect or modify the page list. |
| `BeforeRender` | Before each page renders | Inject scripts, styles, or page params. |
| `BuildDone` | After all pages are written | Generate output files (sitemaps, feeds, search index). |

Each hook receives a typed context object with access to configuration, pages, and output methods.

### Client-side plugins

Client-side plugins are JavaScript and CSS files declared in the plugin manifest (`internal/plugin/clientplugins/manifest.yaml`). Each entry specifies:

- `description`: One-line summary.
- `inject_when`: Condition for loading (`always`, `is_content_page`, `has_headings`, `has_images`, `has_sidebar`, `has_code_blocks`, `has_prev_next`, `has_updated`).
- `assets`: CSS and JS file paths relative to the `assets/` directory.
- `module`: Whether the JS runs as an ES module.

Client-side plugin configuration is defined in a YAML defaults file (`defaults/<plugin-name>.yaml`) with typed fields (`toggle`, `number`, `color`, `text`).

## Documentation contributions

The documentation site is a Sarde project in the `docs/` directory (dogfooding). Content follows the style rules in `my-docs/sarde-docs/WRITING-GUIDE.md`:

- Imperative voice ("Run...", "Add...", "Set...")
- No "I", "we", "us", "let's", "simply", "just", or marketing fluff
- Tables for configuration options
- Realistic examples (not `foo`/`bar` placeholders)
- Every config key, flag, and path verified against source code

To preview documentation changes:

```bash
cd docs
sarde dev
```
