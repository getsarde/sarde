# Sarde

A zero-config static site generator written in Go. Drop Markdown into `content/`, get a production-ready site.

## Why Sarde

Most static site generators make you choose: Hugo gives you speed and a single binary but demands configuration and template boilerplate. Docusaurus and Starlight give you great defaults for docs sites but require Node.js and a build toolchain.

Sarde tries to give you both. It ships as a single Go binary with no runtime dependencies. Your directory structure is your site structure. Put files in `content/blog/` and you get a date-sorted blog with RSS. Put files in `content/docs/` and you get a docs site with sidebar navigation, versioning, and table of contents. No config file needed for either.

When you do need control, everything is overridable through a single `sarde.yaml` file, CLI flags, or environment variables.

## Features

**Content**
- Auto-detected collections: directories named `blog`, `posts`, `articles`, `news` get date-sorted feed layouts; `docs`, `guides`, `courses`, `tutorials` get docs layouts with sidebar and ToC
- Frontmatter in YAML, TOML, or JSON
- Title, date, and weight inferred automatically from filenames, headings, and git history
- Page bundles (directory with `index.md` + images) with automatic responsive image generation
- Versioned docs (Docusaurus-style) with URL shadowing for the latest version
- i18n with per-language directories, RTL support, and translation fallback

**Markdown**
- 24+ Goldmark extensions: code blocks with syntax highlighting (Chroma and Shiki), KaTeX math, Mermaid diagrams, GitHub-style alerts, callouts, cards, tabs, file trees, image comparison, keyboard shortcuts, spoilers, and more
- Two syntax highlighting engines: Chroma (Go-native, fast) and Shiki (400+ grammars, vendored)

**Asset pipeline**
- CSS/JS bundling via esbuild (Go API, no Node.js)
- Image processing: resize, crop, WebP/AVIF conversion, LQIP blur-up placeholders
- Content-hash fingerprinting for cache busting
- HTML/CSS/JS minification

**Built-in plugins** (13 enabled by default)
- `sitemap`, `robots`, `rss`, `atom` for standard web outputs
- `seo` for Open Graph and Twitter Card meta tags
- `social_cards` for server-side OG image generation
- `search` for offline full-text search (Orama)
- `link_validator` for internal/external broken link detection
- `content_lint` for heading structure, image alt text, and frontmatter validation
- `katex`, `mermaid` for math and diagram asset injection
- `redirects` for redirect stubs (HTML or Netlify `_redirects`)
- `llms_txt` for LLM-friendly site index

**Developer experience**
- Dev server with WebSocket live reload and incremental rebuilds
- Link checking with terminal, JSON, and GitHub Actions annotation output
- Content validation without building (`sarde validate`)
- Obsidian vault importer (converts wikilinks and callouts)
- Deploy command for GitHub Pages, Netlify, Cloudflare Pages, and Vercel

## Installation

**Homebrew (macOS/Linux):**

```bash
brew install getsarde/sarde/sarde
```

**Shell script (macOS/Linux):**

```bash
curl -sSfL https://raw.githubusercontent.com/getsarde/sarde/main/install.sh | sh
```

**Binary download:**

Grab the latest release from [GitHub Releases](https://github.com/getsarde/sarde/releases). Available for Linux (amd64), macOS (amd64, arm64), and Windows (amd64).

**From source:**

```bash
go install github.com/getsarde/sarde/cmd/sarde@latest
```

## Quick start

```bash
# Create a new site
sarde new site my-site
cd my-site

# Add some content
mkdir -p content/docs
echo '---
title: Getting Started
---

# Hello

This is your first page.' > content/docs/getting-started.md

# Start the dev server
sarde dev
```

Open `http://localhost:4727`. The `docs/` directory is auto-detected as a docs collection, so you get a sidebar, table of contents, and prev/next navigation out of the box.

To build for production:

```bash
sarde build
```

Output goes to `dist/` by default.

## Project structure

```
my-site/
  content/          # Your Markdown files (required)
    blog/           # Auto-detected as blog collection
    docs/           # Auto-detected as docs collection
    _index.md       # Homepage content (optional)
  static/           # Copied as-is to output
  icons/            # Local SVG icons
  themes/           # Custom themes (optional)
  sarde.yaml        # Configuration (optional)
```

Collections are detected by directory name:

| Directory names | Type | Default sort | Layout |
|---|---|---|---|
| `blog`, `posts`, `articles`, `news` | Blog | date descending | Feed with pagination |
| `docs`, `guides`, `courses`, `tutorials` | Docs | weight ascending | Three-column with sidebar + ToC |
| Anything else | Generic | title ascending | Default |

## Configuration

All configuration goes in `sarde.yaml`. Every option has a sensible default; the file is entirely optional. Values are resolved in a 5-layer cascade (last wins):

1. Embedded defaults (compiled into the binary)
2. `theme.yaml` from the active theme
3. `sarde.yaml` in your project root
4. CLI flags (`--baseURL`, `--drafts`, `--future`)
5. Environment variables (`SARDE_SITE_TITLE`, `SARDE_BUILD_OUTPUT`, etc.)

Here is a minimal configuration:

```yaml
site:
  title: My Project
  url: https://example.com

theme:
  dark: true
```

And a more complete example:

```yaml
site:
  title: My Project
  url: https://example.com
  description: Project documentation and blog
  edit_url: https://github.com/user/repo/edit/main/content

theme:
  preset: docs
  dark: true

build:
  minify: true
  last_updated: git    # "git", "mtime", or "false"

collections:
  docs:
    versioning:
      enabled: true
      last_version: v2
      versions:
        - id: v2
          label: v2.x (latest)
        - id: v1
          label: v1.x

i18n:
  default_language: en
  languages:
    en:
      name: English
    fr:
      name: Français

plugins:
  enabled:
    - sitemap
    - rss
    - search
    - seo
    - link_validator
```

See the [default configuration](sarde/embedded/defaults/sarde.yaml) for every available option and its default value.

## Commands

```bash
sarde build                        # Build for production
sarde dev                          # Dev server with live reload (port 4727)
sarde new site <path>              # Scaffold a new project
sarde new course <name>            # Create a course directory
sarde new lesson <course> <name>   # Add an auto-numbered lesson
sarde check-links                  # Validate links without building
sarde validate                     # Validate config and content
sarde deploy                       # Deploy to configured provider
sarde theme list|add|remove|eject  # Manage themes
sarde icons add|list               # Download/list Iconify icon sets
sarde import obsidian <vault>      # Import an Obsidian vault
sarde version                      # Print version info
```

Global flags: `--config/-c`, `--baseURL`, `--drafts/-D`, `--future`, `--verbose/-v`, `--quiet/-q`

## Architecture

Three layers:

1. **Interface**: CLI (Cobra) and Desktop App (Tauri+Svelte) both call into a unified ProjectManager API
2. **Engine**: Six-phase pipeline: Initialize, Discover, Parse (parallel), Assemble, Assets, Render (parallel), Write
3. **Plugins**: Four lifecycle hooks run in order: ConfigSetup (serial), ContentLoaded (serial), BeforeRender (serial per page), BuildDone (parallel)

The engine is designed around Go interfaces (`ContentDiscoverer`, `FrontmatterParser`, `MarkdownRenderer`, `TemplateEngine`) so each pipeline stage is independently testable.

## Building from source

```bash
cd sarde

# Development build
go build -o ../dist/sarde ./cmd/sarde

# With version tag
go build -ldflags "-X github.com/getsarde/sarde/internal/version.Version=1.0.0" -o ../dist/sarde ./cmd/sarde

# Cross-compile
GOOS=linux   GOARCH=amd64 go build -o ../dist/sarde-linux ./cmd/sarde
GOOS=darwin  GOARCH=arm64 go build -o ../dist/sarde-macos ./cmd/sarde

# Tests
go test ./...

# Benchmarks
go test -bench=. -benchmem -timeout 300s ./internal/build/
```

On Windows, `build.bat` wraps these commands: `build.bat build`, `build.bat test`, `build.bat release`.

## Tech stack

- **Core**: Go, Goldmark, Cobra, Chroma v2, esbuild (Go API), fsnotify, go:embed
- **Desktop app**: Tauri v2 (Rust) + Svelte 5 + CodeMirror 6
- **Generated sites**: Pure HTML/CSS with ~1KB inline JS, no framework runtime
- **Search**: Orama (offline, embedded in output)
- **Minification**: tdewolff/minify (HTML), esbuild (CSS/JS)

## Contributing

Contributions are welcome. Please open an issue to discuss significant changes before submitting a pull request.

```bash
# Clone and build
git clone https://github.com/getsarde/sarde.git
cd sarde/sarde
go build ./cmd/sarde

# Run tests
go test ./...

# Test against the example site
cd ../testsite/general
../../sarde/sarde dev
```

## License

MIT
